package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ramanmalinouski/Elevators/backend/internal/domain"
	"github.com/ramanmalinouski/Elevators/backend/internal/services"
)

type Users struct{ pool *pgxpool.Pool }

func NewUsers(pool *pgxpool.Pool) *Users { return &Users{pool: pool} }

func scanUser(row pgx.Row) (domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.Disabled, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, services.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return u, services.ErrConflict
	}
	if err != nil {
		return u, fmt.Errorf("%w: %v", services.ErrInternal, err)
	}
	return u, err
}

const columns = "id::text, username, password_hash, role, disabled, created_at"

func (r *Users) Create(ctx context.Context, name, hash string) (domain.User, error) {
	return scanUser(r.pool.QueryRow(ctx, "INSERT INTO users(username,password_hash) VALUES($1,$2) RETURNING "+columns, name, hash))
}

func (r *Users) ByUsername(ctx context.Context, name string) (domain.User, error) {
	return scanUser(r.pool.QueryRow(ctx, "SELECT "+columns+" FROM users WHERE username=$1", name))
}

func (r *Users) ByToken(ctx context.Context, token string) (domain.User, error) {
	return scanUser(r.pool.QueryRow(ctx, "SELECT u.id::text,u.username,u.password_hash,u.role,u.disabled,u.created_at FROM users u JOIN auth_sessions a ON a.user_id=u.id WHERE a.token_hash=$1 AND a.expires_at>now()", token))
}

func (r *Users) SaveToken(ctx context.Context, token, id string, expires time.Time) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM auth_sessions WHERE expires_at<now()")
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, "INSERT INTO auth_sessions(token_hash,user_id,expires_at) VALUES($1,$2,$3)", token, id, expires)
	return err
}

func (r *Users) DeleteToken(ctx context.Context, token string) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM auth_sessions WHERE token_hash=$1", token)
	return err
}

func (r *Users) List(ctx context.Context) ([]domain.User, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+columns+" FROM users ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *Users) Update(ctx context.Context, id string, role domain.Role, disabled bool) (domain.User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.User{}, err
	}
	defer tx.Rollback(ctx)
	// Сериализуем изменения прав: два администратора не должны одновременно пройти проверку и понизить друг друга.
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(824632)"); err != nil {
		return domain.User{}, err
	}
	u, err := scanUser(tx.QueryRow(ctx, "SELECT "+columns+" FROM users WHERE id::text=$1 FOR UPDATE", id))
	if err != nil {
		return u, err
	}
	// В системе должен оставаться хотя бы один активный администратор; проверка и изменение выполняются в одной транзакции.
	if u.Role == domain.AdminRole && !u.Disabled && (role != domain.AdminRole || disabled) {
		var count int
		if err = tx.QueryRow(ctx, "SELECT count(*) FROM users WHERE role='admin' AND NOT disabled AND id::text<>$1", id).Scan(&count); err != nil {
			return u, err
		}
		if count == 0 {
			return u, fmt.Errorf("нельзя удалить последнего активного администратора")
		}
	}
	u, err = scanUser(tx.QueryRow(ctx, "UPDATE users SET role=$2,disabled=$3 WHERE id::text=$1 RETURNING "+columns, id, role, disabled))
	if err != nil {
		return u, err
	}
	if disabled {
		if _, err = tx.Exec(ctx, "DELETE FROM auth_sessions WHERE user_id::text=$1", id); err != nil {
			return u, err
		}
	}
	return u, tx.Commit(ctx)
}
