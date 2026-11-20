package usersync

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
)

type fakeRPC struct {
	uid        int64
	loginErr   error
	users      []map[string]json.RawMessage
	employees  []map[string]json.RawMessage
	readErr    error
	modelCalls []string
}

func (f *fakeRPC) ServiceLogin(context.Context, string, string, string) (int64, error) {
	return f.uid, f.loginErr
}

func (f *fakeRPC) SearchRead(
	_ context.Context,
	_ string,
	_ int64,
	_ string,
	model string,
	_ []any,
	_ []string,
	_ map[string]any,
) ([]map[string]json.RawMessage, error) {
	f.modelCalls = append(f.modelCalls, model)
	if f.readErr != nil {
		return nil, f.readErr
	}
	switch model {
	case "res.users":
		return f.users, nil
	case "hr.employee":
		return f.employees, nil
	default:
		return nil, errors.New("unexpected model")
	}
}

type fakeRepo struct {
	users        []OdooUser
	employees    []OdooEmployee
	employeeUser map[int64]*int64
	fallbackNIKs map[int64]string
	started      int
	finished     []string
	state        map[string]string
}

func (f *fakeRepo) UpsertUser(_ context.Context, user OdooUser, fallbackNIK string) (int64, error) {
	f.users = append(f.users, user)
	if f.fallbackNIKs == nil {
		f.fallbackNIKs = map[int64]string{}
	}
	f.fallbackNIKs[user.OdooID] = fallbackNIK
	return int64(100 + len(f.users)), nil
}

func (f *fakeRepo) UpsertEmployee(_ context.Context, employee OdooEmployee, localUserID *int64) error {
	f.employees = append(f.employees, employee)
	if f.employeeUser == nil {
		f.employeeUser = map[int64]*int64{}
	}
	f.employeeUser[employee.OdooID] = localUserID
	return nil
}

func (f *fakeRepo) StartSyncRun(context.Context, string) (int64, error) {
	f.started++
	return int64(f.started), nil
}

func (f *fakeRepo) FinishSyncRun(_ context.Context, _ int64, status, _ string) error {
	f.finished = append(f.finished, status)
	return nil
}

func (f *fakeRepo) SetSyncState(_ context.Context, key, value string) error {
	if f.state == nil {
		f.state = map[string]string{}
	}
	f.state[key] = value
	return nil
}

func makeRow(t *testing.T, values map[string]any) map[string]json.RawMessage {
	t.Helper()
	row := make(map[string]json.RawMessage, len(values))
	for key, value := range values {
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("marshal %s: %v", key, err)
		}
		row[key] = raw
	}
	return row
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestSyncUsersAndEmployees(t *testing.T) {
	rpc := &fakeRPC{
		uid: 6333,
		users: []map[string]json.RawMessage{
			makeRow(t, map[string]any{
				"id": 1, "login": "svc_gsi_workorder", "name": "Service", "email": "svc@example.com", "active": true,
			}),
			makeRow(t, map[string]any{
				"id": 123, "login": "2306001", "name": "Admin", "email": "admin@example.com", "active": true,
			}),
		},
		employees: []map[string]json.RawMessage{
			makeRow(t, map[string]any{
				"id": 99, "nik": "2306001", "name": "Admin", "job_title": "Manager",
				"department_id": []any{1, "Operations"}, "work_location_id": []any{2, "Site A"},
				"user_id": []any{123, "Admin"}, "active": true,
			}),
		},
	}
	repo := &fakeRepo{}
	service := NewService(rpc, repo, "odoo_new", "svc_gsi_workorder", "secret", "nik", testLogger())

	if err := service.Sync(context.Background()); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if len(repo.users) != 1 || repo.users[0].Login != "2306001" {
		t.Fatalf("users = %#v", repo.users)
	}
	if len(repo.employees) != 1 || repo.employees[0].NIK != "2306001" {
		t.Fatalf("employees = %#v", repo.employees)
	}
	if repo.employeeUser[99] == nil || *repo.employeeUser[99] != 101 {
		t.Fatalf("employee user link = %#v", repo.employeeUser[99])
	}
	if repo.fallbackNIKs[123] != "2306001" {
		t.Fatalf("fallback NIK = %q", repo.fallbackNIKs[123])
	}
	if len(repo.finished) != 1 || repo.finished[0] != "success" {
		t.Fatalf("finished = %#v", repo.finished)
	}
	if repo.state["users_synced_count"] != "1" || repo.state["employees_synced_count"] != "1" {
		t.Fatalf("state = %#v", repo.state)
	}
	if len(rpc.modelCalls) != 2 || rpc.modelCalls[0] != "res.users" || rpc.modelCalls[1] != "hr.employee" {
		t.Fatalf("model calls = %#v", rpc.modelCalls)
	}
}

func TestSyncFailureMarksRunFailed(t *testing.T) {
	rpc := &fakeRPC{loginErr: errors.New("odoo unavailable")}
	repo := &fakeRepo{}
	service := NewService(rpc, repo, "odoo_new", "svc", "secret", "nik", testLogger())

	err := service.Sync(context.Background())
	if err == nil {
		t.Fatal("Sync() expected an error")
	}
	if len(repo.finished) != 1 || repo.finished[0] != "failed" {
		t.Fatalf("finished = %#v", repo.finished)
	}
	if len(repo.users) != 0 || len(repo.employees) != 0 {
		t.Fatalf("unexpected upserts: users=%d employees=%d", len(repo.users), len(repo.employees))
	}
}

func TestSyncRejectsMissingNIKField(t *testing.T) {
	rpc := &fakeRPC{
		uid: 6333,
		users: []map[string]json.RawMessage{
			makeRow(t, map[string]any{"id": 123, "login": "2306001", "name": "Admin", "active": true}),
		},
		employees: []map[string]json.RawMessage{
			makeRow(t, map[string]any{"id": 99, "name": "Admin", "active": true}),
		},
	}
	repo := &fakeRepo{}
	service := NewService(rpc, repo, "odoo_new", "svc", "secret", "nik", testLogger())

	if err := service.Sync(context.Background()); err == nil {
		t.Fatal("Sync() expected an error for a missing NIK field")
	}
	if len(repo.finished) != 1 || repo.finished[0] != "failed" {
		t.Fatalf("finished = %#v", repo.finished)
	}
}

func TestParseUsersAcceptsFalseOptionalFields(t *testing.T) {
	rows := []map[string]json.RawMessage{
		makeRow(t, map[string]any{"id": 1, "login": "2306001", "name": false, "email": false, "active": false}),
	}
	users, err := parseUsers(rows)
	if err != nil {
		t.Fatalf("parseUsers() error = %v", err)
	}
	if len(users) != 1 || users[0].Name != "" || users[0].Email != "" || users[0].Active {
		t.Fatalf("users = %#v", users)
	}
}
