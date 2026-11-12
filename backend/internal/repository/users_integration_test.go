package repository

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/ramanmalinouski/Elevators/backend/internal/domain"
	"github.com/ramanmalinouski/Elevators/backend/internal/services"
)

func TestPostgresAccountsAndGoose(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to a disposable PostgreSQL database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil || cfg.ConnConfig.Database != "elevators_test" {
		t.Fatal("integration tests require a disposable database named elevators_test")
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	if err = goose.UpContext(ctx, db, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	_, err = pool.Exec(ctx, "TRUNCATE users CASCADE")
	if err != nil {
		t.Fatal(err)
	}
	repo := NewUsers(pool)
	auth, err := services.NewAuth(repo)
	if err != nil {
		t.Fatal(err)
	}
	u, err := auth.Register(ctx, "integration_user", "test-password-123")
	if err != nil {
		t.Fatal(err)
	}
	if u.Role != domain.UserRole {
		t.Fatal("registration grants admin")
	}
	if _, err = auth.Register(ctx, "integration_user", "test-password-123"); !errors.Is(err, services.ErrConflict) {
		t.Fatal("duplicate user accepted")
	}
	_, token, err := auth.Login(ctx, u.Username, "test-password-123")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = auth.Login(ctx, u.Username, "wrong-password"); !errors.Is(err, services.ErrUnauthorized) {
		t.Fatal("wrong password accepted")
	}
	if _, err = auth.Authenticate(ctx, token); err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, "UPDATE users SET role='admin' WHERE id=$1", u.ID)
	if err != nil {
		t.Fatal(err)
	}
	admin, err := auth.Authenticate(ctx, token)
	if err != nil || admin.Role != domain.AdminRole {
		t.Fatal("token retains stale role")
	}
	if _, err = repo.Update(ctx, u.ID, domain.UserRole, false); err == nil {
		t.Fatal("last admin removed")
	}
	other, err := auth.Register(ctx, "second_admin", "test-password-123")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Update(ctx, other.ID, domain.AdminRole, false); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Update(ctx, u.ID, domain.UserRole, true); err != nil {
		t.Fatal(err)
	}
	if _, err = auth.Authenticate(ctx, token); !errors.Is(err, services.ErrUnauthorized) {
		t.Fatal("disabled user retains access")
	}
	if err = goose.DownContext(ctx, db, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	if err = goose.UpContext(ctx, db, "../../migrations"); err != nil {
		t.Fatal(err)
	}
}
