package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type ImageProcessor struct{}

func NewImageProcessor() *ImageProcessor {
	return &ImageProcessor{}
}

func (ip *ImageProcessor) getDefaultGrayscalePalette() color.Palette {
	return color.Palette{
		color.RGBA{0, 0, 0, 255},
		color.RGBA{85, 85, 85, 255},
		color.RGBA{170, 170, 170, 255},
		color.RGBA{255, 255, 255, 255},
	}
}

func (ip *ImageProcessor) parseHexColorsPalette(colorsStr string) (color.Palette, error) {
	if colorsStr == "" {
		return ip.getDefaultGrayscalePalette(), nil
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
		return ip.getDefaultGrayscalePalette(), nil
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

			colorIndex := ip.findClosestColorIndexWeighted(oldPixel, palette)
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

	// Parse custom color palette
	palette, err := ip.parseHexColorsPalette(colorsStr)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse color palette: %w", err)
	}

	// Check if we're using custom colors (not the default grayscale palette)
	isCustomColors := colorsStr != ""

	// Apply dithering if enabled
	if enableDither {
		var ditheredImg *image.Paletted
		if isCustomColors {
			// For custom colors, work with the original RGB image
			ditheredImg, err = ip.ApplyColorDithering(resizedImg, palette)
		} else {
			// For grayscale, convert to grayscale first and optionally normalize
			grayscaleImg := ip.convertToGrayscale(resizedImg)
			var processedImg image.Image = grayscaleImg
			if enableNormalize {
				processedImg = ip.applyHistogramNormalization(grayscaleImg)
			}
			ditheredImg, err = ip.ApplyDithering(processedImg, palette)
		}
		if err != nil {
			return nil, "", fmt.Errorf("failed to apply dithering: %w", err)
		}
		return ditheredImg, responseETag, nil
	}

	// If dithering is disabled, return processed image
	if isCustomColors {
		// For custom colors without dithering, we still need to reduce to the palette colors
		// but without dithering (simple nearest color matching)
		bounds := resizedImg.Bounds()
		palettedImg := image.NewPaletted(bounds, palette)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				originalColor := resizedImg.At(x, y)
				colorIndex := ip.findClosestColorIndexWeighted(originalColor, palette)
				palettedImg.SetColorIndex(x-bounds.Min.X, y-bounds.Min.Y, colorIndex)
			}
		}
		return palettedImg, responseETag, nil
	} else {
		// For grayscale without dithering
		grayscaleImg := ip.convertToGrayscale(resizedImg)
		if enableNormalize {
			return ip.applyHistogramNormalization(grayscaleImg), responseETag, nil
		}
		return grayscaleImg, responseETag, nil
	}
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
