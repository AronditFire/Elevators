package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/ramanmalinouski/Elevators/backend/internal/config"
	"github.com/ramanmalinouski/Elevators/backend/internal/repository"
	"github.com/ramanmalinouski/Elevators/backend/internal/services"
	transport "github.com/ramanmalinouski/Elevators/backend/internal/transport/http"
	"github.com/ramanmalinouski/Elevators/backend/pkg/database"
	"github.com/ramanmalinouski/Elevators/backend/pkg/runner"
)

func Run(ctx context.Context, cfg config.Config) error {
	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	defer pool.Close()
	auth, err := services.NewAuth(repository.NewUsers(pool))
	if err != nil {
		return err
	}
	sim := services.NewSimulation()
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan struct{})
	go func() { defer close(done); runner.Run(runCtx, sim) }()
	defer func() { cancel(); <-done }()
	handler := transport.New(auth, sim, cfg.CookieSecure, pool.Ping)
	server := &http.Server{Addr: cfg.Address, Handler: handler.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second, BaseContext: func(net.Listener) context.Context { return runCtx }}
	result := make(chan error, 1)
	go func() { slog.Info("server started", "address", cfg.Address); result <- server.ListenAndServe() }()
	select {
	case err := <-result:
		if err != http.ErrServerClosed {
			return err
		}
		return nil
	case <-ctx.Done():
		cancel()
		shutdownCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		return server.Shutdown(shutdownCtx)
	}
}
