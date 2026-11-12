package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		slog.Error("open", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	if err = goose.SetDialect("postgres"); err != nil {
		slog.Error("dialect", "error", err)
		os.Exit(1)
	}
	dir := os.Getenv("MIGRATIONS_DIR")
	if dir == "" {
		dir = "migrations"
	}
	action := "up"
	if len(os.Args) > 1 {
		action = os.Args[1]
	}
	if err = goose.RunContext(ctx, action, db, dir); err != nil {
		slog.Error("migration", "error", err)
		os.Exit(1)
	}
}
