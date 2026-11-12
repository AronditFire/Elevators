package domain

import (
	"fmt"
	"math"
	"slices"
)

func NewSession(id, owner string, config Config, scenario Scenario) (*Session, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if err := scenario.Validate(config.Floors); err != nil {
		return nil, err
	}
	config.Elevators = slices.Clone(config.Elevators)
	scenario.Events = slices.Clone(scenario.Events)
	s := &Session{ID: id, OwnerID: owner, Config: config, Scenario: scenario, State: Ready, Speed: 1, People: []*Person{}}
	if s.Scenario.Events == nil {
		s.Scenario.Events = []ScenarioEvent{}
	}
	for i, c := range config.Elevators {
		s.Elevators = append(s.Elevators, &Elevator{ID: i + 1, Config: c, Position: floorPosition(c.InitialFloor), DoorsOpen: true, Route: []int{}, Passengers: []int{}})
	}
	return s, nil
}

// Configure разрешён только до первого старта: изменение физики и стратегии не должно менять условия уже начатого сеанса.
func (s *Session) Configure(config Config, scenario Scenario) error {
	if s.Started {
		return fmt.Errorf("после первого старта конфигурацию и сценарий менять нельзя")
	}
	if len(s.People) > 0 {
		return fmt.Errorf("создайте новый сеанс: в текущем уже есть люди")
	}
	next, err := NewSession(s.ID, s.OwnerID, config, scenario)
	if err != nil {
		return err
	}
	*s = *next
	return nil
}

func (s *Session) AddPeople(input PeopleInput) error {
	return s.addPeople(input, false)
}

func (s *Session) addPeople(input PeopleInput, automaticGo bool) error {
	if err := input.Validate(s.Config.Floors); err != nil {
		return err
	}
	if len(s.People)+input.Count > MaxPeople {
		return fmt.Errorf("в сеансе не более %d активных людей", MaxPeople)
	}
	for range input.Count {
		s.nextPersonID++
		// Создание включает немедленное нажатие пульта; ожидание отсчитывается от этого модельного времени.
		p := &Person{ID: s.nextPersonID, From: input.From, To: input.To, CalledAt: s.Time, Status: "waiting", automaticGo: automaticGo}
		if s.Fire {
			p.Status = "evacuating"
			t := s.Time
			p.FinishedAt = &t
		}
		s.People = append(s.People, p)
	}
	s.dispatch()
	return nil
}

// Start возобновляет тот же сеанс: время, позиции, статистика и курсор сценария не сбрасываются.
func (s *Session) Start() error {
	if s.State == Running {
		return fmt.Errorf("симуляция уже запущена")
	}
	s.State, s.Started = Running, true
	s.processEvents()
	s.dispatch()
	return nil
}

func (s *Session) Pause() error {
	if s.State != Running {
		return fmt.Errorf("пауза доступна только при работающей симуляции")
	}
	s.State = Paused
	return nil
}

// Stop проверяет только кабины. Люди в очередях остаются в модели и не запрещают остановку.
func (s *Session) Stop() error {
	for _, e := range s.Elevators {
		if e.Moving || len(e.Passengers) > 0 {
			return fmt.Errorf("остановка разрешена, только если все лифты пустые и остановлены")
		}
	}
	s.State = Stopped
	s.updateStatistics()
	return nil
}

func (s *Session) SetSpeed(speed float64) error {
	if !finite(speed) || speed < .25 || speed > 16 {
		return fmt.Errorf("темп должен быть от 0.25 до 16")
	}
	s.Speed = speed
	return nil
}

// Go вручную запускает неподвижную кабину с маршрутом; пожарный спуск запускается отдельно и автоматически.
func (s *Session) Go(id int) error {
	if s.State != Running {
		return fmt.Errorf("«Ход» доступен в работающей симуляции")
	}
	if s.Fire {
		return fmt.Errorf("при тревоге лифты управляются автоматически")
	}
	if id < 1 || id > len(s.Elevators) {
		return fmt.Errorf("лифт не найден")
	}
	e := s.Elevators[id-1]
	if e.Moving {
		return fmt.Errorf("лифт уже движется")
	}
	s.dispatch()
	if len(e.Route) == 0 {
		return fmt.Errorf("у лифта нет маршрута")
	}
	e.Moving, e.DoorsOpen = true, false
	return nil
}

func (s *Session) Advance(dt float64) {
	// Таймеры людей, сценария и тревоги используют единое модельное время; пауза и остановка замораживают их вместе.
	if s.State != Running || !finite(dt) || dt <= 0 {
		return
	}
	end := s.Time + dt
	for s.Time < end-1e-9 {
		s.processEvents()
		// Шаг ограничен 20 мс и временем ближайшего события, чтобы не перескочить момент запуска сценария.
		next := math.Min(end, s.Time+.02)
		if s.EventIndex < len(s.Scenario.Events) {
			next = math.Min(next, s.Scenario.Events[s.EventIndex].At)
		}
		step := next - s.Time
		if step <= 1e-9 {
			s.processEvents()
			continue
		}
		if s.Fire {
			s.Statistics.FireDuration += step
		}
		for _, e := range s.Elevators {
			s.move(e, step)
		}
		s.Time = next
		s.People = slices.DeleteFunc(s.People, func(p *Person) bool { return p.FinishedAt != nil && s.Time-*p.FinishedAt >= 3-1e-9 })
		s.dispatch()
	}
	s.processEvents()
	s.updateStatistics()
}

func (s *Session) move(e *Elevator, dt float64) {
	if !e.Moving {
		return
	}
	offset := 0.0
	for dt > 1e-9 && len(e.Route) > 0 {
		floor := s.nextFloor(e)
		d := floorPosition(floor) - e.Position
		e.beginDirection(sign(d))
		eta := travelTime(d, e.Speed, e.Config.MaxSpeed, e.Config.Acceleration)
		if eta <= dt+1e-9 {
			e.Position, e.Speed = floorPosition(floor), 0
			offset += eta
			// Посадка фиксируется в вычисленный момент прибытия внутри шага, а не в конце общего тика.
			s.arriveAt(e, floor, s.Time+offset)
			dt -= eta
		} else {
			distance, speed := travelDistance(dt, e.Speed, e.Config.MaxSpeed, e.Config.Acceleration)
			e.Position += float64(sign(d)) * distance
			e.Speed = speed
			dt = 0
		}
	}
	if len(e.Route) == 0 {
		e.Moving, e.DoorsOpen, e.Speed = false, true, 0
	}
}

func (s *Session) arrive(e *Elevator, floor int) { s.arriveAt(e, floor, s.Time) }

// На остановке сначала освобождаем места высадкой, затем сажаем назначенных ожидающих; обмен пассажирами мгновенный.
func (s *Session) arriveAt(e *Elevator, floor int, now float64) {
	e.Speed = 0
	e.Route = slices.DeleteFunc(e.Route, func(f int) bool { return f == floor })
	e.DoorsOpen = true
	for _, p := range s.People {
		if p.ElevatorID != e.ID {
			continue
		}
		if p.Status == "riding" || p.Status == "evacuating_in_elevator" {
			// Отсчёт трёх секунд эвакуации пассажира кабины начинается только после прибытия на первый этаж.
			if (s.Fire || p.Status == "evacuating_in_elevator") && floor == 1 {
				p.Status = "evacuating"
			} else if !s.Fire && p.Status == "riding" && p.To == floor {
				p.Status = "delivered"
				e.Transported++
			} else {
				continue
			}
			t := now
			p.FinishedAt = &t
			e.Passengers = slices.DeleteFunc(e.Passengers, func(id int) bool { return id == p.ID })
		}
	}
	if !s.Fire && !s.evacuationPending(e) {
		for _, p := range s.People {
			if p.Status != "waiting" || p.ElevatorID != e.ID || p.From != floor {
				continue
			}
			if len(e.Passengers) >= e.Config.Capacity {
				p.ElevatorID = 0
				continue
			}
			p.Status = "riding"
			t := now
			p.BoardedAt = &t
			e.Passengers = append(e.Passengers, p.ID)
			// minimum_wait сохраняет очередь прежних высадок перед новой целью.
			// minimum_empty затем выбирает остановку по текущему направлению в nextFloor.
			addStop(e, p.To, false)
			// Ожидание завершается при посадке; дальнейшая поездка не входит в среднее, максимум и сумму ожиданий.
			wait := now - p.CalledAt
			s.Statistics.WaitCount++
			s.Statistics.WaitSum += wait
			s.Statistics.WaitMax = math.Max(s.Statistics.WaitMax, wait)
		}
	}
	if len(e.Route) > 0 && e.Moving {
		e.DoorsOpen = false
	}
}

func (s *Session) processEvents() {
	for s.EventIndex < len(s.Scenario.Events) && s.Scenario.Events[s.EventIndex].At <= s.Time+1e-9 {
		e := s.Scenario.Events[s.EventIndex]
		// Событие выполняется один раз даже при отказе: ошибка видна в UI, последующие события не блокируются.
		s.EventIndex++
		var err error
		switch e.Type {
		case "people":
			err = s.addPeople(e.PeopleInput, true)
		case "fire_on":
			err = s.SetFire(true)
		case "fire_off":
			err = s.SetFire(false)
		}
		if err != nil {
			s.LastEventError = fmt.Sprintf("событие %d: %v", s.EventIndex, err)
		}
	}
}

func (s *Session) updateStatistics() {
	s.Statistics.Trips, s.Statistics.Transported = 0, 0
	for _, e := range s.Elevators {
		s.Statistics.Trips += e.Trips
		s.Statistics.Transported += e.Transported
	}
	if s.Statistics.WaitCount > 0 {
		s.Statistics.WaitAverage = s.Statistics.WaitSum / float64(s.Statistics.WaitCount)
	}
}
