package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"net/http"
)

// ImageProcessor handles image downloading and processing
type ImageProcessor struct{}

// NewImageProcessor creates a new ImageProcessor instance
func NewImageProcessor() *ImageProcessor {
	return &ImageProcessor{}
}

// getDefaultGrayscalePalette returns a 4-color grayscale palette
func (ip *ImageProcessor) getDefaultGrayscalePalette() color.Palette {
	return color.Palette{
		color.RGBA{0, 0, 0, 255},       // Black
		color.RGBA{85, 85, 85, 255},    // Dark gray
		color.RGBA{170, 170, 170, 255}, // Light gray
		color.RGBA{255, 255, 255, 255}, // White
	}
}

// findClosestColorIndex finds the index of the closest color in the palette
func (ip *ImageProcessor) findClosestColorIndex(c color.Color, palette color.Palette) uint8 {
	r1, g1, b1, _ := c.RGBA()

	closestIndex := uint8(0)
	minDistance := float64(^uint(0) >> 1) // Max float64

	for i, paletteColor := range palette {
		r2, g2, b2, _ := paletteColor.RGBA()

		// Calculate Euclidean distance in RGB space
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

// ApplyDithering applies Floyd-Steinberg dithering to an image using the fixed palette
// and returns a paletted image (indexed color bitmap)
func (ip *ImageProcessor) ApplyDithering(img image.Image, palette color.Palette) (*image.Paletted, error) {
	return ip.applyFloydSteinbergDithering(img, palette), nil
}

// applyFloydSteinbergDithering implements Floyd-Steinberg dithering with a fixed palette
// and returns a paletted image
func (ip *ImageProcessor) applyFloydSteinbergDithering(img image.Image, palette color.Palette) *image.Paletted {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Create a paletted image with the fixed palette
	palettedImg := image.NewPaletted(image.Rect(0, 0, width, height), palette)

	// Create a working copy for error distribution
	rgbaImg := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgbaImg.Set(x, y, img.At(x, y))
		}
	}

	// Apply Floyd-Steinberg dithering
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			oldPixel := rgbaImg.RGBAAt(x+bounds.Min.X, y+bounds.Min.Y)

			// Find the closest color index in our fixed palette
			colorIndex := ip.findClosestColorIndex(oldPixel, palette)
			newPixel := palette[colorIndex]

			// Set the pixel in the paletted image using the color index
			palettedImg.SetColorIndex(x, y, colorIndex)

			// Update the working image for error distribution
			rgbaImg.Set(x+bounds.Min.X, y+bounds.Min.Y, newPixel)

			// Calculate error
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

// addError adds error to a pixel during Floyd-Steinberg dithering
func (ip *ImageProcessor) addError(img *image.RGBA, x, y int, errR, errG, errB int, factor float64) {
	pixel := img.RGBAAt(x, y)

	newR := int(pixel.R) + int(float64(errR)*factor)
	newG := int(pixel.G) + int(float64(errG)*factor)
	newB := int(pixel.B) + int(float64(errB)*factor)

	// Clamp values to [0, 255]
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

// convertToGrayscale converts an image to grayscale using the luminance formula
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

// applyHistogramNormalization applies histogram normalization to enhance contrast
func (ip *ImageProcessor) applyHistogramNormalization(img *image.Gray) *image.Gray {
	bounds := img.Bounds()

	// Create histogram
	histogram := make([]int, 256)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			gray := img.GrayAt(x, y)
			histogram[gray.Y]++
		}
	}

	// Find min and max non-zero values
	minVal := 255
	maxVal := 0
	for i := 0; i < 256; i++ {
		if histogram[i] > 0 {
			if i < minVal {
				minVal = i
			}
			if i > maxVal {
				maxVal = i
			}
		}
	}

	// If the image is already using the full range, return as-is
	if minVal == 0 && maxVal == 255 {
		return img
	}

	// Create normalized image
	normalizedImg := image.NewGray(bounds)

	// Apply linear stretch normalization
	range_ := float64(maxVal - minVal)
	if range_ == 0 {
		// Handle edge case where all pixels have the same value
		return img
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			gray := img.GrayAt(x, y)

			// Normalize to 0-255 range
			normalizedValue := uint8((float64(gray.Y-uint8(minVal)) / range_) * 255.0)
			normalizedImg.SetGray(x, y, color.Gray{Y: normalizedValue})
		}
	}

	return normalizedImg
}

// ProcessImage downloads an image from URL and resizes it to fit within the specified dimensions
// while maintaining aspect ratio, then converts to grayscale, optionally applies histogram normalization,
// and optionally applies Floyd-Steinberg dithering
func (ip *ImageProcessor) ProcessImage(imageURL string, maxWidth, maxHeight uint, enableDither bool, enableNormalize bool, etag string) (image.Image, error) {
	// Download the image
	img, err := ip.downloadImage(imageURL, etag)
	if err != nil {
		return nil, err
	}

	// Resize the image to fit within the specified dimensions
	resizedImg := ip.resizeImage(img, maxWidth, maxHeight)

	// Convert to grayscale
	grayscaleImg := ip.convertToGrayscale(resizedImg)

	// Apply histogram normalization if enabled
	var processedImg image.Image = grayscaleImg
	if enableNormalize {
		processedImg = ip.applyHistogramNormalization(grayscaleImg)
	}

	// Apply dithering with fixed grayscale palette if enabled
	if enableDither {
		palette := ip.getDefaultGrayscalePalette()
		ditheredImg, err := ip.ApplyDithering(processedImg, palette)
		if err != nil {
			return nil, fmt.Errorf("failed to apply dithering: %w", err)
		}
		return ditheredImg, nil
	}

	return processedImg, nil
}

// ErrNotModified is returned when the image has not been modified (304 status)
type ErrNotModified struct{}

func (e ErrNotModified) Error() string {
	return "image not modified"
}

// downloadImage downloads an image from the given URL with optional ETag support
func (ip *ImageProcessor) downloadImage(imageURL string, etag string) (image.Image, error) {
	// Create HTTP client and request
	client := &http.Client{}
	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Add If-None-Match header if ETag is provided
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	// Make HTTP request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Check for 304 Not Modified
	if resp.StatusCode == http.StatusNotModified {
		return nil, ErrNotModified{}
	}

	// Check if the request was successful
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status: %s", resp.Status)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Decode the JPEG image
	img, err := jpeg.Decode(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to decode JPEG image: %w", err)
	}

	return img, nil
}

// resizeImage resizes an image to fit the specified dimensions exactly while maintaining aspect ratio
// by scaling to fill the target dimensions and cropping any excess
func (ip *ImageProcessor) resizeImage(img image.Image, maxWidth, maxHeight uint) image.Image {
	bounds := img.Bounds()
	originalWidth := uint(bounds.Dx())
	originalHeight := uint(bounds.Dy())

	// Calculate the scaling factor to fill the specified dimensions
	scaleX := float64(maxWidth) / float64(originalWidth)
	scaleY := float64(maxHeight) / float64(originalHeight)

	// Use the larger scale to fill the target dimensions (may require cropping)
	scale := scaleX
	if scaleY > scaleX {
		scale = scaleY
	}

	// Calculate scaled dimensions (may be larger than target)
	scaledWidth := float64(originalWidth) * scale
	scaledHeight := float64(originalHeight) * scale

	// Calculate offset to center the scaled image within the target dimensions
	offsetX := (scaledWidth - float64(maxWidth)) / 2
	offsetY := (scaledHeight - float64(maxHeight)) / 2

	// Create a new image with the exact target dimensions
	resizedImg := image.NewRGBA(image.Rect(0, 0, int(maxWidth), int(maxHeight)))

	// Perform nearest neighbor interpolation with cropping
	for y := 0; y < int(maxHeight); y++ {
		for x := 0; x < int(maxWidth); x++ {
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
