package services

import (
	"backendphotobooth/models"
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type AnalyticsSink interface {
	Track(ctx context.Context, event models.AnalyticsEvent) error
}

type AnalyticsService struct {
	db *gorm.DB
}

func NewAnalyticsService(db *gorm.DB) *AnalyticsService {
	return &AnalyticsService{db: db}
}

func (s *AnalyticsService) Track(ctx context.Context, event models.AnalyticsEvent) error {
	if s == nil || s.db == nil {
		return nil
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	return s.db.WithContext(ctx).Create(&event).Error
}

func (s *AnalyticsService) TrackWithMetadata(ctx context.Context, eventName string, actorUserID *uint, metadata map[string]interface{}) error {
	encoded, _ := json.Marshal(metadata)
	return s.Track(ctx, models.AnalyticsEvent{
		EventName:   eventName,
		ActorUserID: actorUserID,
		Metadata:    string(encoded),
	})
}

func (s *AnalyticsService) AggregateDailyStats(ctx context.Context, date time.Time) error {
	// Raw events are stored first. Aggregation is intentionally conservative and can
	// be expanded without changing event ingestion.
	return nil
}

func (s *AnalyticsService) GetDashboardStats(ctx context.Context, start, end time.Time) (map[string]int64, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Model(&models.AnalyticsEvent{}).
		Where("created_at BETWEEN ? AND ?", start, end).
		Count(&count).Error
	return map[string]int64{"events": count}, err
}
