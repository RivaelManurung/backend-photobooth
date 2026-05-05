package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"backendphotobooth/utils"
)

// ErrorResponse represents a standardized error format
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// RecoveryMiddleware handles panics and returns a 500 error
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log the error and stack trace
				utils.Logger.Error("panic recovered",
					zap.Any("error", err),
					zap.String("stack", string(debug.Stack())),
				)

				c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{
					Error:   "Internal Server Error",
					Message: "An unexpected error occurred. Please try again later.",
				})
			}
		}()
		c.Next()
	}
}

// ErrorHandlerMiddleware provides a unified way to handle domain errors
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if there are any errors in the context
		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			
			// If status is still 200, it means it wasn't set, default to 400
			status := c.Writer.Status()
			if status == http.StatusOK {
				status = http.StatusBadRequest
			}

			c.JSON(status, ErrorResponse{
				Error:   http.StatusText(status),
				Message: err.Error(),
			})
		}
	}
}
