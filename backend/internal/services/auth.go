package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ramanmalinouski/Elevators/backend/internal/domain"
	"github.com/ramanmalinouski/Elevators/backend/pkg/security"
)

var (
	ErrUnauthorized = errors.New("требуется авторизация")
	ErrForbidden    = errors.New("недостаточно прав")
	ErrNotFound     = errors.New("не найдено")
	ErrConflict     = errors.New("пользователь уже существует")
	ErrInternal     = errors.New("внутренняя ошибка сервера")
	usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]{3,40}$`)
)

type UserRepository interface {
	Create(context.Context, string, string) (domain.User, error)
	ByUsername(context.Context, string) (domain.User, error)
	ByToken(context.Context, string) (domain.User, error)
	SaveToken(context.Context, string, string, time.Time) error
	DeleteToken(context.Context, string) error
	List(context.Context) ([]domain.User, error)
	Update(context.Context, string, domain.Role, bool) (domain.User, error)
}

type Auth struct {
	users     UserRepository
	dummyHash string
}

func NewAuth(users UserRepository) (*Auth, error) {
	hash, err := security.HashPassword("dummy-password-never-used")
	if err != nil {
		return nil, err
	}
	return &Auth{users: users, dummyHash: hash}, nil
}

func (a *Auth) Register(ctx context.Context, username, password string) (domain.User, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	if !usernamePattern.MatchString(username) {
		return domain.User{}, fmt.Errorf("логин: 3–40 латинских букв, цифр или символов _ . -")
	}
	if len(password) < 8 || len(password) > 72 {
		return domain.User{}, fmt.Errorf("пароль: от 8 до 72 байт")
	}
	hash, err := security.HashPassword(password)
	if err != nil {
		return domain.User{}, err
	}
	return a.users.Create(ctx, username, hash)
}

func (a *Auth) Login(ctx context.Context, username, password string) (domain.User, string, error) {
	u, err := a.users.ByUsername(ctx, strings.ToLower(strings.TrimSpace(username)))
	if err != nil && !errors.Is(err, ErrNotFound) {
		return domain.User{}, "", err
	}
	hash := u.PasswordHash
	if err != nil {
		// Проверяем bcrypt и для неизвестного логина, чтобы не выдавать существование аккаунта быстрым ответом.
		hash = a.dummyHash
	}
	valid := security.CheckPassword(hash, password)
	if !valid || err != nil || u.Disabled {
		return domain.User{}, "", ErrUnauthorized
	}
	token, err := security.NewToken()
	if err != nil {
		return domain.User{}, "", err
	}
	if err = a.users.SaveToken(ctx, security.TokenHash(token), u.ID, time.Now().Add(24*time.Hour)); err != nil {
		return domain.User{}, "", err
	}
	return u, token, nil
}

func (a *Auth) Authenticate(ctx context.Context, token string) (domain.User, error) {
	if len(token) != 64 {
		return domain.User{}, ErrUnauthorized
	}
	// Роль и блокировка читаются заново при каждом запросе: старый токен не сохраняет прежние права.
	u, err := a.users.ByToken(ctx, security.TokenHash(token))
	if errors.Is(err, ErrNotFound) || u.Disabled {
		return domain.User{}, ErrUnauthorized
	}
	return u, err
}

func (a *Auth) Logout(ctx context.Context, token string) error {
	return a.users.DeleteToken(ctx, security.TokenHash(token))
}

func (a *Auth) Users(ctx context.Context, actor domain.User) ([]domain.User, error) {
	if actor.Role != domain.AdminRole {
		return nil, ErrForbidden
	}
	return a.users.List(ctx)
}

func (a *Auth) UpdateUser(ctx context.Context, actor domain.User, id string, role domain.Role, disabled bool) (domain.User, error) {
	if actor.Role != domain.AdminRole {
		return domain.User{}, ErrForbidden
	}
	if role != domain.UserRole && role != domain.AdminRole {
		return domain.User{}, fmt.Errorf("неизвестная роль")
	}
	if actor.ID == id && disabled {
		return domain.User{}, fmt.Errorf("нельзя заблокировать собственную учётную запись")
	}
	return a.users.Update(ctx, id, role, disabled)
}
