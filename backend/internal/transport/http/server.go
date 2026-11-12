package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/ramanmalinouski/Elevators/backend/internal/domain"
	"github.com/ramanmalinouski/Elevators/backend/internal/services"
	"github.com/ramanmalinouski/Elevators/backend/pkg/files"
)

type AuthService interface {
	Register(context.Context, string, string) (domain.User, error)
	Login(context.Context, string, string) (domain.User, string, error)
	Authenticate(context.Context, string) (domain.User, error)
	Logout(context.Context, string) error
	Users(context.Context, domain.User) ([]domain.User, error)
	UpdateUser(context.Context, domain.User, string, domain.Role, bool) (domain.User, error)
}

type SimulationService interface {
	Create(domain.User, domain.Config, domain.Scenario) (domain.Session, error)
	List(domain.User) []domain.SessionSummary
	Get(domain.User, string) (domain.Session, error)
	Configure(domain.User, string, domain.Config, domain.Scenario) (domain.Session, error)
	Delete(domain.User, string) error
	Execute(domain.User, string, services.Command) (domain.Session, error)
}

type Server struct {
	auth     AuthService
	sim      SimulationService
	secure   bool
	health   func(context.Context) error
	mu       sync.Mutex
	attempts map[string]attempt
}
type attempt struct {
	since time.Time
	count int
}

func New(auth AuthService, sim SimulationService, secure bool, health func(context.Context) error) *Server {
	return &Server{auth: auth, sim: sim, secure: secure, health: health, attempts: map[string]attempt{}}
}

func (s *Server) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /api/health", s.healthcheck)
	m.HandleFunc("POST /api/auth/register", s.register)
	m.HandleFunc("POST /api/auth/login", s.login)
	m.HandleFunc("POST /api/auth/logout", s.logout)
	m.HandleFunc("GET /api/auth/me", s.me)
	m.HandleFunc("GET /api/users", s.users)
	m.HandleFunc("PATCH /api/users/{id}", s.updateUser)
	m.HandleFunc("GET /api/sessions", s.list)
	m.HandleFunc("POST /api/sessions", s.create)
	m.HandleFunc("GET /api/sessions/{id}", s.get)
	m.HandleFunc("PUT /api/sessions/{id}", s.configure)
	m.HandleFunc("DELETE /api/sessions/{id}", s.delete)
	m.HandleFunc("POST /api/sessions/{id}/commands", s.command)
	m.HandleFunc("GET /api/sessions/{id}/config.yaml", s.configYAML)
	m.HandleFunc("PUT /api/sessions/{id}/config.yaml", s.configYAML)
	m.HandleFunc("GET /api/sessions/{id}/scenario.yaml", s.scenarioYAML)
	m.HandleFunc("PUT /api/sessions/{id}/scenario.yaml", s.scenarioYAML)
	m.HandleFunc("GET /api/sessions/{id}/report.csv", s.report)
	m.HandleFunc("GET /api/sessions/{id}/stream", s.stream)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		defer func() {
			if v := recover(); v != nil {
				slog.Error("http panic", "error", v)
				http.Error(w, "внутренняя ошибка", 500)
			}
		}()
		if r.Method != "GET" && r.Method != "HEAD" {
			if origin := r.Header.Get("Origin"); origin != "" {
				u, err := url.Parse(origin)
				if err != nil || u.Host != r.Host || (u.Scheme != "http" && u.Scheme != "https") {
					failure(w, services.ErrForbidden)
					return
				}
			}
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		m.ServeHTTP(w, r)
	})
}

func respond(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func failure(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	switch {
	case errors.Is(err, services.ErrUnauthorized):
		status = 401
	case errors.Is(err, services.ErrForbidden):
		status = 403
	case errors.Is(err, services.ErrNotFound):
		status = 404
	case errors.Is(err, services.ErrConflict):
		status = 409
	case errors.Is(err, services.ErrInternal):
		slog.Error("request failed", "error", err)
		respond(w, 500, map[string]string{"error": "внутренняя ошибка сервера"})
		return
	}
	respond(w, status, map[string]string{"error": err.Error()})
}

func decode(r *http.Request, target any) error {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return fmt.Errorf("ошибка JSON: %w", err)
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return fmt.Errorf("ожидается один JSON-объект")
	}
	return nil
}

func (s *Server) actor(w http.ResponseWriter, r *http.Request) (domain.User, bool) {
	cookie, err := r.Cookie("elevators_session")
	if err != nil {
		failure(w, services.ErrUnauthorized)
		return domain.User{}, false
	}
	u, err := s.auth.Authenticate(r.Context(), cookie.Value)
	if err != nil {
		failure(w, err)
		return domain.User{}, false
	}
	return u, true
}

func (s *Server) cookie(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{Name: "elevators_session", Value: token, Path: "/", HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteStrictMode, MaxAge: maxAge})
}

func (s *Server) rateLimit(w http.ResponseWriter, r *http.Request) bool {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for key, a := range s.attempts {
		if now.Sub(a.since) > time.Minute {
			delete(s.attempts, key)
		}
	}
	a := s.attempts[ip]
	if a.count == 0 {
		a.since = now
	}
	a.count++
	s.attempts[ip] = a
	if a.count > 20 {
		respond(w, 429, map[string]string{"error": "слишком много попыток, повторите через минуту"})
		return true
	}
	return false
}

func (s *Server) healthcheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.health(ctx); err != nil {
		respond(w, 503, map[string]string{"status": "unavailable"})
		return
	}
	respond(w, 200, map[string]string{"status": "ok"})
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	if s.rateLimit(w, r) {
		return
	}
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decode(r, &in); err != nil {
		failure(w, err)
		return
	}
	u, err := s.auth.Register(r.Context(), in.Username, in.Password)
	if err != nil {
		failure(w, err)
		return
	}
	respond(w, 201, u)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if s.rateLimit(w, r) {
		return
	}
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decode(r, &in); err != nil {
		failure(w, err)
		return
	}
	u, token, err := s.auth.Login(r.Context(), in.Username, in.Password)
	if err != nil {
		failure(w, err)
		return
	}
	s.cookie(w, token, 86400)
	respond(w, 200, u)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("elevators_session"); err == nil {
		if err = s.auth.Logout(r.Context(), c.Value); err != nil {
			failure(w, err)
			return
		}
	}
	s.cookie(w, "", -1)
	respond(w, 200, map[string]bool{"ok": true})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	if u, ok := s.actor(w, r); ok {
		respond(w, 200, u)
	}
}

func (s *Server) users(w http.ResponseWriter, r *http.Request) {
	u, ok := s.actor(w, r)
	if !ok {
		return
	}
	list, err := s.auth.Users(r.Context(), u)
	if err != nil {
		failure(w, err)
		return
	}
	respond(w, 200, list)
}

func (s *Server) updateUser(w http.ResponseWriter, r *http.Request) {
	u, ok := s.actor(w, r)
	if !ok {
		return
	}
	var in struct {
		Role     domain.Role `json:"role"`
		Disabled bool        `json:"disabled"`
	}
	if err := decode(r, &in); err != nil {
		failure(w, err)
		return
	}
	updated, err := s.auth.UpdateUser(r.Context(), u, r.PathValue("id"), in.Role, in.Disabled)
	if err != nil {
		failure(w, err)
		return
	}
	respond(w, 200, updated)
}

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	if u, ok := s.actor(w, r); ok {
		respond(w, 200, s.sim.List(u))
	}
}

type setup struct {
	Config   domain.Config   `json:"config"`
	Scenario domain.Scenario `json:"scenario"`
}

func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	u, ok := s.actor(w, r)
	if !ok {
		return
	}
	var in setup
	if err := decode(r, &in); err != nil {
		failure(w, err)
		return
	}
	created, err := s.sim.Create(u, in.Config, in.Scenario)
	if err != nil {
		failure(w, err)
		return
	}
	respond(w, 201, created)
}

func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	u, ok := s.actor(w, r)
	if !ok {
		return
	}
	current, err := s.sim.Get(u, r.PathValue("id"))
	if err != nil {
		failure(w, err)
		return
	}
	respond(w, 200, current)
}

func (s *Server) configure(w http.ResponseWriter, r *http.Request) {
	u, ok := s.actor(w, r)
	if !ok {
		return
	}
	var in setup
	if err := decode(r, &in); err != nil {
		failure(w, err)
		return
	}
	current, err := s.sim.Configure(u, r.PathValue("id"), in.Config, in.Scenario)
	if err != nil {
		failure(w, err)
		return
	}
	respond(w, 200, current)
}

func (s *Server) delete(w http.ResponseWriter, r *http.Request) {
	u, ok := s.actor(w, r)
	if !ok {
		return
	}
	if err := s.sim.Delete(u, r.PathValue("id")); err != nil {
		failure(w, err)
		return
	}
	respond(w, 200, map[string]bool{"ok": true})
}

func (s *Server) command(w http.ResponseWriter, r *http.Request) {
	u, ok := s.actor(w, r)
	if !ok {
		return
	}
	var in services.Command
	if err := decode(r, &in); err != nil {
		failure(w, err)
		return
	}
	current, err := s.sim.Execute(u, r.PathValue("id"), in)
	if err != nil {
		failure(w, err)
		return
	}
	respond(w, 200, current)
}

func attachment(w http.ResponseWriter, name, contentType string, data []byte) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	_, _ = w.Write(data)
}

func (s *Server) configYAML(w http.ResponseWriter, r *http.Request)   { s.yaml(w, r, false) }
func (s *Server) scenarioYAML(w http.ResponseWriter, r *http.Request) { s.yaml(w, r, true) }
func (s *Server) yaml(w http.ResponseWriter, r *http.Request, scenario bool) {
	u, ok := s.actor(w, r)
	if !ok {
		return
	}
	current, err := s.sim.Get(u, r.PathValue("id"))
	if err != nil {
		failure(w, err)
		return
	}
	if r.Method == "GET" {
		var value any = current.Config
		name := "config.yaml"
		if scenario {
			value = current.Scenario
			name = "scenario.yaml"
		}
		data, err := files.EncodeYAML(value)
		if err != nil {
			failure(w, err)
			return
		}
		attachment(w, name, "application/yaml; charset=utf-8", data)
		return
	}
	if scenario {
		var input domain.Scenario
		err = files.DecodeYAML(r.Body, &input)
		current.Scenario = input
	} else {
		var input domain.Config
		err = files.DecodeYAML(r.Body, &input)
		current.Config = input
	}
	if err != nil {
		failure(w, err)
		return
	}
	current, err = s.sim.Configure(u, current.ID, current.Config, current.Scenario)
	if err != nil {
		failure(w, err)
		return
	}
	respond(w, 200, current)
}

func (s *Server) report(w http.ResponseWriter, r *http.Request) {
	u, ok := s.actor(w, r)
	if !ok {
		return
	}
	current, err := s.sim.Get(u, r.PathValue("id"))
	if err != nil {
		failure(w, err)
		return
	}
	if current.State != domain.Stopped {
		failure(w, fmt.Errorf("отчёт доступен после остановки"))
		return
	}
	attachment(w, "statistics.csv", "text/csv; charset=utf-8", files.Report(current))
}

func (s *Server) stream(w http.ResponseWriter, r *http.Request) {
	u, ok := s.actor(w, r)
	if !ok {
		return
	}
	if _, err := s.sim.Get(u, r.PathValue("id")); err != nil {
		failure(w, err)
		return
	}
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer c.CloseNow()
	ctx := c.CloseRead(r.Context())
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	cookie, _ := r.Cookie("elevators_session")
	for {
		// Длительный WebSocket тоже перепроверяет права: смена роли или блокировка учитываются без нового подключения.
		actor, err := s.auth.Authenticate(ctx, cookie.Value)
		if err != nil {
			_ = c.Close(websocket.StatusPolicyViolation, "авторизация завершена")
			return
		}
		current, err := s.sim.Get(actor, r.PathValue("id"))
		if err != nil {
			_ = c.Close(websocket.StatusPolicyViolation, "сеанс недоступен")
			return
		}
		data, err := json.Marshal(current)
		if err != nil {
			return
		}
		writeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		err = c.Write(writeCtx, websocket.MessageText, data)
		cancel()
		if err != nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
