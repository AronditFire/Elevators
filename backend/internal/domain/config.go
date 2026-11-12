package domain

import (
	"fmt"
	"math"
)

const FloorHeight = 3.0
const MaxPeople = 10000

type Strategy string

const (
	MinimumWait  Strategy = "minimum_wait"
	MinimumEmpty Strategy = "minimum_empty"
)

type ElevatorConfig struct {
	Capacity     int     `json:"capacity" yaml:"capacity"`
	MaxSpeed     float64 `json:"max_speed" yaml:"max_speed"`
	Acceleration float64 `json:"acceleration" yaml:"acceleration"`
	InitialFloor int     `json:"initial_floor" yaml:"initial_floor"`
}

type Config struct {
	Floors    int              `json:"floors" yaml:"floors"`
	Strategy  Strategy         `json:"strategy" yaml:"strategy"`
	Elevators []ElevatorConfig `json:"elevators" yaml:"elevators"`
}

type PeopleInput struct {
	From  int `json:"from" yaml:"from"`
	To    int `json:"to" yaml:"to"`
	Count int `json:"count" yaml:"count"`
}

type ScenarioEvent struct {
	At          float64 `json:"at" yaml:"at"`
	Type        string  `json:"type" yaml:"type"`
	PeopleInput `yaml:",inline"`
}

type Scenario struct {
	Events []ScenarioEvent `json:"events" yaml:"events"`
}

func finite(n float64) bool { return !math.IsNaN(n) && !math.IsInf(n, 0) }

func (c Config) Validate() error {
	if c.Floors < 1 || c.Floors > 20 {
		return fmt.Errorf("число этажей должно быть от 1 до 20")
	}
	if c.Strategy != MinimumWait && c.Strategy != MinimumEmpty {
		return fmt.Errorf("неизвестная стратегия")
	}
	if len(c.Elevators) < 1 || len(c.Elevators) > 5 {
		return fmt.Errorf("число лифтов должно быть от 1 до 5")
	}
	for i, e := range c.Elevators {
		if e.Capacity < 1 || e.Capacity > 100 {
			return fmt.Errorf("лифт %d: вместимость от 1 до 100", i+1)
		}
		if !finite(e.MaxSpeed) || e.MaxSpeed <= 0 || e.MaxSpeed > 20 {
			return fmt.Errorf("лифт %d: скорость больше 0 и не больше 20 м/с", i+1)
		}
		if !finite(e.Acceleration) || e.Acceleration <= 0 || e.Acceleration > 20 {
			return fmt.Errorf("лифт %d: ускорение больше 0 и не больше 20 м/с²", i+1)
		}
		if e.InitialFloor < 1 || e.InitialFloor > c.Floors {
			return fmt.Errorf("лифт %d: начальный этаж вне здания", i+1)
		}
	}
	return nil
}

func (p PeopleInput) Validate(floors int) error {
	if p.From < 1 || p.From > floors || p.To < 1 || p.To > floors || p.From == p.To {
		return fmt.Errorf("укажите разные этажи внутри здания")
	}
	if p.Count < 1 || p.Count > 1000 {
		return fmt.Errorf("число людей должно быть от 1 до 1000")
	}
	return nil
}

func (s Scenario) Validate(floors int) error {
	if len(s.Events) > 1000 {
		return fmt.Errorf("в сценарии не более 1000 событий")
	}
	last := -1.0
	total := 0
	fire := false
	for i, e := range s.Events {
		if !finite(e.At) || e.At < 0 || e.At > 86400 || e.At < last {
			return fmt.Errorf("событие %d: время от 0 до 86400, события по возрастанию времени", i+1)
		}
		last = e.At
		switch e.Type {
		case "people":
			if err := e.PeopleInput.Validate(floors); err != nil {
				return fmt.Errorf("событие %d: %w", i+1, err)
			}
			total += e.Count
		case "fire_on":
			if fire {
				return fmt.Errorf("событие %d: тревога уже включена", i+1)
			}
			fire = true
		case "fire_off":
			if !fire {
				return fmt.Errorf("событие %d: тревога ещё не включена", i+1)
			}
			fire = false
		default:
			return fmt.Errorf("событие %d: неизвестный тип", i+1)
		}
	}
	if total > MaxPeople {
		return fmt.Errorf("в сценарии не более %d людей", MaxPeople)
	}
	return nil
}
