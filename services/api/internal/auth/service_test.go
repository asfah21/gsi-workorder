package auth

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"gsi-workorder/internal/odoo"
)

type fakeRepo struct {
	byNIK   map[string]User
	byID    map[int64]User
	touched int64
	updated *User
}

func (f *fakeRepo) FindByNIK(_ context.Context, nik string) (User, error) {
	user, ok := f.byNIK[nik]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return user, nil
}

func (f *fakeRepo) FindByID(_ context.Context, id int64) (User, error) {
	user, ok := f.byID[id]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return user, nil
}

func (f *fakeRepo) TouchLogin(_ context.Context, id int64) error {
	f.touched = id
	return nil
}

func (f *fakeRepo) UpdateRole(_ context.Context, id int64, role string) (User, error) {
	user, ok := f.byID[id]
	if !ok {
		return User{}, ErrUserNotFound
	}
	user.Role = role
	f.updated = &user
	return user, nil
}

func (f *fakeRepo) EnsureAdmin(_ context.Context, nik, name string) (User, error) {
	user := User{ID: 99, NIK: nik, Name: name, Role: RoleAdmin, IsActive: true}
	return user, nil
}

type fakeOdoo struct {
	err    error
	called bool
}

func (f *fakeOdoo) Login(_ context.Context, _, _ string) error {
	f.called = true
	return f.err
}

func testService(t *testing.T, repo Repository, odooClient PasswordAuthenticator) *Service {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewService(repo, odooClient, NewTokenManager("0123456789abcdef0123456789abcdef"), logger)
}

func TestLoginSuccess(t *testing.T) {
	repo := &fakeRepo{
		byNIK: map[string]User{
			"12345": {ID: 7, NIK: "12345", Name: "Rendi", Role: RoleMechanic, IsActive: true},
		},
		byID: map[int64]User{
			7: {ID: 7, NIK: "12345", Name: "Rendi", Role: RoleMechanic, IsActive: true},
		},
	}
	odooClient := &fakeOdoo{}
	service := testService(t, repo, odooClient)

	result, err := service.Login(context.Background(), " 12345 ", "secret")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if result.User.ID != 7 || result.Tokens.AccessToken == "" {
		t.Fatalf("result = %#v", result)
	}
	if repo.touched != 7 {
		t.Fatalf("TouchLogin id = %d, want 7", repo.touched)
	}
	if !odooClient.called {
		t.Fatal("Odoo login was not called")
	}
}

func TestLoginRejectsInvalidCredentials(t *testing.T) {
	repo := &fakeRepo{}
	service := testService(t, repo, &fakeOdoo{err: odoo.ErrInvalidCredentials})

	_, err := service.Login(context.Background(), "12345", "wrong")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
	}
}

func TestLoginRejectsOdooUnavailable(t *testing.T) {
	repo := &fakeRepo{}
	service := testService(t, repo, &fakeOdoo{err: odoo.ErrUnavailable})

	_, err := service.Login(context.Background(), "12345", "secret")
	if !errors.Is(err, ErrOdooUnavailable) {
		t.Fatalf("Login() error = %v, want ErrOdooUnavailable", err)
	}
}

func TestLoginRejectsUnprovisionedUser(t *testing.T) {
	repo := &fakeRepo{byNIK: map[string]User{}}
	service := testService(t, repo, &fakeOdoo{})

	_, err := service.Login(context.Background(), "12345", "secret")
	if !errors.Is(err, ErrUserNotProvisioned) {
		t.Fatalf("Login() error = %v, want ErrUserNotProvisioned", err)
	}
}

func TestLoginRejectsInactiveUser(t *testing.T) {
	repo := &fakeRepo{
		byNIK: map[string]User{
			"12345": {ID: 7, NIK: "12345", Role: RoleMechanic, IsActive: false},
		},
	}
	service := testService(t, repo, &fakeOdoo{})

	_, err := service.Login(context.Background(), "12345", "secret")
	if !errors.Is(err, ErrUserInactive) {
		t.Fatalf("Login() error = %v, want ErrUserInactive", err)
	}
}

func TestLoginValidationDoesNotCallOdoo(t *testing.T) {
	odooClient := &fakeOdoo{}
	service := testService(t, &fakeRepo{}, odooClient)

	_, err := service.Login(context.Background(), "", "secret")
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Login() error = %v, want ErrValidation", err)
	}
	if odooClient.called {
		t.Fatal("Odoo login was called for invalid input")
	}
}

func TestRefreshAndAuthenticate(t *testing.T) {
	repo := &fakeRepo{
		byID: map[int64]User{
			7: {ID: 7, NIK: "12345", Role: RoleAdmin, IsActive: true},
		},
	}
	service := testService(t, repo, &fakeOdoo{})
	pair, err := service.tokens.IssuePair(7, RoleAdmin)
	if err != nil {
		t.Fatalf("IssuePair() error = %v", err)
	}

	user, err := service.Authenticate(context.Background(), pair.AccessToken)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if user.ID != 7 || user.Role != RoleAdmin {
		t.Fatalf("user = %#v", user)
	}

	result, err := service.Refresh(context.Background(), pair.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if result.User.ID != 7 || result.Tokens.AccessToken == "" {
		t.Fatalf("result = %#v", result)
	}
}

func TestAuthenticateRejectsInactiveUser(t *testing.T) {
	repo := &fakeRepo{
		byID: map[int64]User{
			7: {ID: 7, Role: RoleAdmin, IsActive: false},
		},
	}
	service := testService(t, repo, &fakeOdoo{})
	pair, err := service.tokens.IssuePair(7, RoleAdmin)
	if err != nil {
		t.Fatalf("IssuePair() error = %v", err)
	}

	if _, err := service.Authenticate(context.Background(), pair.AccessToken); !errors.Is(err, ErrUserInactive) {
		t.Fatalf("Authenticate() error = %v, want ErrUserInactive", err)
	}
}

func TestUpdateRoleValidation(t *testing.T) {
	repo := &fakeRepo{
		byID: map[int64]User{
			7: {ID: 7, Role: RoleMechanic, IsActive: true},
		},
	}
	service := testService(t, repo, &fakeOdoo{})

	if _, err := service.UpdateRole(context.Background(), 7, "superuser"); !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("UpdateRole() error = %v, want ErrInvalidRole", err)
	}

	user, err := service.UpdateRole(context.Background(), 7, RoleStorekeeper)
	if err != nil {
		t.Fatalf("UpdateRole() error = %v", err)
	}
	if user.Role != RoleStorekeeper || repo.updated == nil {
		t.Fatalf("user = %#v, updated = %#v", user, repo.updated)
	}
}
