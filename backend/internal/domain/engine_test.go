package domain

import (
	"math"
	"testing"
)

func session(t *testing.T, strategy Strategy, initial int) *Session {
	t.Helper()
	s, err := NewSession("test", "owner", Config{Floors: 10, Strategy: strategy, Elevators: []ElevatorConfig{{Capacity: 6, MaxSpeed: 3, Acceleration: 1, InitialFloor: initial}}}, Scenario{})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestGoPauseResumeAndStop(t *testing.T) {
	s := session(t, MinimumWait, 1)
	must(t, s.AddPeople(PeopleInput{From: 1, To: 10, Count: 1}))
	must(t, s.Start())
	s.Advance(5)
	if s.Elevators[0].Moving || s.Elevators[0].Position != 0 {
		t.Fatal("stationary elevator moved without Go")
	}
	if s.Stop() == nil {
		t.Fatal("occupied elevator may not stop system")
	}
	must(t, s.Go(1))
	s.Advance(1)
	position, now := s.Elevators[0].Position, s.Time
	must(t, s.Pause())
	s.Advance(20)
	if s.Time != now || s.Elevators[0].Position != position {
		t.Fatal("pause advances model")
	}
	if s.SetFire(true) == nil {
		t.Fatal("alarm allowed while paused")
	}
	must(t, s.Start())
	s.Advance(20)
	must(t, s.Stop())
	if s.Statistics.Transported != 1 || s.Statistics.WaitSum != 0 {
		t.Fatalf("statistics: %+v", s.Statistics)
	}
	now = s.Time
	must(t, s.Start())
	s.Advance(1)
	if math.Abs(s.Time-now-1) > 1e-8 {
		t.Fatal("start resets model time")
	}
	if s.Configure(s.Config, s.Scenario) == nil {
		t.Fatal("config changed after start")
	}
}

func TestMinimumWaitKeepsOldDeliveryBeforeNewDestination(t *testing.T) {
	s := session(t, MinimumWait, 1)
	must(t, s.Start())
	must(t, s.AddPeople(PeopleInput{From: 1, To: 10, Count: 1}))
	must(t, s.Go(1))
	s.Advance(1)
	must(t, s.AddPeople(PeopleInput{From: 6, To: 2, Count: 1}))
	if s.Elevators[0].Route[0] != 6 {
		t.Fatal("pickup did not interrupt route")
	}
	for s.People[1].Status == "waiting" {
		s.Advance(.1)
	}
	route := s.Elevators[0].Route
	if len(route) != 2 || route[0] != 10 || route[1] != 2 {
		t.Fatalf("old delivery must remain first: %v", route)
	}
	s.Advance(30)
	if s.Statistics.Transported != 2 || s.Elevators[0].Trips != 2 {
		t.Fatalf("unexpected final stats: %+v / %+v", s.Statistics, s.Elevators[0])
	}
}

func TestMinimumEmptyServesAheadThenBehind(t *testing.T) {
	s := session(t, MinimumEmpty, 6)
	must(t, s.Start())
	must(t, s.AddPeople(PeopleInput{From: 6, To: 10, Count: 1}))
	must(t, s.Go(1))
	s.Advance(1)
	must(t, s.AddPeople(PeopleInput{From: 1, To: 2, Count: 1}))
	s.Advance(.1)
	if s.Elevators[0].Route[0] != 10 {
		t.Fatal("reversed before forward destination")
	}
	s.Advance(35)
	if s.Statistics.Transported != 2 {
		t.Fatalf("not delivered: %+v", s.People)
	}
}

func TestEmptyAndLoadedSegmentsAreOneTrip(t *testing.T) {
	s := session(t, MinimumEmpty, 1)
	must(t, s.Start())
	must(t, s.AddPeople(PeopleInput{From: 6, To: 10, Count: 1}))
	must(t, s.Go(1))
	s.Advance(30)
	e := s.Elevators[0]
	if e.Trips != 1 || e.EmptyTrips != 0 {
		t.Fatalf("empty and loaded segments counted separately: %+v", e)
	}
	must(t, s.AddPeople(PeopleInput{From: 10, To: 1, Count: 1}))
	must(t, s.Go(1))
	s.Advance(30)
	if e.Trips != 2 {
		t.Fatal("reversal after idle not counted")
	}
}

func TestFireEvacuatesCabinOnlyAfterFirstFloor(t *testing.T) {
	s := session(t, MinimumWait, 10)
	must(t, s.Start())
	must(t, s.AddPeople(PeopleInput{From: 10, To: 8, Count: 1}))
	must(t, s.AddPeople(PeopleInput{From: 4, To: 7, Count: 1}))
	must(t, s.SetFire(true))
	s.Advance(3.1)
	if len(s.People) != 1 || s.People[0].Status != "evacuating_in_elevator" {
		t.Fatal("cabin evacuated before first floor or floor people not removed")
	}
	s.Advance(13)
	if len(s.People) != 0 || s.Elevators[0].Position != 0 || !s.Elevators[0].DoorsOpen {
		t.Fatal("fire did not evacuate everyone")
	}
	must(t, s.AddPeople(PeopleInput{From: 1, To: 2, Count: 1}))
	s.Advance(3.1)
	if len(s.People) != 0 {
		t.Fatal("new people during alarm stay in building")
	}
	must(t, s.SetFire(false))
	if s.Fire || s.Statistics.FireCount != 1 || math.Abs(s.Statistics.FireDuration-19.2) > .001 {
		t.Fatalf("fire stats: %+v", s.Statistics)
	}
}

func TestEarlyFireOffStillEvacuatesCabin(t *testing.T) {
	s := session(t, MinimumWait, 10)
	must(t, s.Start())
	must(t, s.AddPeople(PeopleInput{From: 10, To: 6, Count: 1}))
	must(t, s.SetFire(true))
	s.Advance(1)
	must(t, s.SetFire(false))
	s.Advance(15)
	if len(s.People) != 0 || len(s.Elevators[0].Passengers) != 0 || s.Elevators[0].Position != 0 {
		t.Fatal("early fire-off strands cabin")
	}
}

func TestScenarioUsesModelTimeAndPreservesCursor(t *testing.T) {
	s := session(t, MinimumWait, 1)
	s.Scenario = Scenario{Events: []ScenarioEvent{{At: 3, Type: "people", PeopleInput: PeopleInput{From: 2, To: 9, Count: 1}}, {At: 7, Type: "fire_on"}, {At: 15, Type: "fire_off"}}}
	must(t, s.Start())
	s.Advance(2)
	must(t, s.Pause())
	s.Advance(10)
	if s.EventIndex != 0 {
		t.Fatal("scenario runs during pause")
	}
	must(t, s.Start())
	s.Advance(14)
	if s.EventIndex != 3 || s.Statistics.FireCount != 1 || math.Abs(s.Statistics.FireDuration-8) > .001 {
		t.Fatalf("events/stats: %d %+v", s.EventIndex, s.Statistics)
	}
	must(t, s.Stop())
	must(t, s.Start())
	s.Advance(1)
	if s.EventIndex != 3 || s.Statistics.FireCount != 1 {
		t.Fatal("replayed scenario after restart")
	}
}

func TestCapacityAndWaitingAtStop(t *testing.T) {
	s := session(t, MinimumWait, 1)
	s.Elevators[0].Config.Capacity = 1
	must(t, s.AddPeople(PeopleInput{From: 2, To: 8, Count: 2}))
	must(t, s.Stop())
	if len(s.People) != 2 {
		t.Fatal("stop removed waiting people")
	}
	must(t, s.Start())
	must(t, s.Go(1))
	s.Advance(20)
	if s.Statistics.Transported != 1 {
		t.Fatalf("capacity violated: %+v", s.Statistics)
	}
	must(t, s.Go(1))
	s.Advance(25)
	if s.Statistics.Transported != 2 || s.Statistics.WaitCount != 2 {
		t.Fatal("unassigned passenger lost")
	}
}

func TestMotionMatchesAnalyticTime(t *testing.T) {
	s := session(t, MinimumWait, 1)
	must(t, s.Start())
	must(t, s.AddPeople(PeopleInput{From: 1, To: 10, Count: 1}))
	must(t, s.Go(1))
	timeToFloor := travelTime(27, 0, 3, 1)
	s.Advance(timeToFloor - .1)
	if s.People[0].Status != "riding" {
		t.Fatal("arrived early")
	}
	s.Advance(.2)
	if s.People[0].FinishedAt == nil || math.Abs(*s.People[0].FinishedAt-timeToFloor) > 1e-7 {
		t.Fatal("arrival not timed exactly")
	}
}
