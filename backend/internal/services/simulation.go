package services

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/ramanmalinouski/Elevators/backend/internal/domain"
	"github.com/ramanmalinouski/Elevators/backend/pkg/security"
)

// Simulation сериализует команды и тики: доменная модель работает без собственных блокировок и зависимости от HTTP.
type Simulation struct {
	mu       sync.RWMutex
	sessions map[string]*domain.Session
}

type Command struct {
	Type       string             `json:"type"`
	ElevatorID int                `json:"elevator_id,omitempty"`
	Speed      float64            `json:"speed,omitempty"`
	Active     bool               `json:"active,omitempty"`
	People     domain.PeopleInput `json:"people,omitempty"`
}

func NewSimulation() *Simulation { return &Simulation{sessions: map[string]*domain.Session{}} }

// Владелец управляет своим сеансом; администратор может вмешиваться в любой, заблокированный аккаунт доступа не имеет.
func allowed(actor domain.User, s *domain.Session) bool {
	return !actor.Disabled && (actor.Role == domain.AdminRole || actor.ID == s.OwnerID)
}

// Наружу отдаём отдельный снимок, чтобы HTTP/WebSocket и вызывающий код не меняли живую модель после снятия блокировки.
func clone(s *domain.Session) domain.Session {
	b, _ := json.Marshal(s)
	var out domain.Session
	_ = json.Unmarshal(b, &out)
	return out
}

func (m *Simulation) Create(actor domain.User, config domain.Config, scenario domain.Scenario) (domain.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.sessions {
		if s.OwnerID == actor.ID {
			return domain.Session{}, fmt.Errorf("у вас уже есть сеанс; сначала удалите его")
		}
	}
	if len(m.sessions) >= 100 {
		return domain.Session{}, fmt.Errorf("достигнут лимит 100 активных сеансов")
	}
	id, err := security.NewToken()
	if err != nil {
		return domain.Session{}, err
	}
	s, err := domain.NewSession(id[:24], actor.ID, config, scenario)
	if err != nil {
		return domain.Session{}, err
	}
	m.sessions[s.ID] = s
	return clone(s), nil
}

func (m *Simulation) List(actor domain.User) []domain.SessionSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []domain.SessionSummary{}
	for _, s := range m.sessions {
		if allowed(actor, s) {
			out = append(out, domain.SessionSummary{ID: s.ID, OwnerID: s.OwnerID, State: s.State, Time: s.Time, Fire: s.Fire})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (m *Simulation) get(actor domain.User, id string) (*domain.Session, error) {
	s := m.sessions[id]
	if s == nil {
		return nil, ErrNotFound
	}
	if !allowed(actor, s) {
		return nil, ErrForbidden
	}
	return s, nil
}

func (m *Simulation) Get(actor domain.User, id string) (domain.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, err := m.get(actor, id)
	if err != nil {
		return domain.Session{}, err
	}
	return clone(s), nil
}

func (m *Simulation) Configure(actor domain.User, id string, config domain.Config, scenario domain.Scenario) (domain.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, err := m.get(actor, id)
	if err != nil {
		return domain.Session{}, err
	}
	if err = s.Configure(config, scenario); err != nil {
		return domain.Session{}, err
	}
	return clone(s), nil
}

func (m *Simulation) Delete(actor domain.User, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, err := m.get(actor, id)
	if err != nil {
		return err
	}
	if s.State == domain.Running || s.State == domain.Paused {
		return fmt.Errorf("сначала остановите сеанс")
	}
	delete(m.sessions, id)
	return nil
}

func (m *Simulation) Execute(actor domain.User, id string, cmd Command) (domain.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, err := m.get(actor, id)
	if err != nil {
		return domain.Session{}, err
	}
	switch cmd.Type {
	case "start":
		err = s.Start()
	case "pause":
		err = s.Pause()
	case "stop":
		err = s.Stop()
	case "speed":
		err = s.SetSpeed(cmd.Speed)
	case "go":
		err = s.Go(cmd.ElevatorID)
	case "fire":
		err = s.SetFire(cmd.Active)
	case "people":
		err = s.AddPeople(cmd.People)
	default:
		err = fmt.Errorf("неизвестная команда")
	}
	if err != nil {
		return domain.Session{}, err
	}
	return clone(s), nil
}

func (m *Simulation) Advance(dt float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.sessions {
		// Темп меняет скорость модельных часов, но не физические ограничения кабин.
		s.Advance(dt * s.Speed)
	}
}
