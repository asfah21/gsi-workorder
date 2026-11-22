package httpx

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"gsi-workorder/internal/auth"
	"gsi-workorder/internal/inventory"
	"gsi-workorder/internal/material"
	"gsi-workorder/internal/spareparts"
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
	WorkOrders    WorkOrderService
	Users         UserDirectoryService
	PM            PMService
	Spareparts    spareparts.Service
	Inventory     inventory.Service
	Materials     material.Service
	Reporting     ReportingService
	AppLocation   *time.Location
	UploadDir     string
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

	var location *time.Location
	if m2 != nil {
		location = m2.AppLocation
	}
	var unitHandler *UnitHandler
	if unitService != nil {
		unitHandler = NewUnitHandler(unitService, logger, location)
		authorized.GET("/unit-models", unitHandler.ListUnitModels)
		authorized.GET("/engine-models", unitHandler.ListEngineModels)
		authorized.GET("/units", unitHandler.ListUnits)
		authorized.GET("/units/:id", unitHandler.GetUnit)
		authorized.GET("/units/:id/meter-readings", unitHandler.ListMeterReadings)
		authorized.GET("/units/:id/history", unitHandler.ListMaintenanceHistory)
		authorized.GET("/units/:id/components", unitHandler.ListUnitComponents)
		authorized.GET("/units/:id/component-intervals", unitHandler.ListComponentIntervals)
		authorized.GET("/units/:id/downtime", unitHandler.UnitDowntime)
		authorized.GET("/components", unitHandler.ListComponents)
		if m2 == nil {
			authorized.GET("/dashboard/summary", unitHandler.DashboardSummary)
		}

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
		if m2.Reporting != nil {
			reportingHandler := NewReportingHandler(m2.Reporting, location, logger)
			authorized.GET("/dashboard/summary", reportingHandler.Summary)
			authorized.GET("/dashboard/pm", reportingHandler.PM)
			authorized.GET("/dashboard/material", reportingHandler.Material)
			authorized.GET("/dashboard/inventory", reportingHandler.Inventory)
			authorized.GET("/dashboard/maintenance", reportingHandler.Maintenance)
			authorized.GET("/reports/:name", reportingHandler.Report)
		} else if unitHandler != nil {
			authorized.GET("/dashboard/summary", unitHandler.DashboardSummary)
		}
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
		if m2.Users != nil {
			userHandler := NewUserDirectoryHandler(m2.Users, logger)
			userReaders := authorized.Group("")
			userReaders.Use(handler.RequireRoles(auth.RoleAdmin, auth.RolePlanner))
			userReaders.GET("/users", userHandler.List)
		}
		if m2.WorkOrders != nil {
			workOrderHandler := NewWorkOrderHandler(m2.WorkOrders, logger, m2.UploadDir)
			authorized.GET("/workorders", workOrderHandler.List)
			authorized.GET("/workorders/:id", workOrderHandler.Get)
			authorized.GET("/workorders/:id/attachments/:attachment_id", workOrderHandler.DownloadAttachment)

			workOrderCreators := authorized.Group("")
			workOrderCreators.Use(handler.RequireRoles(auth.RoleAdmin, auth.RolePlanner))
			workOrderCreators.POST("/workorders", workOrderHandler.Create)

			workOrderWriters := authorized.Group("")
			workOrderWriters.Use(handler.RequireRoles(auth.RoleAdmin, auth.RolePlanner, auth.RoleMechanic))
			workOrderWriters.PATCH("/workorders/:id", workOrderHandler.Update)
			workOrderWriters.PATCH("/workorders/:id/checklist/:item_id", workOrderHandler.UpdateChecklistItem)
			workOrderWriters.POST("/workorders/:id/updates", workOrderHandler.AddUpdate)
			workOrderWriters.POST("/workorders/:id/attachments", workOrderHandler.UploadAttachment)
		}
		if m2.PM != nil {
			pmHandler := NewPMHandler(m2.PM, logger)
			authorized.GET("/pm/plans", pmHandler.ListPlans)
			authorized.GET("/pm/plans/:id", pmHandler.GetPlan)
			authorized.GET("/pm/plans/:id/checklist-items", pmHandler.ListChecklistItems)
			authorized.GET("/pm/schedules", pmHandler.ListSchedules)
			authorized.GET("/pm/schedules/:id", pmHandler.GetSchedule)
			authorized.GET("/pm/due", pmHandler.Due)
			authorized.GET("/pm/compliance", pmHandler.Compliance)

			pmWriters := authorized.Group("")
			pmWriters.Use(handler.RequireRoles(auth.RoleAdmin, auth.RolePlanner))
			pmWriters.POST("/pm/plans", pmHandler.CreatePlan)
			pmWriters.PATCH("/pm/plans/:id", pmHandler.UpdatePlan)
			pmWriters.POST("/pm/plans/:id/status", pmHandler.SetPlanStatus)
			pmWriters.POST("/pm/plans/:id/checklist-items", pmHandler.CreateChecklistItem)
			pmWriters.POST("/pm/schedules/:id/generate-wo", pmHandler.GenerateWO)
			pmWriters.POST("/pm/schedules/:id/override-due", pmHandler.OverrideDue)
			pmWriters.POST("/pm/schedules/:id/skip", pmHandler.Skip)
			pmWriters.POST("/pm/schedules/:id/cancel", pmHandler.Cancel)
			authorized.GET("/pm/plans/:id/materials", pmHandler.ListPlanMaterials)
			pmWriters.POST("/pm/plans/:id/materials", pmHandler.AddPlanMaterial)
			pmWriters.PATCH("/pm/plans/:id/materials/:material_id", pmHandler.UpdatePlanMaterial)
			pmWriters.DELETE("/pm/plans/:id/materials/:material_id", pmHandler.DeletePlanMaterial)
		}
		if m2.Spareparts != nil {
			sparepartHandler := NewSparepartHandler(m2.Spareparts, logger)
			authorized.GET("/spareparts", sparepartHandler.List)
			authorized.GET("/spareparts/search", sparepartHandler.Search)
			authorized.GET("/spareparts/:id", sparepartHandler.Get)
			authorized.GET("/spareparts/:id/numbers", sparepartHandler.ListNumbers)
			authorized.GET("/spareparts/:id/alternatives", sparepartHandler.ListAlternatives)
			authorized.GET("/spareparts/:id/compatibility", sparepartHandler.ListCompatibility)

			sparepartWriters := authorized.Group("")
			sparepartWriters.Use(handler.RequireRoles(auth.RoleAdmin, auth.RolePlanner))
			sparepartWriters.POST("/spareparts", sparepartHandler.Create)
			sparepartWriters.PATCH("/spareparts/:id", sparepartHandler.Update)
			sparepartWriters.POST("/spareparts/:id/numbers", sparepartHandler.CreateNumber)
			sparepartWriters.PATCH("/spareparts/:id/numbers/:number_id", sparepartHandler.UpdateNumber)
			sparepartWriters.POST("/spareparts/:id/alternatives", sparepartHandler.CreateAlternative)
			sparepartWriters.POST("/spareparts/:id/alternatives/:alternative_id/verify", sparepartHandler.VerifyAlternative)
			sparepartWriters.POST("/spareparts/:id/compatibility", sparepartHandler.CreateCompatibility)
			sparepartWriters.POST("/spareparts/:id/compatibility/:compat_id/verify", sparepartHandler.VerifyCompatibility)
		}
		if m2.Inventory != nil {
			inventoryHandler := NewInventoryHandler(m2.Inventory, m2.Materials, logger)
			authorized.GET("/warehouses", inventoryHandler.ListWarehouses)
			authorized.GET("/warehouses/:id/locations", inventoryHandler.ListWarehouseLocations)
			authorized.GET("/warehouse-locations", inventoryHandler.ListAllLocations)
			authorized.GET("/inventory/balances", inventoryHandler.ListBalances)
			authorized.GET("/inventory/movements", inventoryHandler.ListMovements)
			authorized.GET("/inventory/receipts", inventoryHandler.ListReceipts)
			authorized.GET("/inventory/issues", inventoryHandler.ListIssues)
			authorized.GET("/inventory/returns", inventoryHandler.ListReturns)
			authorized.GET("/inventory/transfers", inventoryHandler.ListTransfers)
			authorized.GET("/inventory/adjustments", inventoryHandler.ListAdjustments)
			authorized.GET("/inventory/transactions/:id", inventoryHandler.GetTransaction)
			authorized.GET("/inventory/opname/sessions", inventoryHandler.ListOpnameSessions)
			authorized.GET("/inventory/opname/sessions/:id", inventoryHandler.GetOpnameSession)
			authorized.GET("/inventory/stock-policies", inventoryHandler.ListPolicies)

			inventoryMasters := authorized.Group("")
			inventoryMasters.Use(handler.RequireRoles(auth.RoleAdmin, auth.RolePlanner))
			inventoryMasters.POST("/warehouses", inventoryHandler.CreateWarehouse)
			inventoryMasters.PATCH("/warehouses/:id", inventoryHandler.UpdateWarehouse)
			inventoryMasters.POST("/warehouses/:id/locations", inventoryHandler.CreateLocation)
			inventoryMasters.PATCH("/warehouse-locations/:id", inventoryHandler.UpdateLocation)
			inventoryMasters.POST("/inventory/stock-policies", inventoryHandler.UpsertPolicy)
			inventoryMasters.PATCH("/inventory/stock-policies/:id", inventoryHandler.UpdatePolicy)
			inventoryMasters.POST("/inventory/adjustments", inventoryHandler.PostAdjustment)
			inventoryMasters.POST("/inventory/transactions/:id/reverse", inventoryHandler.Reverse)
			inventoryMasters.POST("/inventory/opname/sessions/:id/post", inventoryHandler.PostOpnameSession)
			inventoryMasters.POST("/inventory/opname/sessions/:id/cancel", inventoryHandler.CancelOpnameSession)
			inventoryMasters.GET("/inventory/reconciliation", inventoryHandler.Reconcile)

			stockWriters := authorized.Group("")
			stockWriters.Use(handler.RequireRoles(auth.RoleAdmin, auth.RoleStorekeeper))
			stockWriters.POST("/inventory/receipts", inventoryHandler.PostReceipt)
			stockWriters.POST("/inventory/issues", inventoryHandler.PostIssue)
			stockWriters.POST("/inventory/returns", inventoryHandler.PostReturn)
			stockWriters.POST("/inventory/transfers", inventoryHandler.PostTransfer)
			stockWriters.POST("/inventory/opname/sessions", inventoryHandler.CreateOpnameSession)
			stockWriters.PATCH("/inventory/opname/sessions/:id/items", inventoryHandler.CountOpnameSession)
		}
		if m2.Materials != nil {
			materialHandler := NewMaterialHandler(m2.Materials, logger)
			authorized.GET("/material-requirements", materialHandler.ListRequirements)
			authorized.GET("/material-requirements/:id", materialHandler.GetRequirement)
			authorized.GET("/material-requirements/:id/updates", materialHandler.ListUpdates)
			authorized.GET("/material-requirements/:id/usages", materialHandler.ListUsages)
			authorized.GET("/inventory/reservations", materialHandler.ListReservations)

			materialWriters := authorized.Group("")
			materialWriters.Use(handler.RequireRoles(auth.RoleAdmin, auth.RolePlanner))
			materialWriters.POST("/material-requirements", materialHandler.CreateRequirement)
			materialWriters.PATCH("/material-requirements/:id", materialHandler.UpdateRequirement)
			materialWriters.POST("/material-requirements/:id/cancel", materialHandler.Cancel)

			materialStockWriters := authorized.Group("")
			materialStockWriters.Use(handler.RequireRoles(auth.RoleAdmin, auth.RolePlanner, auth.RoleStorekeeper))
			materialStockWriters.POST("/material-requirements/:id/check-stock", materialHandler.CheckStock)
			materialStockWriters.POST("/material-requirements/:id/reserve", materialHandler.Reserve)
			materialStockWriters.POST("/inventory/reservations", materialHandler.CreateReservation)
			materialStockWriters.POST("/inventory/reservations/:id/release", materialHandler.ReleaseReservation)

			materialIssueWriters := authorized.Group("")
			materialIssueWriters.Use(handler.RequireRoles(auth.RoleAdmin, auth.RoleStorekeeper))
			materialIssueWriters.POST("/material-requirements/:id/issue", materialHandler.Issue)
			materialIssueWriters.POST("/material-requirements/:id/return", materialHandler.Return)
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
