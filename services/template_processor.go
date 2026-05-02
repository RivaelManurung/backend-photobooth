package services

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"time"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
	"github.com/google/uuid"
	"backendphotobooth/models"
)

type TemplateProcessor struct {
	outputDir string
}

func NewTemplateProcessor(outputDir string) *TemplateProcessor {
	// Create output directory if not exists
	os.MkdirAll(outputDir, 0755)
	
	return &TemplateProcessor{
		outputDir: outputDir,
	}
}

// ApplyTemplate applies user photos to a template
func (tp *TemplateProcessor) ApplyTemplate(template *models.Template, userPhotoPaths []string, metadata map[string]string) (string, error) {
	// 1. Parse template configuration
	var photoZones []models.PhotoZone
	if err := json.Unmarshal([]byte(template.PhotoZones), &photoZones); err != nil {
		return "", fmt.Errorf("failed to parse photo zones: %w", err)
	}
	
	var textElements []models.TextElement
	if template.TextElements != "" {
		if err := json.Unmarshal([]byte(template.TextElements), &textElements); err != nil {
			return "", fmt.Errorf("failed to parse text elements: %w", err)
		}
	}
	
	// 2. Load background image
	background, err := tp.loadImage(template.BackgroundURL)
	if err != nil {
		return "", fmt.Errorf("failed to load background: %w", err)
	}
	
	// 3. Create canvas with template dimensions
	canvas := imaging.New(template.Width, template.Height, color.White)
	
	// 4. Draw background
	canvas = imaging.Paste(canvas, background, image.Pt(0, 0))
	
	// 5. Process and overlay each user photo
	for i, zone := range photoZones {
		if i >= len(userPhotoPaths) {
			break
		}
		
		// Load user photo
		userPhoto, err := tp.loadImage(userPhotoPaths[i])
		if err != nil {
			continue
		}
		
		// Process photo for this zone
		processedPhoto := tp.processPhotoForZone(userPhoto, zone)
		
		// Overlay on canvas
		canvas = imaging.Paste(canvas, processedPhoto, image.Pt(int(zone.X), int(zone.Y)))
	}
	
	// 6. Add text elements
	if len(textElements) > 0 {
		canvasWithText := tp.addTextElements(canvas, textElements, metadata)
		canvas = imaging.Clone(canvasWithText)
	}
	
	// 7. Save final image
	outputPath := filepath.Join(tp.outputDir, fmt.Sprintf("%s.png", uuid.New().String()))
	if err := tp.saveImage(canvas, outputPath); err != nil {
		return "", fmt.Errorf("failed to save image: %w", err)
	}
	
	return outputPath, nil
}

// processPhotoForZone resizes and applies effects to photo for specific zone
func (tp *TemplateProcessor) processPhotoForZone(photo image.Image, zone models.PhotoZone) image.Image {
	// 1. Resize to fit zone
	resized := imaging.Fill(photo, int(zone.Width), int(zone.Height), imaging.Center, imaging.Lanczos)
	
	// 2. Apply rotation if needed
	if zone.Rotation != 0 {
		resized = imaging.Rotate(resized, zone.Rotation, color.Transparent)
	}
	
	// 3. Apply rounded corners
	if zone.Effects.Rounded > 0 {
		rounded := tp.roundCorners(resized, zone.Effects.Rounded)
		resized = imaging.Clone(rounded)
	}
	
	// 4. Apply border
	if zone.Border.Width > 0 {
		bordered := tp.addBorder(resized, zone.Border)
		resized = imaging.Clone(bordered)
	}
	
	// 5. Apply shadow
	if zone.Effects.Shadow {
		shadowed := tp.addShadow(resized, 10, 10, 5)
		resized = imaging.Clone(shadowed)
	}
	
	// 6. Apply opacity
	if zone.Effects.Opacity > 0 && zone.Effects.Opacity < 1 {
		opaque := tp.adjustOpacity(resized, zone.Effects.Opacity)
		resized = imaging.Clone(opaque)
	}
	
	return resized
}

// addTextElements adds text to the canvas
func (tp *TemplateProcessor) addTextElements(canvas image.Image, textElements []models.TextElement, metadata map[string]string) image.Image {
	dc := gg.NewContextForImage(canvas)
	
	for _, elem := range textElements {
		// Replace variables in content
		content := tp.replaceVariables(elem.Content, metadata)
		
		// Load font
		if err := dc.LoadFontFace(tp.getFontPath(elem.Font.Family), float64(elem.Font.Size)); err != nil {
			continue
		}
		
		// Set color
		c := tp.hexToColor(elem.Font.Color)
		dc.SetColor(c)
		
		// Draw text
		switch elem.Align {
		case "center":
			dc.DrawStringAnchored(content, elem.X, elem.Y, 0.5, 0.5)
		case "right":
			dc.DrawStringAnchored(content, elem.X, elem.Y, 1, 0.5)
		default: // left
			dc.DrawStringAnchored(content, elem.X, elem.Y, 0, 0.5)
		}
	}
	
	return dc.Image()
}

// roundCorners applies rounded corners to image
func (tp *TemplateProcessor) roundCorners(img image.Image, radius int) image.Image {
	bounds := img.Bounds()
	mask := image.NewAlpha(bounds)
	
	dc := gg.NewContext(bounds.Dx(), bounds.Dy())
	dc.DrawRoundedRectangle(0, 0, float64(bounds.Dx()), float64(bounds.Dy()), float64(radius))
	dc.Fill()
	
	draw.DrawMask(mask, bounds, dc.Image(), image.Point{}, dc.Image(), image.Point{}, draw.Over)
	
	result := image.NewRGBA(bounds)
	draw.DrawMask(result, bounds, img, image.Point{}, mask, image.Point{}, draw.Over)
	
	return result
}

// addBorder adds border to image
func (tp *TemplateProcessor) addBorder(img image.Image, border models.Border) image.Image {
	bounds := img.Bounds()
	newWidth := bounds.Dx() + border.Width*2
	newHeight := bounds.Dy() + border.Width*2
	
	bordered := imaging.New(newWidth, newHeight, tp.hexToColor(border.Color))
	bordered = imaging.Paste(bordered, img, image.Pt(border.Width, border.Width))
	
	return bordered
}

// addShadow adds drop shadow to image
func (tp *TemplateProcessor) addShadow(img image.Image, offsetX, offsetY, blur int) image.Image {
	bounds := img.Bounds()
	shadow := imaging.New(bounds.Dx()+offsetX+blur*2, bounds.Dy()+offsetY+blur*2, color.Transparent)
	
	// Create shadow
	shadowImg := imaging.New(bounds.Dx(), bounds.Dy(), color.RGBA{0, 0, 0, 100})
	shadowImg = imaging.Blur(shadowImg, float64(blur))
	
	// Paste shadow
	shadow = imaging.Paste(shadow, shadowImg, image.Pt(offsetX+blur, offsetY+blur))
	
	// Paste original image
	shadow = imaging.Paste(shadow, img, image.Pt(blur, blur))
	
	return shadow
}

// adjustOpacity adjusts image opacity
func (tp *TemplateProcessor) adjustOpacity(img image.Image, opacity float64) image.Image {
	bounds := img.Bounds()
	result := image.NewRGBA(bounds)
	
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			result.Set(x, y, color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: uint8(float64(a>>8) * opacity),
			})
		}
	}
	
	return result
}

// loadImage loads image from file path or URL
func (tp *TemplateProcessor) loadImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	img, _, err := image.Decode(file)
	return img, err
}

// saveImage saves image to file
func (tp *TemplateProcessor) saveImage(img image.Image, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Save as PNG for best quality
	return png.Encode(file, img)
}

// replaceVariables replaces template variables with actual values
func (tp *TemplateProcessor) replaceVariables(content string, metadata map[string]string) string {
	// Replace {{date}} with current date
	if content == "{{date}}" {
		return time.Now().Format("2006-01-02")
	}
	
	// Replace {{time}} with current time
	if content == "{{time}}" {
		return time.Now().Format("15:04:05")
	}
	
	// Replace {{datetime}} with current datetime
	if content == "{{datetime}}" {
		return time.Now().Format("2006-01-02 15:04:05")
	}
	
	// Replace custom metadata
	for key, value := range metadata {
		placeholder := fmt.Sprintf("{{%s}}", key)
		if content == placeholder {
			return value
		}
	}
	
	return content
}

// hexToColor converts hex color to color.Color
func (tp *TemplateProcessor) hexToColor(hex string) color.Color {
	var r, g, b uint8
	
	if len(hex) == 7 && hex[0] == '#' {
		fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	}
	
	return color.RGBA{R: r, G: g, B: b, A: 255}
}

// getFontPath returns path to font file
func (tp *TemplateProcessor) getFontPath(family string) string {
	// Map font families to actual font files
	fontMap := map[string]string{
		"Geist":     "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"Arial":     "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"Helvetica": "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"Times":     "/usr/share/fonts/truetype/dejavu/DejaVuSerif.ttf",
	}
	
	if path, ok := fontMap[family]; ok {
		return path
	}
	
	// Default font
	return "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"
}

// GenerateThumbnail generates thumbnail from template background
func (tp *TemplateProcessor) GenerateThumbnail(backgroundPath string, width, height int) (string, error) {
	img, err := tp.loadImage(backgroundPath)
	if err != nil {
		return "", err
	}
	
	thumbnail := imaging.Fit(img, width, height, imaging.Lanczos)
	
	outputPath := filepath.Join(tp.outputDir, fmt.Sprintf("thumb_%s.jpg", uuid.New().String()))
	
	file, err := os.Create(outputPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	
	if err := jpeg.Encode(file, thumbnail, &jpeg.Options{Quality: 85}); err != nil {
		return "", err
	}
	
	return outputPath, nil
}
