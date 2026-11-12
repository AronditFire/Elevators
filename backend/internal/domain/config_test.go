package domain

import (
	"math"
	"testing"
)

func TestRejectInvalidConfigurationAndScenarios(t *testing.T) {
	s := session(t, MinimumWait, 1)
	for _, floors := range []int{0, 21} {
		cfg := s.Config
		cfg.Floors = floors
		if cfg.Validate() == nil {
			t.Fatalf("accepted %d floors", floors)
		}
	}
	for _, speed := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		cfg := s.Config
		cfg.Elevators = append([]ElevatorConfig{}, cfg.Elevators...)
		cfg.Elevators[0].MaxSpeed = speed
		if cfg.Validate() == nil {
			t.Fatalf("accepted speed %v", speed)
		}
	}
	for _, scenario := range []Scenario{
		{Events: []ScenarioEvent{{At: -1, Type: "fire_on"}}},
		{Events: []ScenarioEvent{{At: 1, Type: "fire_off"}}},
		{Events: []ScenarioEvent{{At: 2, Type: "fire_on"}, {At: 1, Type: "fire_off"}}},
		{Events: []ScenarioEvent{{At: 3, Type: "people", PeopleInput: PeopleInput{From: 1, To: 1, Count: 1}}}},
	} {
		if scenario.Validate(10) == nil {
			t.Fatalf("accepted scenario %+v", scenario)
		}
	}
}

func TestPeopleCanBeAddedWithFrozenTime(t *testing.T) {
	s := session(t, MinimumWait, 1)
	must(t, s.AddPeople(PeopleInput{From: 1, To: 5, Count: 1}))
	if s.People[0].Status != "waiting" {
		t.Fatal("boarded before start")
	}
	must(t, s.Start())
	must(t, s.Pause())
	must(t, s.AddPeople(PeopleInput{From: 2, To: 5, Count: 1}))
	now := s.Time
	s.Advance(3)
	if s.Time != now || len(s.People) != 2 {
		t.Fatal("pause rejects creation or advances time")
	}
}
