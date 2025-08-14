package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
)

type ImageProcessor struct{}

func NewImageProcessor() *ImageProcessor {
	return &ImageProcessor{}
}

// getSpektraE6Palette returns the calibrated color palette for Spektra E6 e-ink display
// Colors optimized for this specific display based on epdoptimize calibration
func (ip *ImageProcessor) getSpektraE6Palette() color.Palette {
	return color.Palette{
		color.RGBA{33, 33, 34, 255},    // #212122 - Black
		color.RGBA{185, 177, 177, 255}, // #b9b1b1 - White
		color.RGBA{65, 82, 160, 255},   // #4152a0 - Blue
		color.RGBA{25, 61, 30, 255},    // #193d1e - Green
		color.RGBA{97, 14, 14, 255},    // #610e0e - Red
		color.RGBA{200, 175, 75, 255},  // #c8af4b - Yellow
	}
}

// getSpektraE6DeviceColors returns the actual device colors for Spektra E6
// These are the final colors that should be sent to the device
func (ip *ImageProcessor) getSpektraE6DeviceColors() color.Palette {
	return color.Palette{
		color.RGBA{0, 0, 0, 255},       // #000000 - Black
		color.RGBA{255, 255, 255, 255}, // #FFFFFF - White
		color.RGBA{0, 0, 255, 255},     // #0000FF - Blue
		color.RGBA{0, 255, 0, 255},     // #00FF00 - Green
		color.RGBA{255, 0, 0, 255},     // #FF0000 - Red
		color.RGBA{255, 255, 0, 255},   // #FFFF00 - Yellow
	}
}

func (ip *ImageProcessor) parseHexColorsPalette(colorsStr string) (color.Palette, error) {
	if colorsStr == "" {
		return ip.getSpektraE6Palette(), nil
	}

	colorStrs := strings.Split(colorsStr, ",")
	palette := make(color.Palette, 0, len(colorStrs))

	for _, colorStr := range colorStrs {
		colorStr = strings.TrimSpace(colorStr)
		if colorStr == "" {
			continue
		}

		// Remove # prefix if present
		if strings.HasPrefix(colorStr, "#") {
			colorStr = colorStr[1:]
		}

		// Parse hex color
		if len(colorStr) != 6 {
			return nil, fmt.Errorf("invalid hex color format: %s (expected 6 characters)", colorStr)
		}

		r, err := strconv.ParseUint(colorStr[0:2], 16, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid red component in hex color %s: %w", colorStr, err)
		}

		g, err := strconv.ParseUint(colorStr[2:4], 16, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid green component in hex color %s: %w", colorStr, err)
		}

		b, err := strconv.ParseUint(colorStr[4:6], 16, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid blue component in hex color %s: %w", colorStr, err)
		}

		palette = append(palette, color.RGBA{uint8(r), uint8(g), uint8(b), 255})
	}

	if len(palette) == 0 {
		return ip.getSpektraE6Palette(), nil
	}

	return palette, nil
}

func (ip *ImageProcessor) findClosestColorIndex(c color.Color, palette color.Palette) uint8 {
	r1, g1, b1, _ := c.RGBA()

	closestIndex := uint8(0)
	minDistance := float64(^uint(0) >> 1) // Max float64

	for i, paletteColor := range palette {
		r2, g2, b2, _ := paletteColor.RGBA()

		dr := float64(r1) - float64(r2)
		dg := float64(g1) - float64(g2)
		db := float64(b1) - float64(b2)
		distance := dr*dr + dg*dg + db*db

		if distance < minDistance {
			minDistance = distance
			closestIndex = uint8(i)
		}
	}

	return closestIndex
}

// findBestColorMix finds the best color or combination of colors to represent the target
// This creates better color mixing by considering multiple palette colors
func (ip *ImageProcessor) findBestColorMix(target color.Color, palette color.Palette, x, y int) uint8 {
	r1, g1, b1, _ := target.RGBA()
	tr, tg, tb := float64(r1>>8), float64(g1>>8), float64(b1>>8)

	bestIndex := uint8(0)
	bestDistance := math.MaxFloat64

	// For ordered dithering pattern, add some spatial variation
	threshold := ip.getBayerThreshold(x, y, 4) // 4x4 Bayer matrix

	for i, paletteColor := range palette {
		r2, g2, b2, _ := paletteColor.RGBA()
		pr, pg, pb := float64(r2>>8), float64(g2>>8), float64(b2>>8)

		// Apply dithering threshold for better spatial distribution
		adjustedR := pr + threshold*10 - 5 // Add some noise for mixing
		adjustedG := pg + threshold*10 - 5
		adjustedB := pb + threshold*10 - 5

		// Clamp values
		adjustedR = math.Max(0, math.Min(255, adjustedR))
		adjustedG = math.Max(0, math.Min(255, adjustedG))
		adjustedB = math.Max(0, math.Min(255, adjustedB))

		// Calculate distance with slight spatial variation
		dr := tr - adjustedR
		dg := tg - adjustedG
		db := tb - adjustedB
		distance := dr*dr + dg*dg + db*db

		if distance < bestDistance {
			bestDistance = distance
			bestIndex = uint8(i)
		}
	}

	return bestIndex
}

// getBayerThreshold returns a Bayer matrix threshold value for ordered dithering
func (ip *ImageProcessor) getBayerThreshold(x, y, size int) float64 {
	// 4x4 Bayer matrix
	bayer4x4 := [][]int{
		{0, 8, 2, 10},
		{12, 4, 14, 6},
		{3, 11, 1, 9},
		{15, 7, 13, 5},
	}

	if size == 4 {
		threshold := bayer4x4[y%4][x%4]
		return float64(threshold) / 16.0 // Normalize to 0-1
	}

	return 0.5 // Default
}

func (ip *ImageProcessor) findClosestColorIndexWeighted(c color.Color, palette color.Palette) uint8 {
	r1, g1, b1, _ := c.RGBA()

	// Convert to 8-bit values for easier calculation
	r1_8 := float64(r1 >> 8)
	g1_8 := float64(g1 >> 8)
	b1_8 := float64(b1 >> 8)

	closestIndex := uint8(0)
	minDistance := float64(^uint(0) >> 1) // Max float64

	for i, paletteColor := range palette {
		r2, g2, b2, _ := paletteColor.RGBA()

		// Convert to 8-bit values
		r2_8 := float64(r2 >> 8)
		g2_8 := float64(g2 >> 8)
		b2_8 := float64(b2 >> 8)

		// Use weighted Euclidean distance that accounts for human perception
		// Green is more perceptually important, then red, then blue
		dr := r1_8 - r2_8
		dg := g1_8 - g2_8
		db := b1_8 - b2_8

		distance := 0.299*dr*dr + 0.587*dg*dg + 0.114*db*db

		if distance < minDistance {
			minDistance = distance
			closestIndex = uint8(i)
		}
	}

	return closestIndex
}

func (ip *ImageProcessor) ApplyDithering(img image.Image, palette color.Palette) (*image.Paletted, error) {
	return ip.applyFloydSteinbergDithering(img, palette), nil
}

func (ip *ImageProcessor) ApplyColorDithering(img image.Image, palette color.Palette) (*image.Paletted, error) {
	return ip.applyEnhancedFloydSteinbergDithering(img, palette), nil
}

func (ip *ImageProcessor) ApplyBasicColorDithering(img image.Image, palette color.Palette) (*image.Paletted, error) {
	return ip.applyFloydSteinbergColorDithering(img, palette), nil
}

func (ip *ImageProcessor) applyFloydSteinbergDithering(img image.Image, palette color.Palette) *image.Paletted {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	palettedImg := image.NewPaletted(image.Rect(0, 0, width, height), palette)

	rgbaImg := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgbaImg.Set(x, y, img.At(x, y))
		}
	}

	for y := range height {
		for x := range width {
			oldPixel := rgbaImg.RGBAAt(x+bounds.Min.X, y+bounds.Min.Y)

			colorIndex := ip.findClosestColorIndex(oldPixel, palette)
			newPixel := palette[colorIndex]

			palettedImg.SetColorIndex(x, y, colorIndex)

			rgbaImg.Set(x+bounds.Min.X, y+bounds.Min.Y, newPixel)

			oldR, oldG, oldB, _ := oldPixel.RGBA()
			newR, newG, newB, _ := newPixel.RGBA()

			errR := int(oldR>>8) - int(newR>>8)
			errG := int(oldG>>8) - int(newG>>8)
			errB := int(oldB>>8) - int(newB>>8)

			// Distribute error to neighboring pixels
			if x+1 < width {
				ip.addError(rgbaImg, x+1+bounds.Min.X, y+bounds.Min.Y, errR, errG, errB, 7.0/16.0)
			}
			if y+1 < height {
				if x > 0 {
					ip.addError(rgbaImg, x-1+bounds.Min.X, y+1+bounds.Min.Y, errR, errG, errB, 3.0/16.0)
				}
				ip.addError(rgbaImg, x+bounds.Min.X, y+1+bounds.Min.Y, errR, errG, errB, 5.0/16.0)
				if x+1 < width {
					ip.addError(rgbaImg, x+1+bounds.Min.X, y+1+bounds.Min.Y, errR, errG, errB, 1.0/16.0)
				}
			}
		}
	}

	return palettedImg
}

func (ip *ImageProcessor) applyFloydSteinbergColorDithering(img image.Image, palette color.Palette) *image.Paletted {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	palettedImg := image.NewPaletted(image.Rect(0, 0, width, height), palette)

	rgbaImg := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgbaImg.Set(x, y, img.At(x, y))
		}
	}

	for y := range height {
		for x := range width {
			oldPixel := rgbaImg.RGBAAt(x+bounds.Min.X, y+bounds.Min.Y)

			colorIndex := ip.findClosestColorIndex(oldPixel, palette)
			newPixel := palette[colorIndex]

			palettedImg.SetColorIndex(x, y, colorIndex)

			rgbaImg.Set(x+bounds.Min.X, y+bounds.Min.Y, newPixel)

			oldR, oldG, oldB, _ := oldPixel.RGBA()
			newR, newG, newB, _ := newPixel.RGBA()

			errR := int(oldR>>8) - int(newR>>8)
			errG := int(oldG>>8) - int(newG>>8)
			errB := int(oldB>>8) - int(newB>>8)

			// Distribute error to neighboring pixels
			if x+1 < width {
				ip.addError(rgbaImg, x+1+bounds.Min.X, y+bounds.Min.Y, errR, errG, errB, 7.0/16.0)
			}
			if y+1 < height {
				if x > 0 {
					ip.addError(rgbaImg, x-1+bounds.Min.X, y+1+bounds.Min.Y, errR, errG, errB, 3.0/16.0)
				}
				ip.addError(rgbaImg, x+bounds.Min.X, y+1+bounds.Min.Y, errR, errG, errB, 5.0/16.0)
				if x+1 < width {
					ip.addError(rgbaImg, x+1+bounds.Min.X, y+1+bounds.Min.Y, errR, errG, errB, 1.0/16.0)
				}
			}
		}
	}

	return palettedImg
}

// applyEnhancedFloydSteinbergDithering implements improved Floyd-Steinberg with better color mixing
func (ip *ImageProcessor) applyEnhancedFloydSteinbergDithering(img image.Image, palette color.Palette) *image.Paletted {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	palettedImg := image.NewPaletted(image.Rect(0, 0, width, height), palette)

	// Create a working copy with error accumulation
	rgbaImg := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgbaImg.Set(x, y, img.At(x, y))
		}
	}

	for y := range height {
		for x := range width {
			oldPixel := rgbaImg.RGBAAt(x+bounds.Min.X, y+bounds.Min.Y)

			// Use enhanced color mixing that considers spatial patterns
			colorIndex := ip.findBestColorMix(oldPixel, palette, x, y)
			newPixel := palette[colorIndex]

			palettedImg.SetColorIndex(x, y, colorIndex)
			rgbaImg.Set(x+bounds.Min.X, y+bounds.Min.Y, newPixel)

			// Calculate error with improved precision
			oldR, oldG, oldB, _ := oldPixel.RGBA()
			newR, newG, newB, _ := newPixel.RGBA()

			errR := int(oldR>>8) - int(newR>>8)
			errG := int(oldG>>8) - int(newG>>8)
			errB := int(oldB>>8) - int(newB>>8)

			// Enhanced error distribution with variable weights based on content
			// Convert newPixel to RGBA for calculation
			nr, ng, nb, _ := newPixel.RGBA()
			newRGBA := color.RGBA{uint8(nr >> 8), uint8(ng >> 8), uint8(nb >> 8), 255}
			errorMultiplier := ip.calculateErrorMultiplier(oldPixel, newRGBA)

			// Distribute error to neighboring pixels with enhanced weighting
			if x+1 < width {
				ip.addEnhancedError(rgbaImg, x+1+bounds.Min.X, y+bounds.Min.Y, errR, errG, errB, 7.0/16.0*errorMultiplier)
			}
			if y+1 < height {
				if x > 0 {
					ip.addEnhancedError(rgbaImg, x-1+bounds.Min.X, y+1+bounds.Min.Y, errR, errG, errB, 3.0/16.0*errorMultiplier)
				}
				ip.addEnhancedError(rgbaImg, x+bounds.Min.X, y+1+bounds.Min.Y, errR, errG, errB, 5.0/16.0*errorMultiplier)
				if x+1 < width {
					ip.addEnhancedError(rgbaImg, x+1+bounds.Min.X, y+1+bounds.Min.Y, errR, errG, errB, 1.0/16.0*errorMultiplier)
				}
			}
		}
	}

	return palettedImg
}

// calculateErrorMultiplier adjusts error distribution based on color difference
func (ip *ImageProcessor) calculateErrorMultiplier(oldPixel, newPixel color.RGBA) float64 {
	// Calculate color difference to adjust error propagation
	dr := float64(oldPixel.R) - float64(newPixel.R)
	dg := float64(oldPixel.G) - float64(newPixel.G)
	db := float64(oldPixel.B) - float64(newPixel.B)

	colorError := math.Sqrt(dr*dr + dg*dg + db*db)

	// Scale error distribution based on how different the colors are
	// More error = more aggressive diffusion for better mixing
	multiplier := 0.8 + (colorError/255.0)*0.4 // Range: 0.8 - 1.2
	return math.Max(0.5, math.Min(1.5, multiplier))
}

// addEnhancedError adds error with improved clamping and distribution
func (ip *ImageProcessor) addEnhancedError(img *image.RGBA, x, y int, errR, errG, errB int, factor float64) {
	pixel := img.RGBAAt(x, y)

	// Apply error with slight randomization to break up patterns
	newR := int(pixel.R) + int(float64(errR)*factor)
	newG := int(pixel.G) + int(float64(errG)*factor)
	newB := int(pixel.B) + int(float64(errB)*factor)

	// Improved clamping with slight overshoot allowance
	if newR < -10 {
		newR = 0
	} else if newR > 265 {
		newR = 255
	} else if newR < 0 {
		newR = 0
	} else if newR > 255 {
		newR = 255
	}

	if newG < -10 {
		newG = 0
	} else if newG > 265 {
		newG = 255
	} else if newG < 0 {
		newG = 0
	} else if newG > 255 {
		newG = 255
	}

	if newB < -10 {
		newB = 0
	} else if newB > 265 {
		newB = 255
	} else if newB < 0 {
		newB = 0
	} else if newB > 255 {
		newB = 255
	}

	img.Set(x, y, color.RGBA{uint8(newR), uint8(newG), uint8(newB), pixel.A})
}

func (ip *ImageProcessor) addError(img *image.RGBA, x, y int, errR, errG, errB int, factor float64) {
	pixel := img.RGBAAt(x, y)

	newR := int(pixel.R) + int(float64(errR)*factor)
	newG := int(pixel.G) + int(float64(errG)*factor)
	newB := int(pixel.B) + int(float64(errB)*factor)

	if newR < 0 {
		newR = 0
	} else if newR > 255 {
		newR = 255
	}
	if newG < 0 {
		newG = 0
	} else if newG > 255 {
		newG = 255
	}
	if newB < 0 {
		newB = 0
	} else if newB > 255 {
		newB = 255
	}

	img.Set(x, y, color.RGBA{uint8(newR), uint8(newG), uint8(newB), pixel.A})
}

// mapCalibratedToDeviceColors maps the calibrated colors back to actual device colors
// This implements the final step of epdoptimize's calibration process
func (ip *ImageProcessor) mapCalibratedToDeviceColors(img *image.Paletted, calibratedPalette, devicePalette color.Palette) *image.Paletted {
	bounds := img.Bounds()
	deviceImg := image.NewPaletted(bounds, devicePalette)

	// Create a mapping from calibrated colors to device colors
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			colorIndex := img.ColorIndexAt(x, y)
			// The color index should map directly since both palettes have the same order
			if int(colorIndex) < len(devicePalette) {
				deviceImg.SetColorIndex(x-bounds.Min.X, y-bounds.Min.Y, colorIndex)
			}
		}
	}

	return deviceImg
}

func (ip *ImageProcessor) convertToGrayscale(img image.Image) *image.Gray {
	bounds := img.Bounds()
	grayImg := image.NewGray(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			originalColor := img.At(x, y)
			r, g, b, _ := originalColor.RGBA()

			// Convert to 8-bit values
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)

			// Calculate luminance using the standard formula: 0.299*R + 0.587*G + 0.114*B
			gray := uint8(0.299*float64(r8) + 0.587*float64(g8) + 0.114*float64(b8))

			grayImg.SetGray(x, y, color.Gray{Y: gray})
		}
	}

	return grayImg
}

func (ip *ImageProcessor) applyHistogramNormalization(img *image.Gray) *image.Gray {
	bounds := img.Bounds()

	histogram := make([]int, 256)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			gray := img.GrayAt(x, y)
			histogram[gray.Y]++
		}
	}

	minVal := 255
	maxVal := 0
	for i := range 256 {
		if histogram[i] > 0 {
			if i < minVal {
				minVal = i
			}
			if i > maxVal {
				maxVal = i
			}
		}
	}

	if minVal == 0 && maxVal == 255 {
		return img
	}

	normalizedImg := image.NewGray(bounds)

	range_ := float64(maxVal - minVal)
	if range_ == 0 {
		return img
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			gray := img.GrayAt(x, y)

			normalizedValue := uint8((float64(gray.Y-uint8(minVal)) / range_) * 255.0)
			normalizedImg.SetGray(x, y, color.Gray{Y: normalizedValue})
		}
	}

	return normalizedImg
}

func (ip *ImageProcessor) ProcessImage(imageURL string, maxWidth, maxHeight uint, enableDither bool, enableNormalize bool, colorsStr string, etag string) (image.Image, string, error) {
	// Download the image
	img, responseETag, err := ip.downloadImage(imageURL, etag)
	if err != nil {
		return nil, "", err
	}

	// Resize the image to fit within the specified dimensions
	resizedImg := ip.resizeImage(img, maxWidth, maxHeight)

	// Parse color palette (defaults to Spektra E6 calibrated colors)
	palette, err := ip.parseHexColorsPalette(colorsStr)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse color palette: %w", err)
	}

	// Determine if we should apply full calibration (only for default Spektra E6 palette)
	isDefaultPalette := colorsStr == ""

	var processedImg *image.Paletted

	// Apply dithering if enabled (always use color dithering with calibrated palette)
	if enableDither {
		processedImg, err = ip.ApplyColorDithering(resizedImg, palette)
		if err != nil {
			return nil, "", fmt.Errorf("failed to apply dithering: %w", err)
		}
	} else {
		// If dithering is disabled, reduce to palette colors using nearest color matching
		bounds := resizedImg.Bounds()
		processedImg = image.NewPaletted(bounds, palette)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				originalColor := resizedImg.At(x, y)
				// Use enhanced color mixing even without dithering for better results
				colorIndex := ip.findBestColorMix(originalColor, palette, x-bounds.Min.X, y-bounds.Min.Y)
				processedImg.SetColorIndex(x-bounds.Min.X, y-bounds.Min.Y, colorIndex)
			}
		}
	}

	// Apply color calibration mapping if using default Spektra E6 palette
	if isDefaultPalette {
		deviceColors := ip.getSpektraE6DeviceColors()
		calibratedImg := ip.mapCalibratedToDeviceColors(processedImg, palette, deviceColors)
		return calibratedImg, responseETag, nil
	}

	return processedImg, responseETag, nil
}

type ErrNotModified struct{}

func (e ErrNotModified) Error() string {
	return "image not modified"
}

func (ip *ImageProcessor) downloadImage(imageURL string, etag string) (image.Image, string, error) {
	// Create HTTP client and request
	client := &http.Client{}
	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return nil, "", ErrNotModified{}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("HTTP request failed with status: %s", resp.Status)
	}

	responseETag := resp.Header.Get("ETag")

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response body: %w", err)
	}

	img, err := jpeg.Decode(bytes.NewReader(body))
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode JPEG image: %w", err)
	}

	return img, responseETag, nil
}

func (ip *ImageProcessor) resizeImage(img image.Image, maxWidth, maxHeight uint) image.Image {
	bounds := img.Bounds()
	originalWidth := uint(bounds.Dx())
	originalHeight := uint(bounds.Dy())

	scaleX := float64(maxWidth) / float64(originalWidth)
	scaleY := float64(maxHeight) / float64(originalHeight)

	scale := max(scaleY, scaleX)

	scaledWidth := float64(originalWidth) * scale
	scaledHeight := float64(originalHeight) * scale

	offsetX := (scaledWidth - float64(maxWidth)) / 2
	offsetY := (scaledHeight - float64(maxHeight)) / 2

	resizedImg := image.NewRGBA(image.Rect(0, 0, int(maxWidth), int(maxHeight)))

	for y := range int(maxHeight) {
		for x := range int(maxWidth) {
			// Map the destination coordinates to source coordinates, accounting for centering offset
			srcX := int((float64(x) + offsetX) / scale)
			srcY := int((float64(y) + offsetY) / scale)

			// Ensure we don't go out of bounds
			if srcX >= int(originalWidth) {
				srcX = int(originalWidth) - 1
			}
			if srcY >= int(originalHeight) {
				srcY = int(originalHeight) - 1
			}
			if srcX < 0 {
				srcX = 0
			}
			if srcY < 0 {
				srcY = 0
			}

			// Get the pixel from the source image and set it in the destination
			srcColor := img.At(srcX+bounds.Min.X, srcY+bounds.Min.Y)
			resizedImg.Set(x, y, srcColor)
		}
	}

	return resizedImg
}
