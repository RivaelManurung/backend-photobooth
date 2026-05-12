package services

import (
	"backendphotobooth/config"
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	storage_go "github.com/supabase-community/storage-go"
	_ "golang.org/x/image/webp"
)

type StorageService struct {
	config         *config.Config
	supabaseClient *storage_go.Client
}

func NewStorageService(cfg *config.Config) *StorageService {
	var supabaseClient *storage_go.Client
	if cfg.Storage.Provider == "supabase" {
		// Ensure URL has /storage/v1 suffix
		storageURL := cfg.Storage.SupabaseURL
		if !strings.HasSuffix(storageURL, "/storage/v1") {
			storageURL = strings.TrimSuffix(storageURL, "/") + "/storage/v1"
		}
		supabaseClient = storage_go.NewClient(storageURL, cfg.Storage.SupabaseKey, nil)
	}
	return &StorageService{
		config:         cfg,
		supabaseClient: supabaseClient,
	}
}

// UploadFile uploads a file to storage
func (s *StorageService) UploadFile(file *multipart.FileHeader, folder string) (string, error) {
	// Validate file size
	if file.Size > s.config.Storage.MaxUploadSize {
		return "", fmt.Errorf("file size exceeds maximum allowed size")
	}

	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, src); err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	data := buf.Bytes()
	contentType, ext, err := s.validateImageBytes(data)
	if err != nil {
		return "", err
	}

	// Generate unique filename. Do not trust the client-provided filename.
	filename := fmt.Sprintf("%s-%s%s", uuid.New().String(), time.Now().Format("20060102"), ext)

	if s.config.Storage.Provider == "supabase" || s.config.Storage.Driver == "supabase" {
		path := fmt.Sprintf("%s/%s", folder, filename)
		_, err = s.supabaseClient.UploadFile(s.config.Storage.SupabaseBucket, path, bytes.NewReader(data), storage_go.FileOptions{
			ContentType: &contentType,
		})
		if err != nil {
			fmt.Printf("❌ Supabase Upload Error (Bucket: %s, Path: %s): %v\n", s.config.Storage.SupabaseBucket, path, err)
			return "", fmt.Errorf("failed to upload to supabase: %w", err)
		}

		// Return path for database storage (will be converted to presigned URL when needed)
		return path, nil
	}

	// Local Storage Logic
	fullPath := filepath.Join(s.config.Storage.LocalPath, folder, filename)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	relativeURL := filepath.Join("/uploads", folder, filename)
	return strings.ReplaceAll(relativeURL, "\\", "/"), nil
}

// UploadFromBytes uploads raw bytes (e.g. from base64-decoded PNG) to storage
func (s *StorageService) UploadFromBytes(data []byte, folder, contentType string) (string, error) {
	ext := ".jpg"
	if contentType == "image/png" {
		ext = ".png"
	}
	filename := fmt.Sprintf("%s-%s%s", uuid.New().String(), time.Now().Format("20060102"), ext)
	path := fmt.Sprintf("%s/%s", folder, filename)

	if s.config.Storage.Provider == "supabase" {
		_, err := s.supabaseClient.UploadFile(s.config.Storage.SupabaseBucket, path, bytes.NewReader(data), storage_go.FileOptions{
			ContentType: &contentType,
		})
		if err != nil {
			fmt.Printf("❌ Supabase Upload Error (Bucket: %s, Path: %s): %v\n", s.config.Storage.SupabaseBucket, path, err)
			return "", fmt.Errorf("failed to upload to supabase: %w", err)
		}
		return path, nil
	}

	// Local fallback
	fullPath := filepath.Join(s.config.Storage.LocalPath, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", err
	}
	relativeURL := filepath.Join("/uploads", path)
	return strings.ReplaceAll(relativeURL, "\\", "/"), nil
}

// DeleteFile deletes a file from storage
func (s *StorageService) DeleteFile(filePath string) error {
	if filePath == "" {
		return nil
	}

	filePath = strings.TrimPrefix(filePath, "/")

	if s.config.Storage.Provider == "supabase" {
		// Remove uploads/ prefix if present
		filePath = strings.TrimPrefix(filePath, "uploads/")

		_, err := s.supabaseClient.RemoveFile(s.config.Storage.SupabaseBucket, []string{filePath})
		if err != nil {
			return fmt.Errorf("failed to delete from supabase: %w", err)
		}
		return nil
	}

	// Local Storage Logic
	fullPath := filepath.Join(s.config.Storage.LocalPath, filePath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(fullPath)
}

// isAllowedFileType checks if file type is allowed
func (s *StorageService) isAllowedFileType(mimeType string) bool {
	for _, allowed := range s.config.Storage.AllowedFormats {
		if allowed == mimeType {
			return true
		}
	}
	return false
}

func (s *StorageService) validateImageBytes(data []byte) (string, string, error) {
	if len(data) == 0 {
		return "", "", fmt.Errorf("empty file")
	}
	if int64(len(data)) > s.config.Storage.MaxUploadSize {
		return "", "", fmt.Errorf("file size exceeds maximum allowed size")
	}

	contentType := http.DetectContentType(data)
	if contentType == "image/jpg" {
		contentType = "image/jpeg"
	}
	if !s.isAllowedFileType(contentType) {
		return "", "", fmt.Errorf("file type not allowed")
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", "", fmt.Errorf("invalid image file")
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return "", "", fmt.Errorf("invalid image dimensions")
	}
	if cfg.Width > s.config.Storage.MaxImageWidth || cfg.Height > s.config.Storage.MaxImageHeight {
		return "", "", fmt.Errorf("image dimensions exceed maximum allowed size")
	}

	switch contentType {
	case "image/jpeg":
		return contentType, ".jpg", nil
	case "image/png":
		return contentType, ".png", nil
	case "image/webp":
		return contentType, ".webp", nil
	default:
		return "", "", fmt.Errorf("file type not allowed")
	}
}

// GetPublicURL returns a permanent public URL (no expiry) for Supabase files.
// Requires the bucket to be set to PUBLIC in Supabase dashboard.
func (s *StorageService) GetPublicURL(relativePath string) string {
	if relativePath == "" {
		return ""
	}
	if s.config.Storage.Provider == "supabase" {
		relativePath = strings.TrimPrefix(relativePath, "/")
		relativePath = strings.TrimPrefix(relativePath, "uploads/")
		// Format: https://PROJECT.supabase.co/storage/v1/object/public/BUCKET/PATH
		return fmt.Sprintf("%s/storage/v1/object/public/%s/%s",
			s.config.Storage.SupabaseURL,
			s.config.Storage.SupabaseBucket,
			relativePath)
	}
	return "/uploads/" + relativePath
}

// GetFileURL returns the URL for a file.
// For Supabase: returns public URL (permanent, no expiry).
// Falls back to signed URL if bucket is not public.
func (s *StorageService) GetFileURL(relativePath string) string {
	if relativePath == "" {
		return ""
	}

	if s.config.Storage.Provider == "supabase" {
		// Use public URL — no 1-hour expiry issue
		return s.GetPublicURL(relativePath)
	}

	// For local storage, return URL served by backend
	// Ensure path starts with /uploads
	if !strings.HasPrefix(relativePath, "/uploads") {
		if strings.HasPrefix(relativePath, "uploads") {
			relativePath = "/" + relativePath
		} else {
			relativePath = "/uploads/" + relativePath
		}
	}

	// Return full URL with backend host
	// In production, use actual domain. For now, use relative path
	return relativePath
}

// CreateThumbnail creates a thumbnail for an image
func (s *StorageService) CreateThumbnail(originalPath string, width, height int) (string, error) {
	// TODO: Implement image resizing using imaging library
	return originalPath, nil
}
