package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	JobPhotoUploaded     = "photo.uploaded"
	JobPhotoProcess      = "photo.process"
	JobRenderTemplate    = "photo.render_template"
	JobGenerateThumbnail = "photo.generate_thumbnail"
	JobGenerateStrip     = "photo.generate_strip"
	JobCleanupFailed     = "photo.cleanup_failed"
	JobNotificationReady = "notification.photo_ready"
	defaultQueueName     = "photobooth:jobs"
)

type QueueJob struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Attempts  int             `json:"attempts"`
	CreatedAt time.Time       `json:"created_at"`
}

type PhotoProcessPayload struct {
	PhotoID uint `json:"photo_id"`
}

type QueueService struct {
	client *redis.Client
	queue  string
}

func NewQueueService(redisService *RedisService) *QueueService {
	if redisService == nil || redisService.Client() == nil {
		return nil
	}
	return &QueueService{client: redisService.Client(), queue: defaultQueueName}
}

func (q *QueueService) Enqueue(ctx context.Context, jobType string, payload interface{}) error {
	if q == nil || q.client == nil {
		return fmt.Errorf("queue is not configured")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	jobBody, err := json.Marshal(QueueJob{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Type:      jobType,
		Payload:   body,
		CreatedAt: time.Now(),
	})
	if err != nil {
		return err
	}
	return q.client.LPush(ctx, q.queue, jobBody).Err()
}

func (q *QueueService) EnqueuePhotoProcess(ctx context.Context, photoID uint) error {
	return q.Enqueue(ctx, JobPhotoProcess, PhotoProcessPayload{PhotoID: photoID})
}

func (q *QueueService) Dequeue(ctx context.Context, timeout time.Duration) (*QueueJob, error) {
	if q == nil || q.client == nil {
		return nil, fmt.Errorf("queue is not configured")
	}
	result, err := q.client.BRPop(ctx, timeout, q.queue).Result()
	if err != nil {
		return nil, err
	}
	if len(result) != 2 {
		return nil, fmt.Errorf("unexpected queue response")
	}
	var job QueueJob
	if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
		return nil, err
	}
	return &job, nil
}
