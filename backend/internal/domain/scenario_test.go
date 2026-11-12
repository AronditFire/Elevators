package domain

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestExampleScenarioRunsWithoutManualGo(t *testing.T) {
	for _, strategy := range []Strategy{MinimumWait, MinimumEmpty} {
		t.Run(string(strategy), func(t *testing.T) {
			var config Config
			var scenario Scenario
			for name, target := range map[string]any{"config": &config, "scenario": &scenario} {
				data, err := os.ReadFile("../../../examples/" + name + ".yaml")
				must(t, err)
				must(t, yaml.Unmarshal(data, target))
			}
			config.Strategy = strategy
			s, err := NewSession("test", "owner", config, scenario)
			must(t, err)
			must(t, s.Start())
			s.Advance(3)
			if !s.Elevators[0].Moving || !s.Elevators[1].Moving {
				t.Fatal("scenario passengers must automatically press Go, including an empty pickup trip")
			}
			s.Advance(17)
			if s.EventIndex != 4 || s.Fire {
				t.Fatalf("scenario did not reach the post-fire group: event=%d fire=%v", s.EventIndex, s.Fire)
			}
			s.Advance(40)
			if s.Statistics.Transported != 3 || len(s.People) != 0 {
				t.Fatalf("post-fire passengers remain stranded: stats=%+v people=%+v", s.Statistics, s.People)
			}
		})
	}
}

func TestScenarioGroupLargerThanCapacityNeedsNoManualGo(t *testing.T) {
	s := session(t, MinimumWait, 1)
	s.Elevators[0].Config.Capacity = 1
	s.Scenario.Events = []ScenarioEvent{{At: 0, Type: "people", PeopleInput: PeopleInput{From: 3, To: 6, Count: 3}}}
	must(t, s.Start())
	s.Advance(60)
	if s.Statistics.Transported != 3 || len(s.People) != 0 {
		t.Fatalf("later passengers were not collected automatically: %+v", s.Statistics)
	}
}

func TestManualCallStartsAutomaticallyAfterScenario(t *testing.T) {
	s := session(t, MinimumWait, 1)
	s.Scenario.Events = []ScenarioEvent{{At: 0, Type: "people", PeopleInput: PeopleInput{From: 1, To: 3, Count: 1}}}
	must(t, s.Start())
	s.Advance(4)
	must(t, s.AddPeople(PeopleInput{From: 3, To: 5, Count: 1}))
	s.Advance(5)
	if s.Statistics.Transported != 2 || s.Elevators[0].Position != floorPosition(5) {
		t.Fatal("manual call after the scenario requires intervention")
	}
}

func TestScenarioAutomaticGoRespectsPauseAndResume(t *testing.T) {
	s := session(t, MinimumWait, 1)
	s.Scenario.Events = []ScenarioEvent{{At: 1, Type: "people", PeopleInput: PeopleInput{From: 3, To: 6, Count: 1}}}
	must(t, s.Start())
	s.Advance(2)
	must(t, s.Pause())
	position, now := s.Elevators[0].Position, s.Time
	s.Advance(20)
	if s.Time != now || s.Elevators[0].Position != position {
		t.Fatal("automatic Go bypassed the pause")
	}
	must(t, s.Start())
	s.Advance(20)
	if s.Statistics.Transported != 1 {
		t.Fatal("automatic scenario did not resume")
	}
}
