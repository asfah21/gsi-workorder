package httpx

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

// NewRouter builds the Milestone 0 HTTP router. Business routes are added in
// later milestones; only health checking exists now.
func NewRouter(logger *slog.Logger, checker Checker) *gin.Engine {
	router := gin.New()
	_ = router.SetTrustedProxies(nil)
	router.Use(RequestLogger(logger), gin.Recovery())
	router.GET("/healthz", HealthHandler(checker))
	return router
}
