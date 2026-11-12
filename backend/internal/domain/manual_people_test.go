package domain

import "testing"

func TestManualCreationCallsAndTransportsPeopleWithoutIntervention(t *testing.T) {
	for _, strategy := range []Strategy{MinimumWait, MinimumEmpty} {
		for _, from := range []int{1, 4} {
			t.Run(string(strategy)+"/from="+string(rune('0'+from)), func(t *testing.T) {
				s := session(t, strategy, 1)
				s.Elevators[0].Config.Capacity = 2
				must(t, s.Start())
				s.Advance(7.25)
				must(t, s.AddPeople(PeopleInput{From: from, To: 9, Count: 5}))
				for _, p := range s.People {
					if p.CalledAt != 7.25 || p.To != 9 {
						t.Fatal("person did not immediately call the destination floor")
					}
				}
				if !s.Elevators[0].Moving {
					t.Fatal("empty pickup or departure requires manual Go")
				}
				s.Advance(150)
				if s.Statistics.Transported != 5 || s.Statistics.WaitCount != 5 || len(s.People) != 0 {
					t.Fatalf("manual group was not fully served: %+v", s.Statistics)
				}
			})
		}
	}
}

func TestManualCreationDuringFrozenStatesWaitsForStart(t *testing.T) {
	for _, state := range []State{Ready, Paused, Stopped} {
		t.Run(string(state), func(t *testing.T) {
			s := session(t, MinimumWait, 1)
			if state != Ready {
				must(t, s.Start())
				s.Advance(2)
			}
			if state == Paused {
				must(t, s.Pause())
			}
			if state == Stopped {
				must(t, s.Stop())
			}
			now := s.Time
			must(t, s.AddPeople(PeopleInput{From: 3, To: 7, Count: 1}))
			s.Advance(10)
			if s.Time != now || s.People[0].CalledAt != now || s.Elevators[0].Moving {
				t.Fatal("creation must record the call without advancing a frozen simulation")
			}
			must(t, s.Start())
			s.Advance(30)
			if s.Statistics.Transported != 1 {
				t.Fatal("passenger not delivered after resuming")
			}
		})
	}
}

func TestManualCreationAfterFireNeedsNoGo(t *testing.T) {
	s := session(t, MinimumWait, 1)
	must(t, s.Start())
	must(t, s.SetFire(true))
	s.Advance(3)
	must(t, s.SetFire(false))
	must(t, s.AddPeople(PeopleInput{From: 6, To: 2, Count: 3}))
	s.Advance(30)
	if s.Statistics.Transported != 3 {
		t.Fatal("manual post-fire group remains stranded")
	}
}
