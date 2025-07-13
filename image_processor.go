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

// resizeImage resizes an image to fit within the specified dimensions while maintaining aspect ratio
func (ip *ImageProcessor) resizeImage(img image.Image, maxWidth, maxHeight uint) image.Image {
	bounds := img.Bounds()
	originalWidth := uint(bounds.Dx())
	originalHeight := uint(bounds.Dy())

	// Calculate the scaling factor to fit within the specified dimensions
	scaleX := float64(maxWidth) / float64(originalWidth)
	scaleY := float64(maxHeight) / float64(originalHeight)

	// Use the smaller scale to maintain aspect ratio
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	// Calculate new dimensions
	newWidth := uint(float64(originalWidth) * scale)
	newHeight := uint(float64(originalHeight) * scale)

	// Create a new image with the calculated dimensions
	resizedImg := image.NewRGBA(image.Rect(0, 0, int(newWidth), int(newHeight)))

	// Perform nearest neighbor interpolation
	for y := 0; y < int(newHeight); y++ {
		for x := 0; x < int(newWidth); x++ {
			// Map the destination coordinates to source coordinates
			srcX := int(float64(x) / scale)
			srcY := int(float64(y) / scale)

			// Ensure we don't go out of bounds
			if srcX >= int(originalWidth) {
				srcX = int(originalWidth) - 1
			}
			if srcY >= int(originalHeight) {
				srcY = int(originalHeight) - 1
			}

			// Get the pixel from the source image and set it in the destination
			srcColor := img.At(srcX+bounds.Min.X, srcY+bounds.Min.Y)
			resizedImg.Set(x, y, srcColor)
		}
	}

	return resizedImg
}
