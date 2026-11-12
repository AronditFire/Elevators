package domain

import "math"

func addStop(e *Elevator, floor int, front bool) {
	for i, f := range e.Route {
		if f != floor {
			continue
		}
		if front {
			e.Route = append(e.Route[:i], e.Route[i+1:]...)
			break
		}
		return
	}
	if front {
		e.Route = append([]int{floor}, e.Route...)
	} else {
		e.Route = append(e.Route, floor)
	}
}

func (s *Session) evacuationPending(e *Elevator) bool {
	for _, p := range s.People {
		if p.ElevatorID == e.ID && p.Status == "evacuating_in_elevator" {
			return true
		}
	}
	return false
}

func (s *Session) selectElevator(p *Person) *Elevator {
	var best *Elevator
	bestETA := math.Inf(1)
	for _, e := range s.Elevators {
		if s.evacuationPending(e) {
			continue
		}
		// Учитываем и пассажиров, и уже назначенных ожидающих, чтобы не зарезервировать одно место нескольким вызовам.
		reserved := len(e.Passengers)
		for _, other := range s.People {
			if other.Status == "waiting" && other.ElevatorID == e.ID {
				reserved++
			}
		}
		if reserved >= e.Config.Capacity {
			continue
		}
		distance := floorPosition(p.From) - e.Position
		initialSpeed := e.Speed
		if sign(distance) != e.Direction {
			initialSpeed = 0
		}
		// Для minimum_wait оцениваем прямое время до посадки; задержку до будущего ручного «Ход» предсказать нельзя.
		eta := travelTime(distance, initialSpeed, e.Config.MaxSpeed, e.Config.Acceleration)
		// Для minimum_empty время до вызова учитывает обслуживание целей по текущему направлению.
		if s.Config.Strategy == MinimumEmpty && len(e.Route) > 0 {
			position, speed := e.Position, e.Speed
			goals := append(append([]int{}, e.Route...), p.From)
			direction := e.Direction
			eta = 0
			for len(goals) > 0 {
				i := nearestStop(goals, position, direction)
				floor := goals[i]
				d := floorPosition(floor) - position
				if sign(d) != direction {
					speed = 0
				}
				eta += travelTime(d, speed, e.Config.MaxSpeed, e.Config.Acceleration)
				if floor == p.From {
					break
				}
				direction, position, speed = sign(d), floorPosition(floor), 0
				goals = append(goals[:i], goals[i+1:]...)
			}
		}
		if best == nil || eta < bestETA {
			best, bestETA = e, eta
		}
	}
	return best
}

// Ближайшая цель впереди имеет приоритет. Разворот допустим только после обслуживания всех целей по направлению.
func nearestStop(route []int, position float64, direction int) int {
	best, distance := -1, math.Inf(1)
	for i, f := range route {
		d := floorPosition(f) - position
		if direction != 0 && sign(d) != 0 && sign(d) != direction {
			continue
		}
		if math.Abs(d) < distance {
			best, distance = i, math.Abs(d)
		}
	}
	if best >= 0 {
		return best
	}
	return nearestStop(route, position, 0)
}

func (s *Session) nextFloor(e *Elevator) int {
	if s.Config.Strategy == MinimumEmpty && !s.Fire && !s.evacuationPending(e) {
		i := nearestStop(e.Route, e.Position, e.Direction)
		e.Route[0], e.Route[i] = e.Route[i], e.Route[0]
	}
	return e.Route[0]
}

func (s *Session) dispatch() {
	if s.Fire {
		return
	}
	for _, p := range s.People {
		if p.Status != "waiting" || p.ElevatorID != 0 {
			continue
		}
		if e := s.selectElevator(p); e != nil {
			p.ElevatorID = e.ID
			// minimum_wait ставит новую посадку первой; minimum_empty выбирает остановку по направлению в nextFloor.
			addStop(e, p.From, s.Config.Strategy == MinimumWait)
		}
	}
	// Маршруты назначаются в любом состоянии, посадка и автоматический «Ход» — только при running.
	if s.State != Running {
		return
	}
	for _, e := range s.Elevators {
		if len(e.Route) > 0 && sign(floorPosition(s.nextFloor(e))-e.Position) == 0 {
			s.arrive(e, e.Route[0])
		}
		if !e.Moving && len(e.Route) > 0 && s.hasAutomaticCall(e) {
			e.Moving, e.DoorsOpen = true, false
		}
	}
}

func (s *Session) hasAutomaticCall(e *Elevator) bool {
	// Сценарий имитирует «Ход» и при подаче пустой кабины, и после посадки.
	// Признак хранится у человека, чтобы сработать и после ожидания свободного места;
	// доставленные и эвакуируемые люди не запускают последующие ручные поездки.
	for _, p := range s.People {
		if p.automaticGo && p.ElevatorID == e.ID && (p.Status == "waiting" || p.Status == "riding") {
			return true
		}
	}
	return false
}
