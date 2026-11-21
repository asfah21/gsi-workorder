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
	return newRouter(logger, checker, authService, nil, nil)
}

// NewRouterWithUnits registers M1C unit-master routes in addition to auth.
func NewRouterWithUnits(logger *slog.Logger, checker Checker, authService AuthService, unitService UnitService) *gin.Engine {
	return newRouter(logger, checker, authService, unitService, nil)
}

type M2Services struct {
	Candidates    CandidateService
	Notifications NotificationService
	Settings      SettingsService
	P2H           P2HService
}

// NewRouterWithM2 registers M2 P2H/candidate/notification/settings routes.
func NewRouterWithM2(
	logger *slog.Logger,
	checker Checker,
	authService AuthService,
	unitService UnitService,
	services M2Services,
) *gin.Engine {
	return newRouter(logger, checker, authService, unitService, &services)
}

func newRouter(logger *slog.Logger, checker Checker, authService AuthService, unitService UnitService, m2 *M2Services) *gin.Engine {
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

	if m2 != nil {
		if m2.Candidates != nil {
			candidateHandler := NewCandidateHandler(m2.Candidates, logger)
			authorized.GET("/wo-candidates", candidateHandler.List)
			authorized.GET("/wo-candidates/:id", candidateHandler.Get)

			candidateWriters := authorized.Group("")
			candidateWriters.Use(handler.RequireRoles(auth.RoleAdmin, auth.RolePlanner))
			candidateWriters.POST("/wo-candidates/:id/approve", candidateHandler.Approve)
			candidateWriters.POST("/wo-candidates/:id/skip", candidateHandler.Skip)
			candidateWriters.POST("/wo-candidates/bulk", candidateHandler.Bulk)
		}
		if m2.P2H != nil {
			p2hHandler := NewP2HHandler(m2.P2H, logger)
			authorized.GET("/p2h/reports", p2hHandler.ListReports)
			authorized.GET("/p2h/reports/:id", p2hHandler.GetReport)
		}
		if m2.Notifications != nil {
			notificationHandler := NewNotificationHandler(m2.Notifications, logger)
			authorized.GET("/notifications", notificationHandler.List)
			authorized.POST("/notifications/read", notificationHandler.MarkRead)
		}
		if m2.Settings != nil {
			settingsHandler := NewSettingsHandler(m2.Settings, logger)
			settingsGroup := authorized.Group("")
			settingsGroup.Use(handler.RequireRoles(auth.RoleAdmin))
			settingsGroup.GET("/settings", settingsHandler.Get)
			settingsGroup.PATCH("/settings", settingsHandler.Update)
		}
	}

	return router
}
