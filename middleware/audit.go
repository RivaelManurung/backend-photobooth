package middleware

import (
	"backendphotobooth/database"
	"backendphotobooth/models"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// AuditMiddleware logs all requests for audit purposes
func AuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Get user if authenticated
		user, _ := GetCurrentUser(c)

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(startTime).Milliseconds()

		// Determine if this should be audited
		if shouldAudit(c) {
			auditLog := &models.AuditLog{
				Method:    c.Request.Method,
				Path:      c.Request.URL.Path,
				IPAddress: c.ClientIP(),
				UserAgent: c.Request.UserAgent(),
				Status:    getStatus(c.Writer.Status()),
				Duration:  duration,
			}

			// Set user info if authenticated
			if user != nil {
				auditLog.UserID = &user.ID
				auditLog.ActorEmail = user.Email
				auditLog.ActorName = user.Name
				auditLog.ActorRole = user.Role
			}

			// Determine action and resource from path
			action, resource := parseActionResource(c)
			auditLog.Action = action
			auditLog.Resource = resource

			// Get resource ID from params
			if id := c.Param("id"); id != "" {
				auditLog.ResourceID = id
			}

			// Store error if any
			if len(c.Errors) > 0 {
				auditLog.ErrorMessage = c.Errors.String()
			}

			// Save audit log asynchronously
			go func() {
				if err := models.CreateAuditLog(database.DB, auditLog); err != nil {
					fmt.Printf("Failed to create audit log: %v\n", err)
				}
			}()
		}
	}
}

// shouldAudit determines if request should be audited
func shouldAudit(c *gin.Context) bool {
	// Don't audit health checks and static files
	path := c.Request.URL.Path
	
	if path == "/health" || path == "/metrics" {
		return false
	}
	
	if len(path) > 8 && path[:8] == "/uploads" {
		return false
	}
	
	// Audit all API calls
	if len(path) > 4 && path[:4] == "/api" {
		return true
	}
	
	return false
}

// parseActionResource parses action and resource from request
func parseActionResource(c *gin.Context) (string, string) {
	method := c.Request.Method
	path := c.Request.URL.Path
	
	var action string
	switch method {
	case "GET":
		action = "read"
	case "POST":
		action = "create"
	case "PUT", "PATCH":
		action = "update"
	case "DELETE":
		action = "delete"
	default:
		action = "unknown"
	}
	
	// Extract resource from path
	// Example: /api/v1/templates/123 -> templates
	resource := "unknown"
	if len(path) > 8 {
		parts := splitPath(path)
		if len(parts) >= 3 {
			resource = parts[2] // After /api/v1/
		}
	}
	
	return action, resource
}

// splitPath splits path into parts
func splitPath(path string) []string {
	var parts []string
	current := ""
	
	for _, char := range path {
		if char == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	
	if current != "" {
		parts = append(parts, current)
	}
	
	return parts
}

// getStatus converts HTTP status code to string
func getStatus(statusCode int) string {
	if statusCode >= 200 && statusCode < 300 {
		return "success"
	}
	return "failed"
}

// AuditAction manually creates an audit log for specific actions
func AuditAction(c *gin.Context, action, resource, resourceID string, oldValues, newValues interface{}) {
	user, _ := GetCurrentUser(c)
	
	auditLog := &models.AuditLog{
		Method:     c.Request.Method,
		Path:       c.Request.URL.Path,
		IPAddress:  c.ClientIP(),
		UserAgent:  c.Request.UserAgent(),
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Status:     "success",
	}
	
	if user != nil {
		auditLog.UserID = &user.ID
		auditLog.ActorEmail = user.Email
		auditLog.ActorName = user.Name
		auditLog.ActorRole = user.Role
	}
	
	if oldValues != nil {
		auditLog.SetOldValues(oldValues)
	}
	
	if newValues != nil {
		auditLog.SetNewValues(newValues)
	}
	
	go models.CreateAuditLog(database.DB, auditLog)
}
