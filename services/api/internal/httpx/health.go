package httpx

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const healthCheckTimeout = 2 * time.Second

// Checker is the dependency required by the health endpoint. It is implemented
// by CheckFunc in production and by tests.
type Checker interface {
	Check(ctx context.Context) error
}

// CheckFunc adapts a function to the Checker interface.
type CheckFunc func(ctx context.Context) error

func (f CheckFunc) Check(ctx context.Context) error {
	return f(ctx)
}

// HealthHandler reports liveness and database connectivity.
func HealthHandler(checker Checker) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), healthCheckTimeout)
		defer cancel()

		if err := checker.Check(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":   "degraded",
				"database": "down",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   "ok",
			"database": "up",
		})
	}
}
