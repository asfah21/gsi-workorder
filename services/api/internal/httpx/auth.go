package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"gsi-workorder/internal/auth"
)

const authUserContextKey = "auth.user"

// AuthService is the subset of auth.Service used by HTTP handlers. Keeping it
// as an interface makes handler and middleware tests independent of PostgreSQL
// and Odoo.
type AuthService interface {
	Login(ctx context.Context, nik, password string) (auth.LoginResult, error)
	Refresh(ctx context.Context, refreshToken string) (auth.LoginResult, error)
	Authenticate(ctx context.Context, accessToken string) (auth.User, error)
	UpdateRole(ctx context.Context, userID int64, role string) (auth.User, error)
}

type AuthHandler struct {
	service AuthService
	logger  *slog.Logger
}

func NewAuthHandler(service AuthService, logger *slog.Logger) *AuthHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &AuthHandler{service: service, logger: logger}
}

type loginRequest struct {
	NIK      string `json:"nik"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type updateRoleRequest struct {
	Role string `json:"role"`
}

type userResponse struct {
	ID       int64  `json:"id"`
	NIK      string `json:"nik"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	IsActive bool   `json:"is_active"`
}

type tokenResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	TokenType    string       `json:"token_type"`
	ExpiresIn    int64        `json:"expires_in"`
	ExpiresAt    time.Time    `json:"expires_at"`
	User         userResponse `json:"user"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var request loginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeAuthError(c, http.StatusBadRequest, "invalid_request", "Body login tidak valid")
		return
	}

	result, err := h.service.Login(c.Request.Context(), request.NIK, request.Password)
	if err != nil {
		h.writeServiceError(c, err, "login")
		return
	}
	writeTokenResponse(c, result)
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var request refreshRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeAuthError(c, http.StatusBadRequest, "invalid_request", "Body refresh tidak valid")
		return
	}

	result, err := h.service.Refresh(c.Request.Context(), request.RefreshToken)
	if err != nil {
		h.writeServiceError(c, err, "refresh")
		return
	}
	writeTokenResponse(c, result)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	// Stateless JWT: the client discards its tokens. A server-side denylist is
	// intentionally not introduced in M1A.
	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) Me(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		writeAuthError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid")
		return
	}
	c.JSON(http.StatusOK, newUserResponse(user))
}

func (h *AuthHandler) UpdateRole(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || userID <= 0 {
		writeAuthError(c, http.StatusBadRequest, "invalid_user_id", "ID user tidak valid")
		return
	}

	var request updateRoleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeAuthError(c, http.StatusBadRequest, "invalid_request", "Body role tidak valid")
		return
	}

	user, err := h.service.UpdateRole(c.Request.Context(), userID, strings.TrimSpace(request.Role))
	if err != nil {
		h.writeServiceError(c, err, "update_role")
		return
	}
	if actor, ok := currentUser(c); ok {
		h.logger.Info("user role updated", "actor_id", actor.ID, "target_id", userID, "role", user.Role)
	}
	c.JSON(http.StatusOK, newUserResponse(user))
}

func (h *AuthHandler) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := bearerToken(c.Request)
		if !ok {
			c.Header("WWW-Authenticate", "Bearer")
			writeAuthError(c, http.StatusUnauthorized, "unauthorized", "Token akses diperlukan")
			c.Abort()
			return
		}

		user, err := h.service.Authenticate(c.Request.Context(), token)
		if err != nil {
			switch {
			case errors.Is(err, auth.ErrUserInactive):
				writeAuthError(c, http.StatusForbidden, "user_inactive", "User tidak aktif")
			case errors.Is(err, auth.ErrInvalidToken):
				c.Header("WWW-Authenticate", "Bearer")
				writeAuthError(c, http.StatusUnauthorized, "unauthorized", "Token akses tidak valid")
			default:
				h.logger.Error("authentication failed", "error", err)
				writeAuthError(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan internal")
			}
			c.Abort()
			return
		}

		c.Set(authUserContextKey, user)
		c.Next()
	}
}

func (h *AuthHandler) RequireRoles(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(c *gin.Context) {
		user, ok := currentUser(c)
		if !ok {
			writeAuthError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid")
			c.Abort()
			return
		}
		if _, ok := allowed[user.Role]; !ok {
			writeAuthError(c, http.StatusForbidden, "forbidden", "Role tidak memiliki akses")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (h *AuthHandler) writeServiceError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, auth.ErrValidation):
		writeAuthError(c, http.StatusBadRequest, "validation_error", "NIK dan password wajib diisi")
	case errors.Is(err, auth.ErrInvalidCredentials):
		writeAuthError(c, http.StatusUnauthorized, "invalid_credentials", "NIK atau password salah")
	case errors.Is(err, auth.ErrUserNotProvisioned):
		writeAuthError(c, http.StatusForbidden, "user_not_provisioned", "User belum tersedia di aplikasi; hubungi admin")
	case errors.Is(err, auth.ErrUserInactive):
		writeAuthError(c, http.StatusForbidden, "user_inactive", "User tidak aktif")
	case errors.Is(err, auth.ErrRefreshTokenMissing):
		writeAuthError(c, http.StatusBadRequest, "refresh_token_required", "Refresh token wajib diisi")
	case errors.Is(err, auth.ErrInvalidToken):
		writeAuthError(c, http.StatusUnauthorized, "unauthorized", "Token tidak valid")
	case errors.Is(err, auth.ErrInvalidRole):
		writeAuthError(c, http.StatusUnprocessableEntity, "invalid_role", "Role tidak valid")
	case errors.Is(err, auth.ErrUserNotFound):
		writeAuthError(c, http.StatusNotFound, "user_not_found", "User tidak ditemukan")
	case errors.Is(err, auth.ErrOdooUnavailable):
		writeAuthError(c, http.StatusServiceUnavailable, "odoo_unavailable", "Layanan autentikasi Odoo tidak tersedia")
	default:
		h.logger.Error("auth operation failed", "operation", operation, "error", err)
		writeAuthError(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan internal")
	}
}

func writeTokenResponse(c *gin.Context, result auth.LoginResult) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.JSON(http.StatusOK, newTokenResponse(result))
}

func newTokenResponse(result auth.LoginResult) tokenResponse {
	return tokenResponse{
		AccessToken:  result.Tokens.AccessToken,
		RefreshToken: result.Tokens.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    result.Tokens.ExpiresIn,
		ExpiresAt:    result.Tokens.ExpiresAt,
		User:         newUserResponse(result.User),
	}
}

func newUserResponse(user auth.User) userResponse {
	return userResponse{
		ID:       user.ID,
		NIK:      user.NIK,
		Name:     user.Name,
		Email:    user.Email,
		Role:     user.Role,
		IsActive: user.IsActive,
	}
}

func currentUser(c *gin.Context) (auth.User, bool) {
	value, exists := c.Get(authUserContextKey)
	if !exists {
		return auth.User{}, false
	}
	user, ok := value.(auth.User)
	return user, ok
}

func bearerToken(request *http.Request) (string, bool) {
	parts := strings.Fields(request.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") || strings.TrimSpace(parts[1]) == "" {
		return "", false
	}
	return parts[1], true
}

func writeAuthError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}
