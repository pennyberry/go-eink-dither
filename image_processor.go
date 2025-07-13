package main

import (
	"bytes"
	"fmt"
	"image"
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

// ProcessImage downloads an image from URL and resizes it to fit within the specified dimensions
// while maintaining aspect ratio
func (ip *ImageProcessor) ProcessImage(imageURL string, maxWidth, maxHeight uint) (image.Image, error) {
	// Download the image
	img, err := ip.downloadImage(imageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}

	// Resize the image to fit within the specified dimensions
	resizedImg := ip.resizeImage(img, maxWidth, maxHeight)

	return resizedImg, nil
}

// downloadImage downloads an image from the given URL
func (ip *ImageProcessor) downloadImage(imageURL string) (image.Image, error) {
	// Make HTTP GET request
	resp, err := http.Get(imageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

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
