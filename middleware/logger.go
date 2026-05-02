package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// LoggerMiddleware logs request details
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(startTime)

		// Get status code
		statusCode := c.Writer.Status()

		// Log format
		log.Printf(
			"[%s] %s %s %d %s %s",
			c.Request.Method,
			c.Request.RequestURI,
			c.ClientIP(),
			statusCode,
			latency,
			c.Errors.String(),
		)
	}
}
