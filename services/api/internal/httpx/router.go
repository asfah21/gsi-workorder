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
	var authService AuthService
	if len(authServices) > 0 {
		authService = authServices[0]
	}
	return newRouter(logger, checker, authService, nil)
}

// NewRouterWithUnits registers M1C unit-master routes in addition to auth.
func NewRouterWithUnits(logger *slog.Logger, checker Checker, authService AuthService, unitService UnitService) *gin.Engine {
	return newRouter(logger, checker, authService, unitService)
}

func newRouter(logger *slog.Logger, checker Checker, authService AuthService, unitService UnitService) *gin.Engine {
	router := gin.New()
	_ = router.SetTrustedProxies(nil)
	router.Use(RequestLogger(logger), gin.Recovery())
	router.GET("/healthz", HealthHandler(checker))

	if authService == nil {
		return router
	}

	handler := NewAuthHandler(authService, logger)
	router.POST("/api/auth/login", handler.Login)
	router.POST("/api/auth/refresh", handler.Refresh)
	router.POST("/api/auth/logout", handler.Logout)

	authorized := router.Group("/api")
	authorized.Use(handler.RequireAuth())
	authorized.GET("/me", handler.Me)

	admin := authorized.Group("")
	admin.Use(handler.RequireRoles(auth.RoleAdmin))
	admin.PATCH("/users/:id/role", handler.UpdateRole)

	if unitService != nil {
		unitHandler := NewUnitHandler(unitService, logger)
		authorized.GET("/unit-models", unitHandler.ListUnitModels)
		authorized.GET("/engine-models", unitHandler.ListEngineModels)
		authorized.GET("/units", unitHandler.ListUnits)
		authorized.GET("/units/:id", unitHandler.GetUnit)
		authorized.GET("/units/:id/meter-readings", unitHandler.ListMeterReadings)
		authorized.GET("/dashboard/summary", unitHandler.DashboardSummary)

		unitWriters := authorized.Group("")
		unitWriters.Use(handler.RequireRoles(auth.RoleAdmin, auth.RolePlanner))
		unitWriters.POST("/unit-models", unitHandler.CreateUnitModel)
		unitWriters.PATCH("/unit-models/:id", unitHandler.UpdateUnitModel)
		unitWriters.POST("/engine-models", unitHandler.CreateEngineModel)
		unitWriters.PATCH("/engine-models/:id", unitHandler.UpdateEngineModel)
		unitWriters.POST("/units", unitHandler.CreateUnit)
		unitWriters.PATCH("/units/:id", unitHandler.UpdateUnit)

		meterWriters := authorized.Group("")
		meterWriters.Use(handler.RequireRoles(auth.RoleAdmin, auth.RolePlanner, auth.RoleMechanic))
		meterWriters.POST("/units/:id/meter-readings", unitHandler.AddMeterReading)
	}

	return router
}
