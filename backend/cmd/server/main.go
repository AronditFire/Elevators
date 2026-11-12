package main

import (
	"context"
	"github.com/ramanmalinouski/Elevators/backend/internal/app"
	"github.com/ramanmalinouski/Elevators/backend/internal/config"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration", "error", err)
		os.Exit(1)
	}
	if err = app.Run(ctx, cfg); err != nil {
		slog.Error("application", "error", err)
		os.Exit(1)
	}
}
