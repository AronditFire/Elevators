package domain

import "time"

type Role string

const (
	UserRole  Role = "user"
	AdminRole Role = "admin"
)

type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Role         Role      `json:"role"`
	Disabled     bool      `json:"disabled"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type State string

const (
	Ready   State = "ready"
	Running State = "running"
	Paused  State = "paused"
	Stopped State = "stopped"
)

type Person struct {
	ID          int      `json:"id"`
	From        int      `json:"from"`
	To          int      `json:"to"`
	Status      string   `json:"status"`
	ElevatorID  int      `json:"elevator_id"`
	CalledAt    float64  `json:"called_at"`
	BoardedAt   *float64 `json:"boarded_at"`
	FinishedAt  *float64 `json:"finished_at"`
	automaticGo bool
}

type Elevator struct {
	ID          int            `json:"id"`
	Config      ElevatorConfig `json:"config"`
	Position    float64        `json:"position"`
	Speed       float64        `json:"speed"`
	Moving      bool           `json:"moving"`
	DoorsOpen   bool           `json:"doors_open"`
	Direction   int            `json:"direction"`
	Route       []int          `json:"route"`
	Passengers  []int          `json:"passengers"`
	Trips       int            `json:"trips"`
	EmptyTrips  int            `json:"empty_trips"`
	Transported int            `json:"transported"`
	tripLoaded  bool
}

type Statistics struct {
	Trips        int     `json:"trips"`
	Transported  int     `json:"transported"`
	WaitCount    int     `json:"wait_count"`
	WaitSum      float64 `json:"wait_sum"`
	WaitMax      float64 `json:"wait_max"`
	WaitAverage  float64 `json:"wait_average"`
	FireCount    int     `json:"fire_count"`
	FireDuration float64 `json:"fire_duration"`
}

type Session struct {
	ID             string      `json:"id"`
	OwnerID        string      `json:"owner_id"`
	Config         Config      `json:"config"`
	Scenario       Scenario    `json:"scenario"`
	State          State       `json:"state"`
	Started        bool        `json:"started"`
	Time           float64     `json:"time"`
	Speed          float64     `json:"speed"`
	Fire           bool        `json:"fire"`
	Elevators      []*Elevator `json:"elevators"`
	People         []*Person   `json:"people"`
	Statistics     Statistics  `json:"statistics"`
	EventIndex     int         `json:"event_index"`
	LastEventError string      `json:"last_event_error,omitempty"`
	nextPersonID   int
}

type SessionSummary struct {
	ID      string  `json:"id"`
	OwnerID string  `json:"owner_id"`
	State   State   `json:"state"`
	Time    float64 `json:"time"`
	Fire    bool    `json:"fire"`
}
