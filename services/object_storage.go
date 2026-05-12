package services

import (
	"backendphotobooth/config"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type StoredObject struct {
	Key         string `json:"key"`
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

type ObjectStorage interface {
	Put(ctx context.Context, key string, contentType string, body io.Reader, size int64) (*StoredObject, error)
	GetSignedURL(ctx context.Context, key string, ttl time.Duration) (string, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

func NewObjectStorage(cfg *config.Config) (ObjectStorage, error) {
	switch strings.ToLower(cfg.Storage.Driver) {
	case "", "local":
		return NewLocalObjectStorage(cfg.Storage.LocalPath, cfg.Storage.PublicBaseURL), nil
	case "minio", "r2", "s3":
		return NewS3ObjectStorage(cfg)
	default:
		return nil, fmt.Errorf("unsupported storage driver %q", cfg.Storage.Driver)
	}
}

func NewObjectKey(prefix, ext string) string {
	prefix = strings.Trim(strings.TrimSpace(prefix), "/")
	ext = strings.TrimPrefix(ext, ".")
	if ext == "" {
		ext = "bin"
	}
	return fmt.Sprintf("%s/%s.%s", prefix, uuid.New().String(), ext)
}

type LocalObjectStorage struct {
	root          string
	publicBaseURL string
}

func NewLocalObjectStorage(root, publicBaseURL string) *LocalObjectStorage {
	if root == "" {
		root = "./uploads"
	}
	return &LocalObjectStorage{root: root, publicBaseURL: strings.TrimRight(publicBaseURL, "/")}
}

func (s *LocalObjectStorage) Put(ctx context.Context, key string, contentType string, body io.Reader, size int64) (*StoredObject, error) {
	key = cleanObjectKey(key)
	fullPath := filepath.Join(s.root, key)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return nil, err
	}
	file, err := os.Create(fullPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	written, err := io.Copy(file, body)
	if err != nil {
		return nil, err
	}
	return &StoredObject{Key: key, URL: s.publicURL(key), ContentType: contentType, Size: written}, nil
}

func (s *LocalObjectStorage) GetSignedURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	return s.publicURL(cleanObjectKey(key)), nil
}

func (s *LocalObjectStorage) Delete(ctx context.Context, key string) error {
	err := os.Remove(filepath.Join(s.root, cleanObjectKey(key)))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s *LocalObjectStorage) Exists(ctx context.Context, key string) (bool, error) {
	_, err := os.Stat(filepath.Join(s.root, cleanObjectKey(key)))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (s *LocalObjectStorage) publicURL(key string) string {
	key = strings.TrimPrefix(cleanObjectKey(key), "/")
	if s.publicBaseURL != "" {
		return s.publicBaseURL + "/" + key
	}
	return "/uploads/" + key
}

type S3ObjectStorage struct {
	client        *minio.Client
	bucket        string
	publicBaseURL string
}

func NewS3ObjectStorage(cfg *config.Config) (*S3ObjectStorage, error) {
	endpoint := strings.TrimPrefix(strings.TrimPrefix(cfg.Storage.Endpoint, "https://"), "http://")
	if endpoint == "" {
		return nil, fmt.Errorf("storage endpoint is required for %s", cfg.Storage.Driver)
	}
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.Storage.AccessKey, cfg.Storage.SecretKey, ""),
		Secure: strings.HasPrefix(cfg.Storage.Endpoint, "https://") || cfg.Storage.Driver == "r2",
		Region: cfg.Storage.Region,
	})
	if err != nil {
		return nil, err
	}
	return &S3ObjectStorage{
		client:        client,
		bucket:        cfg.Storage.Bucket,
		publicBaseURL: strings.TrimRight(cfg.Storage.PublicBaseURL, "/"),
	}, nil
}

func (s *S3ObjectStorage) Put(ctx context.Context, key string, contentType string, body io.Reader, size int64) (*StoredObject, error) {
	key = cleanObjectKey(key)
	_, err := s.client.PutObject(ctx, s.bucket, key, body, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return nil, err
	}
	urlValue, err := s.GetSignedURL(ctx, key, time.Hour)
	if err != nil {
		urlValue = s.publicURL(key)
	}
	return &StoredObject{Key: key, URL: urlValue, ContentType: contentType, Size: size}, nil
}

func (s *S3ObjectStorage) GetSignedURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if s.publicBaseURL != "" {
		return s.publicURL(key), nil
	}
	reqParams := make(url.Values)
	u, err := s.client.PresignedGetObject(ctx, s.bucket, cleanObjectKey(key), ttl, reqParams)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (s *S3ObjectStorage) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, cleanObjectKey(key), minio.RemoveObjectOptions{})
}

func (s *S3ObjectStorage) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucket, cleanObjectKey(key), minio.StatObjectOptions{})
	if err == nil {
		return true, nil
	}
	resp := minio.ToErrorResponse(err)
	if resp.Code == "NoSuchKey" || resp.Code == "NoSuchBucket" || resp.StatusCode == 404 {
		return false, nil
	}
	return false, err
}

func (s *S3ObjectStorage) publicURL(key string) string {
	if s.publicBaseURL == "" {
		return ""
	}
	return s.publicBaseURL + "/" + strings.TrimPrefix(cleanObjectKey(key), "/")
}

func cleanObjectKey(key string) string {
	key = filepath.ToSlash(strings.TrimSpace(key))
	key = strings.TrimPrefix(key, "/")
	key = strings.ReplaceAll(key, "..", "")
	return key
}
