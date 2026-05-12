package main

import (
	"backendphotobooth/config"
	"backendphotobooth/database"
	"backendphotobooth/models"
	"backendphotobooth/services"
	"backendphotobooth/utils"
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func main() {
	cfg := config.LoadConfig()
	utils.InitLogger(cfg.Server.Environment)
	defer utils.Logger.Sync()

	if err := database.InitDatabase(cfg); err != nil {
		log.Fatal("failed to initialize database:", err)
	}

	redisService := services.NewRedisService(cfg)
	queue := services.NewQueueService(redisService)
	if queue == nil {
		log.Fatal("redis queue is not configured")
	}

	concurrency := getEnvInt("WORKER_CONCURRENCY", 5)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			runWorker(ctx, workerID, queue)
		}(i + 1)
	}

	<-ctx.Done()
	wg.Wait()
}

func runWorker(ctx context.Context, workerID int, queue *services.QueueService) {
	logger := utils.Logger.With(zap.Int("worker_id", workerID))
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		job, err := queue.Dequeue(ctx, 5*time.Second)
		if err != nil {
			if errors.Is(err, redis.Nil) || errors.Is(err, context.Canceled) {
				continue
			}
			logger.Warn("failed to dequeue job", zap.Error(err))
			continue
		}
		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error("worker recovered panic", zap.Any("panic", recovered))
				}
			}()
			if err := handleJob(ctx, job); err != nil {
				logger.Error("job failed", zap.String("job_type", job.Type), zap.Error(err))
			}
		}()
	}
}

func handleJob(ctx context.Context, job *services.QueueJob) error {
	switch job.Type {
	case services.JobPhotoProcess:
		var payload services.PhotoProcessPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return err
		}
		return markPhotoProcessed(payload.PhotoID)
	default:
		utils.Logger.Warn("unknown job type", zap.String("job_type", job.Type))
		return nil
	}
}

func markPhotoProcessed(photoID uint) error {
	var photo models.Photo
	if err := database.DB.First(&photo, photoID).Error; err != nil {
		return err
	}
	now := time.Now()
	return database.DB.Model(&photo).Updates(map[string]interface{}{
		"status":            "completed",
		"processing_status": "completed",
		"processed_at":      &now,
	}).Error
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
