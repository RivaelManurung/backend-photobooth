package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	
	// Actor (who performed the action)
	UserID      *uint          `gorm:"index" json:"user_id"`
	User        *User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	ActorEmail  string         `json:"actor_email"`
	ActorName   string         `json:"actor_name"`
	ActorRole   string         `json:"actor_role"`
	
	// Action details
	Action      string         `gorm:"not null;index" json:"action"` // create, update, delete, login, etc
	Resource    string         `gorm:"not null;index" json:"resource"` // user, template, photo, order, etc
	ResourceID  string         `gorm:"index" json:"resource_id"`
	
	// Request details
	Method      string         `json:"method"` // GET, POST, PUT, DELETE
	Path        string         `json:"path"`
	IPAddress   string         `gorm:"index" json:"ip_address"`
	UserAgent   string         `json:"user_agent"`
	
	// Changes
	OldValues   string         `gorm:"type:jsonb" json:"old_values,omitempty"`
	NewValues   string         `gorm:"type:jsonb" json:"new_values,omitempty"`
	
	// Status
	Status      string         `json:"status"` // success, failed
	ErrorMessage string        `json:"error_message,omitempty"`
	
	// Additional metadata
	Metadata    string         `gorm:"type:jsonb" json:"metadata,omitempty"`
	Duration    int64          `json:"duration"` // Request duration in milliseconds
}

// CreateAuditLog creates a new audit log entry
func CreateAuditLog(db *gorm.DB, log *AuditLog) error {
	return db.Create(log).Error
}

// GetAuditLogs retrieves audit logs with filters
func GetAuditLogs(db *gorm.DB, filters map[string]interface{}, limit, offset int) ([]AuditLog, int64, error) {
	var logs []AuditLog
	var total int64
	
	query := db.Model(&AuditLog{}).Preload("User")
	
	// Apply filters
	if userID, ok := filters["user_id"]; ok {
		query = query.Where("user_id = ?", userID)
	}
	if action, ok := filters["action"]; ok {
		query = query.Where("action = ?", action)
	}
	if resource, ok := filters["resource"]; ok {
		query = query.Where("resource = ?", resource)
	}
	if ipAddress, ok := filters["ip_address"]; ok {
		query = query.Where("ip_address = ?", ipAddress)
	}
	if dateFrom, ok := filters["date_from"]; ok {
		query = query.Where("created_at >= ?", dateFrom)
	}
	if dateTo, ok := filters["date_to"]; ok {
		query = query.Where("created_at <= ?", dateTo)
	}
	
	// Count total
	query.Count(&total)
	
	// Get logs
	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&logs).Error
	
	return logs, total, err
}

// GetUserAuditTrail gets audit trail for specific user
func GetUserAuditTrail(db *gorm.DB, userID uint, limit int) ([]AuditLog, error) {
	var logs []AuditLog
	err := db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

// GetResourceAuditTrail gets audit trail for specific resource
func GetResourceAuditTrail(db *gorm.DB, resource, resourceID string, limit int) ([]AuditLog, error) {
	var logs []AuditLog
	err := db.Where("resource = ? AND resource_id = ?", resource, resourceID).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

// SetOldValues sets old values as JSON
func (a *AuditLog) SetOldValues(values interface{}) error {
	jsonData, err := json.Marshal(values)
	if err != nil {
		return err
	}
	a.OldValues = string(jsonData)
	return nil
}

// SetNewValues sets new values as JSON
func (a *AuditLog) SetNewValues(values interface{}) error {
	jsonData, err := json.Marshal(values)
	if err != nil {
		return err
	}
	a.NewValues = string(jsonData)
	return nil
}

// SetMetadata sets metadata as JSON
func (a *AuditLog) SetMetadata(metadata interface{}) error {
	jsonData, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	a.Metadata = string(jsonData)
	return nil
}

// GetOldValues gets old values from JSON
func (a *AuditLog) GetOldValues() (map[string]interface{}, error) {
	var values map[string]interface{}
	if a.OldValues == "" {
		return values, nil
	}
	err := json.Unmarshal([]byte(a.OldValues), &values)
	return values, err
}

// GetNewValues gets new values from JSON
func (a *AuditLog) GetNewValues() (map[string]interface{}, error) {
	var values map[string]interface{}
	if a.NewValues == "" {
		return values, nil
	}
	err := json.Unmarshal([]byte(a.NewValues), &values)
	return values, err
}

// GetMetadata gets metadata from JSON
func (a *AuditLog) GetMetadata() (map[string]interface{}, error) {
	var metadata map[string]interface{}
	if a.Metadata == "" {
		return metadata, nil
	}
	err := json.Unmarshal([]byte(a.Metadata), &metadata)
	return metadata, err
}
