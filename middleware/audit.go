package middleware

import (
	"backendphotobooth/database"
	"backendphotobooth/models"
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func AuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		// Capture request body
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Wrap response writer to capture response
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		c.Next()

		// Only log mutations or security-sensitive actions
		method := c.Request.Method
		if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete {
			user, _ := GetCurrentUser(c)
			
			log := &models.AuditLog{
				Action:     method,
				Resource:   c.FullPath(),
				ResourceID: c.Param("id"),
				Method:     method,
				Path:       c.Request.URL.Path,
				IPAddress:  c.ClientIP(),
				UserAgent:  c.Request.UserAgent(),
				Status:     "success",
				Duration:   time.Since(start).Milliseconds(),
			}

			if user != nil {
				log.UserID = &user.ID
				log.ActorEmail = user.Email
				log.ActorName = user.Name
				log.ActorRole = user.Role
			} else {
				log.ActorName = "Guest"
			}

			if c.Writer.Status() >= 400 {
				log.Status = "failed"
				log.ErrorMessage = blw.body.String()
			}

			// Background save to avoid blocking response
			go models.CreateAuditLog(database.DB, log)
		}
	}
}
