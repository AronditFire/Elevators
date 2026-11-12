package domain

import "fmt"

func (s *Session) SetFire(active bool) error {
	if active && s.State == Paused {
		return fmt.Errorf("нельзя включить тревогу на паузе")
	}
	if s.Fire == active {
		return fmt.Errorf("состояние тревоги уже установлено")
	}
	s.Fire = active
	if active {
		// Обычные вызовы отменяются: на этажах сразу эвакуация, пассажиры кабин сначала спускаются на первый этаж.
		s.Statistics.FireCount++
		for _, p := range s.People {
			if p.Status == "riding" {
				p.Status = "evacuating_in_elevator"
			}
			if p.Status == "waiting" {
				p.Status, p.ElevatorID = "evacuating", 0
				t := s.Time
				p.FinishedAt = &t
			}
		}
		for _, e := range s.Elevators {
			// Пожарный маршрут не содержит промежуточных остановок и не требует ручного «Ход».
			e.Route = []int{1}
			if sign(e.Position) == 0 {
				s.arrive(e, 1)
				e.Moving, e.Speed, e.DoorsOpen = false, 0, true
			} else {
				e.Moving, e.DoorsOpen = true, false
			}
		}
	} else {
		for _, e := range s.Elevators {
			// Раннее отключение тревоги не бросает кабину между этажами: начатый спуск завершается на первом.
			if s.evacuationPending(e) || e.Moving {
				e.Route = []int{1}
				e.Moving = true
				continue
			}
			e.Route = []int{}
			e.Moving, e.Speed, e.DoorsOpen = false, 0, true
		}
		s.dispatch()
	}
	return nil
}
