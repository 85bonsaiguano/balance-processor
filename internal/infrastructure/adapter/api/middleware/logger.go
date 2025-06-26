package middleware

import (
	"time"

	coreport "github.com/amirhossein-jamali/balance-processor/internal/domain/port/core"
	"github.com/gin-gonic/gin"
)

// Logger middleware logs incoming requests and their responses
func Logger(logger coreport.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		ip := c.ClientIP()

		// Process request
		c.Next()

		// Calculate request time
		latency := time.Since(start)
		statusCode := c.Writer.Status()

		// Log the request
		logger.Info("Request processed", map[string]any{
			"method":      method,
			"path":        path,
			"status":      statusCode,
			"latency_ms":  latency.Milliseconds(),
			"ip":          ip,
			"request_id":  c.GetHeader("X-Request-ID"),
			"user_agent":  c.Request.UserAgent(),
			"errors":      c.Errors.Errors(),
			"status_text": statusText(statusCode),
		})
	}
}

// statusText returns the text for the HTTP status code
func statusText(code int) string {
	switch {
	case code >= 100 && code < 200:
		return "Informational"
	case code >= 200 && code < 300:
		return "Success"
	case code >= 300 && code < 400:
		return "Redirect"
	case code >= 400 && code < 500:
		return "Client Error"
	default:
		return "Server Error"
	}
}
