package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/ramanmalinouski/Elevators/backend/internal/domain"
	"github.com/ramanmalinouski/Elevators/backend/internal/services"
)

type testAuth struct{ users map[string]domain.User }

func (a *testAuth) Authenticate(_ context.Context, token string) (domain.User, error) {
	if u, ok := a.users[token]; ok && !u.Disabled {
		return u, nil
	}
	return domain.User{}, services.ErrUnauthorized
}
func (a *testAuth) Register(context.Context, string, string) (domain.User, error) {
	return domain.User{}, services.ErrForbidden
}
func (a *testAuth) Login(context.Context, string, string) (domain.User, string, error) {
	return domain.User{}, "", services.ErrUnauthorized
}
func (a *testAuth) Logout(context.Context, string) error { return nil }
func (a *testAuth) Users(context.Context, domain.User) ([]domain.User, error) {
	return nil, services.ErrForbidden
}
func (a *testAuth) UpdateUser(context.Context, domain.User, string, domain.Role, bool) (domain.User, error) {
	return domain.User{}, services.ErrForbidden
}

func TestHTTPAccessConfigCommandsAndReport(t *testing.T) {
	auth := &testAuth{users: map[string]domain.User{"owner": {ID: "owner", Role: domain.UserRole}, "other": {ID: "other", Role: domain.UserRole}, "admin": {ID: "admin", Role: domain.AdminRole}}}
	sim := services.NewSimulation()
	s, err := sim.Create(auth.users["owner"], domain.Config{Floors: 3, Strategy: domain.MinimumWait, Elevators: []domain.ElevatorConfig{{Capacity: 2, MaxSpeed: 2, Acceleration: 1, InitialFloor: 1}}}, domain.Scenario{})
	if err != nil {
		t.Fatal(err)
	}
	handler := New(auth, sim, false, func(context.Context) error { return nil }).Handler()
	request := func(method, path, token, origin, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://localhost"+path, bytes.NewBufferString(body))
		if token != "" {
			r.AddCookie(&http.Cookie{Name: "elevators_session", Value: token})
		}
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	path := "/api/sessions/" + s.ID
	checks := []struct {
		method, path, token, origin, body string
		status                            int
	}{
		{"GET", path, "", "", "", 401},
		{"GET", path, "other", "", "", 403},
		{"GET", path, "admin", "", "", 200},
		{"POST", path + "/commands", "owner", "https://evil.example", `{"type":"start"}`, 403},
		{"POST", path + "/commands", "owner", "http://localhost", `{"type":"start","unknown":1}`, 400},
		{"PUT", path + "/config.yaml", "owner", "", "floors: 3\nstrategy: minimum_wait\nelevators: []\nunknown: 1", 400},
		{"POST", path + "/commands", "owner", "", `{"type":"start"}`, 200},
		{"PUT", path + "/config.yaml", "owner", "", "floors: 3\nstrategy: minimum_wait\nelevators:\n  - capacity: 2\n    max_speed: 2\n    acceleration: 1\n    initial_floor: 1", 400},
		{"GET", path + "/report.csv", "owner", "", "", 400},
		{"POST", path + "/commands", "owner", "", `{"type":"stop"}`, 200},
		{"GET", path + "/report.csv", "owner", "", "", 200},
	}
	for _, check := range checks {
		w := request(check.method, check.path, check.token, check.origin, check.body)
		if w.Code != check.status {
			t.Fatalf("%s %s => %d: %s", check.method, check.path, w.Code, w.Body.String())
		}
	}
	w := request("GET", path+"/report.csv", "owner", "", "")
	if !strings.Contains(w.Body.String(), "Поездок всего") || !strings.Contains(w.Header().Get("Content-Disposition"), "statistics.csv") {
		t.Fatal("CSV export missing")
	}
}

func TestWebSocketRejectsUnauthenticatedHandshake(t *testing.T) {
	handler := New(&testAuth{users: map[string]domain.User{}}, services.NewSimulation(), false, func(context.Context) error { return nil }).Handler()
	server := httptest.NewServer(handler)
	defer server.Close()
	r, err := http.Get(server.URL + "/api/sessions/missing/stream")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	_, _ = io.ReadAll(r.Body)
	if r.StatusCode != 401 {
		t.Fatal("stream bypasses authentication")
	}
}

func TestWebSocketStreamsModelSnapshots(t *testing.T) {
	u := domain.User{ID: "owner", Role: domain.UserRole}
	sim := services.NewSimulation()
	s, err := sim.Create(u, domain.Config{Floors: 3, Strategy: domain.MinimumWait, Elevators: []domain.ElevatorConfig{{Capacity: 2, MaxSpeed: 2, Acceleration: 1, InitialFloor: 1}}}, domain.Scenario{})
	if err != nil {
		t.Fatal(err)
	}
	handler := New(&testAuth{users: map[string]domain.User{"owner": u}}, sim, false, func(context.Context) error { return nil }).Handler()
	server := httptest.NewServer(handler)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/api/sessions/"+s.ID+"/stream", &websocket.DialOptions{HTTPHeader: http.Header{"Cookie": []string{"elevators_session=owner"}}})
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	_, data, err := c.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var snapshot domain.Session
	if err = json.Unmarshal(data, &snapshot); err != nil || snapshot.ID != s.ID || snapshot.State != domain.Ready {
		t.Fatal("invalid initial stream snapshot")
	}
	_, err = sim.Execute(u, s.ID, services.Command{Type: "start"})
	if err != nil {
		t.Fatal(err)
	}
	_, data, err = c.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(data, &snapshot); err != nil || snapshot.State != domain.Running {
		t.Fatal("stream does not update after command")
	}
}
