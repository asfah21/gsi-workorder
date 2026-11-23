package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"gsi-workorder/internal/odoo"
)

const (
	RoleAdmin       = "admin"
	RolePlanner     = "planner"
	RoleMechanic    = "mechanic"
	RoleStorekeeper = "storekeeper"
	RoleViewer      = "viewer"
	RoleDispatch    = "dispatch"
)

var (
	ErrValidation          = errors.New("auth: validation failed")
	ErrInvalidCredentials  = errors.New("auth: invalid credentials")
	ErrOdooUnavailable     = errors.New("auth: odoo unavailable")
	ErrUserNotProvisioned  = errors.New("auth: user not provisioned")
	ErrUserInactive        = errors.New("auth: user inactive")
	ErrUserNotFound        = errors.New("auth: user not found")
	ErrInvalidRole         = errors.New("auth: invalid role")
	ErrRefreshTokenMissing = errors.New("auth: refresh token missing")
)

var validRoles = map[string]struct{}{
	RoleAdmin:       {},
	RolePlanner:     {},
	RoleMechanic:    {},
	RoleStorekeeper: {},
	RoleViewer:      {},
	RoleDispatch:    {},
}

func ValidRole(role string) bool {
	_, ok := validRoles[role]
	return ok
}

// User is the local application identity. Password hashes are intentionally
// absent: authentication is delegated to Odoo.
type User struct {
	ID       int64
	NIK      string
	Name     string
	Email    string
	Role     string
	IsActive bool
}

// Repository is the persistence contract used by the auth service.
type Repository interface {
	FindByNIK(ctx context.Context, nik string) (User, error)
	FindByID(ctx context.Context, id int64) (User, error)
	TouchLogin(ctx context.Context, id int64) error
	UpdateRole(ctx context.Context, id int64, role string) (User, error)
	EnsureAdmin(ctx context.Context, nik, name string) (User, error)
}

// PasswordAuthenticator validates credentials against Odoo.
type PasswordAuthenticator interface {
	Login(ctx context.Context, login, password string) error
}

type LoginResult struct {
	User   User
	Tokens TokenPair
}

type Service struct {
	repo   Repository
	odoo   PasswordAuthenticator
	tokens *TokenManager
	logger *slog.Logger
}

func NewService(repo Repository, odooClient PasswordAuthenticator, tokens *TokenManager, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		repo:   repo,
		odoo:   odooClient,
		tokens: tokens,
		logger: logger,
	}
}

func (s *Service) Login(ctx context.Context, nik, password string) (LoginResult, error) {
	nik = strings.TrimSpace(nik)
	if nik == "" || password == "" {
		return LoginResult{}, ErrValidation
	}

	if err := s.odoo.Login(ctx, nik, password); err != nil {
		switch {
		case errors.Is(err, odoo.ErrInvalidCredentials):
			return LoginResult{}, ErrInvalidCredentials
		case errors.Is(err, odoo.ErrUnavailable), errors.Is(err, odoo.ErrUnexpectedResponse):
			return LoginResult{}, fmt.Errorf("%w: %v", ErrOdooUnavailable, err)
		default:
			return LoginResult{}, fmt.Errorf("validate Odoo credentials: %w", err)
		}
	}

	user, err := s.repo.FindByNIK(ctx, nik)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return LoginResult{}, ErrUserNotProvisioned
		}
		return LoginResult{}, fmt.Errorf("find user: %w", err)
	}
	if !user.IsActive {
		return LoginResult{}, ErrUserInactive
	}

	tokens, err := s.tokens.IssuePair(user.ID, user.Role)
	if err != nil {
		return LoginResult{}, fmt.Errorf("issue tokens: %w", err)
	}
	if err := s.repo.TouchLogin(ctx, user.ID); err != nil {
		s.logger.Warn("update last_login_at failed", "user_id", user.ID, "error", err)
	}

	return LoginResult{User: user, Tokens: tokens}, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (LoginResult, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return LoginResult{}, ErrRefreshTokenMissing
	}

	claims, err := s.tokens.ParseRefresh(refreshToken)
	if err != nil {
		return LoginResult{}, ErrInvalidToken
	}
	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || userID <= 0 {
		return LoginResult{}, ErrInvalidToken
	}

	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return LoginResult{}, ErrInvalidToken
		}
		return LoginResult{}, fmt.Errorf("find user for refresh: %w", err)
	}
	if !user.IsActive {
		return LoginResult{}, ErrUserInactive
	}

	tokens, err := s.tokens.IssuePair(user.ID, user.Role)
	if err != nil {
		return LoginResult{}, fmt.Errorf("issue refreshed tokens: %w", err)
	}
	return LoginResult{User: user, Tokens: tokens}, nil
}

func (s *Service) Authenticate(ctx context.Context, accessToken string) (User, error) {
	claims, err := s.tokens.ParseAccess(accessToken)
	if err != nil {
		return User{}, ErrInvalidToken
	}
	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || userID <= 0 {
		return User{}, ErrInvalidToken
	}

	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return User{}, ErrInvalidToken
		}
		return User{}, fmt.Errorf("find user for authentication: %w", err)
	}
	if !user.IsActive {
		return User{}, ErrUserInactive
	}
	return user, nil
}

func (s *Service) UpdateRole(ctx context.Context, userID int64, role string) (User, error) {
	if userID <= 0 || !ValidRole(role) {
		return User{}, ErrInvalidRole
	}
	user, err := s.repo.UpdateRole(ctx, userID, role)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *Service) EnsureAdmin(ctx context.Context, nik, name string) (User, error) {
	nik = strings.TrimSpace(nik)
	name = strings.TrimSpace(name)
	if nik == "" {
		return User{}, ErrValidation
	}
	if name == "" {
		name = "Seed Admin"
	}
	return s.repo.EnsureAdmin(ctx, nik, name)
}
