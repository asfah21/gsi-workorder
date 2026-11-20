package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"gsi-workorder/internal/auth"
)

type fakeAuthService struct {
	loginResult    auth.LoginResult
	loginErr       error
	refreshResult  auth.LoginResult
	refreshErr     error
	authUser       auth.User
	authErr        error
	updateRoleUser auth.User
	updateRoleErr  error
	lastRole       string
}

func (f *fakeAuthService) Login(context.Context, string, string) (auth.LoginResult, error) {
	return f.loginResult, f.loginErr
}

func (f *fakeAuthService) Refresh(context.Context, string) (auth.LoginResult, error) {
	return f.refreshResult, f.refreshErr
}

func (f *fakeAuthService) Authenticate(_ context.Context, token string) (auth.User, error) {
	if token == "good" {
		return f.authUser, f.authErr
	}
	return auth.User{}, f.authErr
}

func (f *fakeAuthService) UpdateRole(_ context.Context, _ int64, role string) (auth.User, error) {
	f.lastRole = role
	return f.updateRoleUser, f.updateRoleErr
}

func authTestRouter(service AuthService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	return NewRouter(testLogger(), CheckFunc(func(context.Context) error { return nil }), service)
}

func TestLoginHandlerSuccess(t *testing.T) {
	service := &fakeAuthService{
		loginResult: auth.LoginResult{
			User: auth.User{ID: 7, NIK: "12345", Name: "Rendi", Role: auth.RoleMechanic, IsActive: true},
			Tokens: auth.TokenPair{
				AccessToken:  "access",
				RefreshToken: "refresh",
				ExpiresAt:    time.Date(2026, 11, 20, 20, 0, 0, 0, time.UTC),
				ExpiresIn:    43200,
			},
		},
	}
	router := authTestRouter(service)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"nik":"12345","password":"secret"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var body tokenResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.AccessToken != "access" || body.User.NIK != "12345" {
		t.Fatalf("body = %#v", body)
	}
}

func TestLoginHandlerMapsInvalidCredentials(t *testing.T) {
	router := authTestRouter(&fakeAuthService{loginErr: auth.ErrInvalidCredentials})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"nik":"12345","password":"wrong"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestMeRequiresBearerToken(t *testing.T) {
	router := authTestRouter(&fakeAuthService{authErr: auth.ErrInvalidToken})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestMeReturnsCurrentUser(t *testing.T) {
	router := authTestRouter(&fakeAuthService{
		authUser: auth.User{ID: 7, NIK: "12345", Role: auth.RoleAdmin, IsActive: true},
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	request.Header.Set("Authorization", "Bearer good")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestRoleEndpointRejectsNonAdmin(t *testing.T) {
	router := authTestRouter(&fakeAuthService{
		authUser: auth.User{ID: 7, Role: auth.RoleMechanic, IsActive: true},
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/api/users/7/role", strings.NewReader(`{"role":"storekeeper"}`))
	request.Header.Set("Authorization", "Bearer good")
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestRoleEndpointAllowsAdmin(t *testing.T) {
	service := &fakeAuthService{
		authUser:       auth.User{ID: 1, Role: auth.RoleAdmin, IsActive: true},
		updateRoleUser: auth.User{ID: 7, Role: auth.RoleStorekeeper, IsActive: true},
	}
	router := authTestRouter(service)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/api/users/7/role", strings.NewReader(`{"role":"storekeeper"}`))
	request.Header.Set("Authorization", "Bearer good")
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if service.lastRole != auth.RoleStorekeeper {
		t.Fatalf("role = %q", service.lastRole)
	}
}

func TestAuthMiddlewareMapsUnexpectedErrorTo500(t *testing.T) {
	router := authTestRouter(&fakeAuthService{authErr: errors.New("database down")})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	request.Header.Set("Authorization", "Bearer good")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}
