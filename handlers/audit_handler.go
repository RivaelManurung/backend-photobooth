package handlers

import (
	"backendphotobooth/database"
	"backendphotobooth/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AuditHandler struct{}

func NewAuditHandler() *AuditHandler {
	return &AuditHandler{}
}

// GetAuditLogs returns audit logs with filters (admin only)
func (h *AuditHandler) GetAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset := (page - 1) * limit

	filters := make(map[string]interface{})

	// Apply filters
	if userID := c.Query("user_id"); userID != "" {
		filters["user_id"] = userID
	}
	if action := c.Query("action"); action != "" {
		filters["action"] = action
	}
	if resource := c.Query("resource"); resource != "" {
		filters["resource"] = resource
	}
	if ipAddress := c.Query("ip_address"); ipAddress != "" {
		filters["ip_address"] = ipAddress
	}
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		filters["date_from"] = dateFrom
	}
	if dateTo := c.Query("date_to"); dateTo != "" {
		filters["date_to"] = dateTo
	}

	logs, total, err := models.GetAuditLogs(database.DB, filters, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch audit logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetUserAuditTrail returns audit trail for specific user (admin only)
func (h *AuditHandler) GetUserAuditTrail(c *gin.Context) {
	userID := c.Param("user_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))

	var uid uint
	if id, err := strconv.ParseUint(userID, 10, 32); err == nil {
		uid = uint(id)
	}

	logs, err := models.GetUserAuditTrail(database.DB, uid, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch audit trail"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"logs":    logs,
		"total":   len(logs),
	})
}

// GetResourceAuditTrail returns audit trail for specific resource (admin only)
func (h *AuditHandler) GetResourceAuditTrail(c *gin.Context) {
	resource := c.Param("resource_type")
	if resource == "" {
		resource = c.Param("resource")
	}
	resourceID := c.Param("resource_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))

	logs, err := models.GetResourceAuditTrail(database.DB, resource, resourceID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch audit trail"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"resource":    resource,
		"resource_id": resourceID,
		"logs":        logs,
		"total":       len(logs),
	})
}

// GetAuditStats returns audit statistics (admin only)
func (h *AuditHandler) GetAuditStats(c *gin.Context) {
	var stats struct {
		TotalLogs     int64                    `json:"total_logs"`
		TodayLogs     int64                    `json:"today_logs"`
		FailedActions int64                    `json:"failed_actions"`
		TopActions    []map[string]interface{} `json:"top_actions"`
		TopResources  []map[string]interface{} `json:"top_resources"`
		TopUsers      []map[string]interface{} `json:"top_users"`
	}

	// Total logs
	database.DB.Model(&models.AuditLog{}).Count(&stats.TotalLogs)

	// Today's logs
	database.DB.Model(&models.AuditLog{}).
		Where("DATE(created_at) = CURRENT_DATE").
		Count(&stats.TodayLogs)

	// Failed actions
	database.DB.Model(&models.AuditLog{}).
		Where("status = ?", "failed").
		Count(&stats.FailedActions)

	// Top actions
	database.DB.Model(&models.AuditLog{}).
		Select("action, COUNT(*) as count").
		Group("action").
		Order("count DESC").
		Limit(10).
		Scan(&stats.TopActions)

	// Top resources
	database.DB.Model(&models.AuditLog{}).
		Select("resource, COUNT(*) as count").
		Group("resource").
		Order("count DESC").
		Limit(10).
		Scan(&stats.TopResources)

	// Top users
	database.DB.Model(&models.AuditLog{}).
		Select("actor_email, actor_name, COUNT(*) as count").
		Where("actor_email IS NOT NULL").
		Group("actor_email, actor_name").
		Order("count DESC").
		Limit(10).
		Scan(&stats.TopUsers)

	c.JSON(http.StatusOK, stats)
}

// ExportAuditLogs exports audit logs to CSV (admin only)
func (h *AuditHandler) ExportAuditLogs(c *gin.Context) {
	filters := make(map[string]interface{})

	// Apply filters
	if userID := c.Query("user_id"); userID != "" {
		filters["user_id"] = userID
	}
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		filters["date_from"] = dateFrom
	}
	if dateTo := c.Query("date_to"); dateTo != "" {
		filters["date_to"] = dateTo
	}

	logs, _, err := models.GetAuditLogs(database.DB, filters, 10000, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch audit logs"})
		return
	}

	// Set headers for CSV download
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=audit_logs.csv")

	// Write CSV header
	c.Writer.Write([]byte("Timestamp,User,Action,Resource,Resource ID,IP Address,Status\n"))

	// Write audit log data
	for _, log := range logs {
		line := []byte(
			log.CreatedAt.Format("2006-01-02 15:04:05") + "," +
				log.ActorEmail + "," +
				log.Action + "," +
				log.Resource + "," +
				log.ResourceID + "," +
				log.IPAddress + "," +
				log.Status + "\n",
		)
		c.Writer.Write(line)
	}
}
