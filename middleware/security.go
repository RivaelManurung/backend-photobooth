package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityMiddleware adds standard security headers to all responses
func SecurityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent page from being displayed in frames (Clickjacking protection)
		c.Header("X-Frame-Options", "DENY")
		
		// Prevent the browser from interpreting files as something else than declared
		c.Header("X-Content-Type-Options", "nosniff")
		
		// Enable XSS filtering in browsers
		c.Header("X-XSS-Protection", "1; mode=block")
		
		// Force HTTPS (HSTS) - only in production
		// c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		
		// Content Security Policy
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self' https:;")
		
		c.Next()
	}
}
