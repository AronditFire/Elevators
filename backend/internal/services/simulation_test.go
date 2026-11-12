package services

import (
	"errors"
	"sync"
	"testing"

	"github.com/ramanmalinouski/Elevators/backend/internal/domain"
)

func TestSessionAccessAndSnapshotIsolation(t *testing.T) {
	m := NewSimulation()
	owner := domain.User{ID: "owner", Role: domain.UserRole}
	other := domain.User{ID: "other", Role: domain.UserRole}
	admin := domain.User{ID: "admin", Role: domain.AdminRole}
	cfg := domain.Config{Floors: 10, Strategy: domain.MinimumWait, Elevators: []domain.ElevatorConfig{{Capacity: 6, MaxSpeed: 2, Acceleration: 1, InitialFloor: 1}}}
	s, err := m.Create(owner, cfg, domain.Scenario{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = m.Get(other, s.ID); !errors.Is(err, ErrForbidden) {
		t.Fatal("foreign read allowed")
	}
	if len(m.List(other)) != 0 || len(m.List(admin)) != 1 {
		t.Fatal("listing leaks or hides sessions")
	}
	if _, err = m.Execute(other, s.ID, Command{Type: "start"}); !errors.Is(err, ErrForbidden) {
		t.Fatal("foreign command allowed")
	}
	if _, err = m.Execute(admin, s.ID, Command{Type: "start"}); err != nil {
		t.Fatal(err)
	}
	s.Config.Elevators[0].Capacity = 99
	s.Elevators[0].Position = 99
	current, err := m.Get(owner, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Config.Elevators[0].Capacity != 6 || current.Elevators[0].Position != 0 {
		t.Fatal("snapshot mutates model")
	}
	if _, err = m.Create(owner, cfg, domain.Scenario{}); err == nil {
		t.Fatal("more than one own session")
	}
	if err = m.Delete(owner, s.ID); err == nil {
		t.Fatal("deleted running session")
	}
}

func TestConcurrentCommandsReadsAndClock(t *testing.T) {
	m := NewSimulation()
	owner := domain.User{ID: "owner", Role: domain.UserRole}
	s, err := m.Create(owner, domain.Config{Floors: 3, Strategy: domain.MinimumEmpty, Elevators: []domain.ElevatorConfig{{Capacity: 2, MaxSpeed: 2, Acceleration: 1, InitialFloor: 1}}}, domain.Scenario{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.Execute(owner, s.ID, Command{Type: "start"})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 40 {
				m.Advance(.02)
				_, _ = m.Get(owner, s.ID)
				_ = m.List(owner)
				_, _ = m.Execute(owner, s.ID, Command{Type: "speed", Speed: 2})
			}
		}()
	}
	wg.Wait()
}
