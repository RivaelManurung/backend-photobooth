package utils

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	_ "golang.org/x/image/webp"
)

type ValidationConfig struct {
	MaxFileSize    int64
	AllowedMimeTypes []string
}

func ValidateImageUpload(fileHeader *multipart.FileHeader, config ValidationConfig) error {
	// 1. Check file size
	if fileHeader.Size > config.MaxFileSize {
		return fmt.Errorf("file size %d exceeds maximum allowed %d", fileHeader.Size, config.MaxFileSize)
	}

	// 2. Open file for content validation
	file, err := fileHeader.Open()
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// 3. Check MIME type via Sniffing
	buffer := make([]byte, 512)
	if _, err := file.Read(buffer); err != nil {
		return fmt.Errorf("failed to read file header: %w", err)
	}

	mimeType := http.DetectContentType(buffer)
	isAllowed := false
	for _, allowed := range config.AllowedMimeTypes {
		if mimeType == allowed {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		return fmt.Errorf("mime type %s is not allowed", mimeType)
	}

	// 4. Seek back to start for decoding
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to reset file pointer: %w", err)
	}

	// 5. Validate image decoding (ensures file is not corrupted and is a real image)
	_, _, err = image.DecodeConfig(file)
	if err != nil {
		return fmt.Errorf("invalid image format or corrupted file: %w", err)
	}

	// 6. Safe filename
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		return fmt.Errorf("file extension %s is not allowed", ext)
	}

	return nil
}
