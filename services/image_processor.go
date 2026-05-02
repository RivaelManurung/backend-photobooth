package services

import (
	"backendphotobooth/models"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
)

type ImageProcessor struct {
	storageService *StorageService
}

func NewImageProcessor(storage *StorageService) *ImageProcessor {
	return &ImageProcessor{
		storageService: storage,
	}
}

// ProcessPhoto processes a photo with template and filters
func (ip *ImageProcessor) ProcessPhoto(photo *models.Photo, template *models.Template, filter string) error {
	// Load original image
	originalPath := filepath.Join(".", photo.StoragePath)
	img, err := imaging.Open(originalPath)
	if err != nil {
		return fmt.Errorf("failed to open image: %w", err)
	}

	// Apply filter
	img = ip.applyFilter(img, filter)

	// Apply template overlay if exists
	if template.OverlayURL != "" {
		img, err = ip.applyOverlay(img, template.OverlayURL)
		if err != nil {
			return fmt.Errorf("failed to apply overlay: %w", err)
		}
	}

	// Add watermark if needed
	if photo.HasWatermark {
		img = ip.addWatermark(img)
	}

	// Save processed image
	processedPath := filepath.Join(".", "uploads", "processed", filepath.Base(photo.StoragePath))
	if err := os.MkdirAll(filepath.Dir(processedPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := imaging.Save(img, processedPath); err != nil {
		return fmt.Errorf("failed to save processed image: %w", err)
	}

	// Create thumbnail
	thumbnail := imaging.Resize(img, 300, 0, imaging.Lanczos)
	thumbnailPath := filepath.Join(".", "uploads", "thumbnails", filepath.Base(photo.StoragePath))
	if err := os.MkdirAll(filepath.Dir(thumbnailPath), 0755); err != nil {
		return fmt.Errorf("failed to create thumbnail directory: %w", err)
	}

	if err := imaging.Save(thumbnail, thumbnailPath); err != nil {
		return fmt.Errorf("failed to save thumbnail: %w", err)
	}

	return nil
}

// applyFilter applies image filter
func (ip *ImageProcessor) applyFilter(img image.Image, filter string) image.Image {
	switch filter {
	case "bw":
		return imaging.Grayscale(img)
	case "sepia":
		return ip.applySepia(img)
	case "vivid":
		return imaging.AdjustSaturation(img, 30)
	case "vintage":
		img = ip.applySepia(img)
		return imaging.AdjustContrast(img, -10)
	default:
		return img
	}
}

// applySepia applies sepia tone
func (ip *ImageProcessor) applySepia(img image.Image) image.Image {
	bounds := img.Bounds()
	sepia := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			
			// Convert to 8-bit
			r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)
			
			// Apply sepia formula
			tr := uint8(float64(r8)*0.393 + float64(g8)*0.769 + float64(b8)*0.189)
			tg := uint8(float64(r8)*0.349 + float64(g8)*0.686 + float64(b8)*0.168)
			tb := uint8(float64(r8)*0.272 + float64(g8)*0.534 + float64(b8)*0.131)
			
			sepia.Set(x, y, color.RGBA{tr, tg, tb, uint8(a >> 8)})
		}
	}

	return sepia
}

// applyOverlay applies template overlay
func (ip *ImageProcessor) applyOverlay(base image.Image, overlayPath string) (image.Image, error) {
	// Load overlay image
	overlay, err := imaging.Open(overlayPath)
	if err != nil {
		return base, err
	}

	// Resize overlay to match base image
	overlay = imaging.Resize(overlay, base.Bounds().Dx(), base.Bounds().Dy(), imaging.Lanczos)

	// Create new image
	result := image.NewRGBA(base.Bounds())
	draw.Draw(result, result.Bounds(), base, image.Point{}, draw.Src)
	draw.Draw(result, result.Bounds(), overlay, image.Point{}, draw.Over)

	return result, nil
}

// addWatermark adds watermark to image
func (ip *ImageProcessor) addWatermark(img image.Image) image.Image {
	// Create watermark text
	watermarked := imaging.Clone(img)
	
	// TODO: Add text watermark using freetype library
	// For now, just return the image
	
	return watermarked
}

// CreatePhotoStrip creates a photo strip from multiple images
func (ip *ImageProcessor) CreatePhotoStrip(photos []string, template *models.Template) (string, error) {
	if len(photos) == 0 {
		return "", fmt.Errorf("no photos provided")
	}

	// Load all images
	images := make([]image.Image, len(photos))
	for i, photoPath := range photos {
		img, err := imaging.Open(photoPath)
		if err != nil {
			return "", fmt.Errorf("failed to open photo %d: %w", i, err)
		}
		images[i] = img
	}

	// Determine strip dimensions
	stripWidth := 400
	photoHeight := 300
	spacing := 20
	stripHeight := (photoHeight * len(photos)) + (spacing * (len(photos) + 1))

	// Create strip canvas
	strip := image.NewRGBA(image.Rect(0, 0, stripWidth, stripHeight))
	
	// Fill background
	bgColor := color.RGBA{255, 255, 255, 255}
	draw.Draw(strip, strip.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	// Place photos
	currentY := spacing
	for _, img := range images {
		// Resize photo to fit
		resized := imaging.Resize(img, stripWidth-2*spacing, photoHeight, imaging.Lanczos)
		
		// Draw photo
		draw.Draw(strip, image.Rect(spacing, currentY, stripWidth-spacing, currentY+photoHeight),
			resized, image.Point{}, draw.Over)
		
		currentY += photoHeight + spacing
	}

	// Save strip
	outputPath := filepath.Join(".", "uploads", "strips", fmt.Sprintf("strip-%d.png", os.Getpid()))
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return "", err
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return "", err
	}
	defer outFile.Close()

	if err := png.Encode(outFile, strip); err != nil {
		return "", err
	}

	return outputPath, nil
}

// OptimizeImage optimizes image for web
func (ip *ImageProcessor) OptimizeImage(inputPath, outputPath string, quality int) error {
	img, err := imaging.Open(inputPath)
	if err != nil {
		return err
	}

	// Resize if too large
	maxWidth := 1920
	if img.Bounds().Dx() > maxWidth {
		img = imaging.Resize(img, maxWidth, 0, imaging.Lanczos)
	}

	// Save with compression
	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	return jpeg.Encode(outFile, img, &jpeg.Options{Quality: quality})
}
