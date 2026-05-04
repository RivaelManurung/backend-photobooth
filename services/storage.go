package services

import (
	"backendphotobooth/config"
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	storage_go "github.com/supabase-community/storage-go"
)

type StorageService struct {
	config         *config.Config
	supabaseClient *storage_go.Client
}

func NewStorageService(cfg *config.Config) *StorageService {
	var supabaseClient *storage_go.Client
	if cfg.Storage.Provider == "supabase" {
		// Use Supabase URL directly for storage client
		supabaseClient = storage_go.NewClient(cfg.Storage.SupabaseURL, cfg.Storage.SupabaseKey, nil)
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

	// Validate file type
	if !s.isAllowedFileType(file.Header.Get("Content-Type")) {
		return "", fmt.Errorf("file type not allowed")
	}

	// Generate unique filename
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%s-%s%s", uuid.New().String(), time.Now().Format("20060102"), ext)

	if s.config.Storage.Provider == "supabase" {
		src, err := file.Open()
		if err != nil {
			return "", fmt.Errorf("failed to open uploaded file: %w", err)
		}
		defer src.Close()

		buf := bytes.NewBuffer(nil)
		if _, err := io.Copy(buf, src); err != nil {
			return "", fmt.Errorf("failed to read file: %w", err)
		}

		// Path in Supabase bucket (without leading slash)
		path := fmt.Sprintf("%s/%s", folder, filename)
		contentType := file.Header.Get("Content-Type")
		
		_, err = s.supabaseClient.UploadFile(s.config.Storage.SupabaseBucket, path, bytes.NewReader(buf.Bytes()), storage_go.FileOptions{
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

	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	relativeURL := filepath.Join("/uploads", folder, filename)
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

// GetFileURL returns the full URL for a file
func (s *StorageService) GetFileURL(relativePath string) string {
	if relativePath == "" {
		return ""
	}
	
	if s.config.Storage.Provider == "supabase" {
		relativePath = strings.TrimPrefix(relativePath, "/")
		relativePath = strings.TrimPrefix(relativePath, "uploads/")
		
		// Try to create presigned URL valid for 1 hour (3600 seconds)
		resp, err := s.supabaseClient.CreateSignedUrl(s.config.Storage.SupabaseBucket, relativePath, 3600)
		if err != nil {
			// If presigned URL fails, try public URL
			fmt.Printf("Error creating signed URL for %s: %v, trying public URL\n", relativePath, err)
			
			// Return public URL format: https://PROJECT.supabase.co/storage/v1/object/public/BUCKET/PATH
			publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", 
				s.config.Storage.SupabaseURL, 
				s.config.Storage.SupabaseBucket, 
				relativePath)
			return publicURL
		}
		
		return resp.SignedURL
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
