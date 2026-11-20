package usersync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// OdooUser is the subset of res.users needed by the local identity model.
type OdooUser struct {
	OdooID int64
	Login  string
	Name   string
	Email  string
	Active bool
}

// OdooEmployee is the subset of hr.employee needed by the local master.
type OdooEmployee struct {
	OdooID       int64
	NIK          string
	Name         string
	JobTitle     string
	Department   string
	SiteLocation string
	OdooUserID   *int64
	Active       bool
}

// RPC is the read-only Odoo JSON-RPC surface used by the sync service.
type RPC interface {
	ServiceLogin(ctx context.Context, db, login, password string) (int64, error)
	SearchRead(
		ctx context.Context,
		db string,
		uid int64,
		password string,
		model string,
		domain []any,
		fields []string,
		searchContext map[string]any,
	) ([]map[string]json.RawMessage, error)
}

// Repository is the local persistence contract for M1B sync.
type Repository interface {
	UpsertUser(ctx context.Context, user OdooUser, fallbackNIK string) (int64, error)
	UpsertEmployee(ctx context.Context, employee OdooEmployee, localUserID *int64) error
	StartSyncRun(ctx context.Context, kind string) (int64, error)
	FinishSyncRun(ctx context.Context, id int64, status, errorMessage string) error
	SetSyncState(ctx context.Context, key, value string) error
}

type Service struct {
	rpc             RPC
	repo            Repository
	db              string
	serviceLogin    string
	servicePassword string
	nikField        string
	logger          *slog.Logger
}

func NewService(
	rpc RPC,
	repo Repository,
	db string,
	serviceLogin string,
	servicePassword string,
	nikField string,
	logger *slog.Logger,
) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		rpc:             rpc,
		repo:            repo,
		db:              db,
		serviceLogin:    serviceLogin,
		servicePassword: servicePassword,
		nikField:        nikField,
		logger:          logger,
	}
}

// Sync reads users and employees from Odoo and upserts them locally. It never
// deletes local rows and never overwrites a local role.
func (s *Service) Sync(ctx context.Context) error {
	runID, err := s.repo.StartSyncRun(ctx, "users")
	if err != nil {
		return fmt.Errorf("start user sync run: %w", err)
	}

	fail := func(syncErr error) error {
		message := truncateError(syncErr.Error())
		if finishErr := s.repo.FinishSyncRun(ctx, runID, "failed", message); finishErr != nil {
			s.logger.Warn("finish failed user sync run", "error", finishErr)
		}
		return syncErr
	}

	uid, err := s.rpc.ServiceLogin(ctx, s.db, s.serviceLogin, s.servicePassword)
	if err != nil {
		return fail(fmt.Errorf("odoo service login: %w", err))
	}

	userRows, err := s.rpc.SearchRead(
		ctx,
		s.db,
		uid,
		s.servicePassword,
		"res.users",
		[]any{[]any{"share", "=", false}},
		[]string{"id", "login", "name", "email", "active"},
		map[string]any{"active_test": false},
	)
	if err != nil {
		return fail(fmt.Errorf("read Odoo users: %w", err))
	}

	users, err := parseUsers(userRows)
	if err != nil {
		return fail(fmt.Errorf("parse Odoo users: %w", err))
	}

	employeeFields := []string{
		"id",
		s.nikField,
		"name",
		"job_title",
		"department_id",
		"work_location_id",
		"user_id",
		"active",
	}
	employeeRows, err := s.rpc.SearchRead(
		ctx,
		s.db,
		uid,
		s.servicePassword,
		"hr.employee",
		[]any{},
		employeeFields,
		map[string]any{"active_test": false},
	)
	if err != nil {
		return fail(fmt.Errorf("read Odoo employees: %w", err))
	}

	employees, err := parseEmployees(employeeRows, s.nikField)
	if err != nil {
		return fail(fmt.Errorf("parse Odoo employees: %w", err))
	}

	// If Odoo login differs from the employee NIK (for example an email login),
	// use the employee NIK as a fallback to match an existing local seed admin.
	employeeNIKByUserID := make(map[int64]string)
	for _, employee := range employees {
		if employee.OdooUserID != nil && strings.TrimSpace(employee.NIK) != "" {
			employeeNIKByUserID[*employee.OdooUserID] = employee.NIK
		}
	}

	userMap := make(map[int64]int64, len(users))
	usersSynced := 0
	for _, user := range users {
		if strings.EqualFold(strings.TrimSpace(user.Login), strings.TrimSpace(s.serviceLogin)) {
			continue
		}
		localID, err := s.repo.UpsertUser(ctx, user, employeeNIKByUserID[user.OdooID])
		if err != nil {
			return fail(fmt.Errorf("upsert user %d: %w", user.OdooID, err))
		}
		userMap[user.OdooID] = localID
		usersSynced++
	}

	employeesSynced := 0
	for _, employee := range employees {
		var localUserID *int64
		if employee.OdooUserID != nil {
			if id, ok := userMap[*employee.OdooUserID]; ok {
				localUserID = &id
			}
		}
		if err := s.repo.UpsertEmployee(ctx, employee, localUserID); err != nil {
			return fail(fmt.Errorf("upsert employee %d: %w", employee.OdooID, err))
		}
		employeesSynced++
	}

	if err := s.repo.FinishSyncRun(ctx, runID, "success", ""); err != nil {
		return fmt.Errorf("finish user sync run: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	states := map[string]string{
		"users_last_synced_at":   now,
		"users_synced_count":     fmt.Sprintf("%d", usersSynced),
		"employees_synced_count": fmt.Sprintf("%d", employeesSynced),
	}
	for key, value := range states {
		if err := s.repo.SetSyncState(ctx, key, value); err != nil {
			s.logger.Warn("set sync state failed", "key", key, "error", err)
		}
	}

	s.logger.Info("user sync completed", "users", usersSynced, "employees", employeesSynced)
	return nil
}

func truncateError(message string) string {
	const max = 1000
	if len(message) <= max {
		return message
	}
	return message[:max]
}

var errMissingField = errors.New("usersync: required field missing")
