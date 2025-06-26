package middleware

import (
	"net/http"

	domainerr "github.com/amirhossein-jamali/balance-processor/internal/domain/error"
	coreport "github.com/amirhossein-jamali/balance-processor/internal/domain/port/core"
	"github.com/amirhossein-jamali/balance-processor/internal/infrastructure/adapter/api/dto"
	"github.com/gin-gonic/gin"
)

// ErrorHandler middleware recovers from panics and returns appropriate error responses
func ErrorHandler(logger coreport.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log the error with stack trace
				logger.Error("Panic recovered in API request", map[string]any{
					"error":      err,
					"path":       c.Request.URL.Path,
					"method":     c.Request.Method,
					"client_ip":  c.ClientIP(),
					"request_id": c.GetHeader("X-Request-ID"),
					"user_agent": c.Request.UserAgent(),
				})

				// Return a 500 Internal Server Error response
				c.AbortWithStatusJSON(http.StatusInternalServerError, dto.ErrorResponse{
					Code:    domainerr.ErrorCode(domainerr.ErrInternalServer),
					Message: "Internal server error",
				})
			}
		}()

		c.Next()
	}
}
