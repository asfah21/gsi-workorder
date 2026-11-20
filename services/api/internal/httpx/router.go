package httpx

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"gsi-workorder/internal/auth"
)

// NewRouter builds the HTTP router. M0 only had health checking; the optional
// auth service registers the M1A authentication/RBAC routes. Keeping auth
// optional lets the existing M0 health tests remain focused.
func NewRouter(logger *slog.Logger, checker Checker, authServices ...AuthService) *gin.Engine {
	router := gin.New()
	_ = router.SetTrustedProxies(nil)
	router.Use(RequestLogger(logger), gin.Recovery())
	router.GET("/healthz", HealthHandler(checker))

	if len(authServices) > 0 && authServices[0] != nil {
		handler := NewAuthHandler(authServices[0], logger)
		router.POST("/api/auth/login", handler.Login)
		router.POST("/api/auth/refresh", handler.Refresh)
		router.POST("/api/auth/logout", handler.Logout)

		authorized := router.Group("/api")
		authorized.Use(handler.RequireAuth())
		authorized.GET("/me", handler.Me)

		admin := authorized.Group("")
		admin.Use(handler.RequireRoles(auth.RoleAdmin))
		admin.PATCH("/users/:id/role", handler.UpdateRole)
	}

	return router
}
