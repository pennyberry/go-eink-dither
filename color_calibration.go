package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
)

// ColorCalibration represents a calibrated color palette system
type ColorCalibration struct {
	ProcessingPalette color.Palette // Colors used for dithering (calibrated/realistic)
	DevicePalette     color.Palette // Colors sent to device (pure/digital)
	Name              string        // Display name (e.g., "spectra6", "acep")
}

// Spectra6Calibration contains the calibrated palette optimized for Spectra 6 e-ink displays
var Spectra6Calibration = ColorCalibration{
	ProcessingPalette: color.Palette{
		color.RGBA{33, 33, 34, 255},    // Calibrated black (#212122) - how black actually looks
		color.RGBA{185, 177, 177, 255}, // Calibrated white (#b9b1b1) - how white actually looks
		color.RGBA{65, 82, 160, 255},   // Calibrated blue (#4152a0) - how blue actually looks
		color.RGBA{25, 61, 30, 255},    // Calibrated green (#193d1e) - how green actually looks
		color.RGBA{97, 14, 14, 255},    // Calibrated red (#610e0e) - how red actually looks
		color.RGBA{200, 175, 75, 255},  // Calibrated yellow (#c8af4b) - how yellow actually looks
	},
	DevicePalette: color.Palette{
		color.RGBA{0, 0, 0, 255},       // Pure black - value sent to device
		color.RGBA{255, 255, 255, 255}, // Pure white - value sent to device
		color.RGBA{0, 0, 255, 255},     // Pure blue - value sent to device
		color.RGBA{0, 255, 0, 255},     // Pure green - value sent to device
		color.RGBA{255, 0, 0, 255},     // Pure red - value sent to device
		color.RGBA{255, 255, 0, 255},   // Pure yellow - value sent to device
	},
	Name: "spectra6",
}

// GetSpectra6Calibration returns the Spectra 6 calibration (the only supported display)
func GetSpectra6Calibration() ColorCalibration {
	return Spectra6Calibration
}

// ParseCustomSpectra6Colors creates a custom Spectra 6 calibration from hex colors
// If colors are provided, they should be exactly 6 colors for Spectra 6 compatibility
func ParseCustomSpectra6Colors(processingColors, deviceColors string) (ColorCalibration, error) {
	if processingColors == "" && deviceColors == "" {
		return GetSpectra6Calibration(), nil
	}

	var cal ColorCalibration
	cal.Name = "custom_spectra6"

	// Parse processing palette
	if processingColors != "" {
		palette, err := parseHexColorsPalette(processingColors)
		if err != nil {
			return cal, fmt.Errorf("failed to parse processing colors: %w", err)
		}
		if len(palette) != 6 {
			return cal, fmt.Errorf("processing colors must contain exactly 6 colors for Spectra 6 (got %d)", len(palette))
		}
		cal.ProcessingPalette = palette
	} else {
		cal.ProcessingPalette = Spectra6Calibration.ProcessingPalette
	}

	// Parse device palette
	if deviceColors != "" {
		palette, err := parseHexColorsPalette(deviceColors)
		if err != nil {
			return cal, fmt.Errorf("failed to parse device colors: %w", err)
		}
		if len(palette) != 6 {
			return cal, fmt.Errorf("device colors must contain exactly 6 colors for Spectra 6 (got %d)", len(palette))
		}
		cal.DevicePalette = palette
	} else {
		cal.DevicePalette = Spectra6Calibration.DevicePalette
	}

	return cal, nil
}

// ApplyDeviceColorMapping replaces processing colors with device colors in a paletted image
func ApplyDeviceColorMapping(img *image.Paletted, calibration ColorCalibration) *image.Paletted {
	// Create new image with device palette
	deviceImg := image.NewPaletted(img.Bounds(), calibration.DevicePalette)

	// Copy pixels, mapping color indices remain the same since palettes are aligned
	copy(deviceImg.Pix, img.Pix)

	return deviceImg
}

// GetSpectra6ColorInfo returns information about the Spectra 6 calibration
func GetSpectra6ColorInfo() map[string]interface{} {
	return map[string]interface{}{
		"display_name": "Spectra 6",
		"color_count":  6,
		"colors": map[string]interface{}{
			"processing": []string{"#212122", "#b9b1b1", "#4152a0", "#193d1e", "#610e0e", "#c8af4b"},
			"device":     []string{"#000000", "#ffffff", "#0000ff", "#00ff00", "#ff0000", "#ffff00"},
		},
		"description": "Optimized for Spectra 6 e-ink displays with realistic color calibration",
	}
}

// CalibrationToJSON serializes a calibration to JSON for API responses
func CalibrationToJSON(cal ColorCalibration) ([]byte, error) {
	type JSONColor struct {
		R uint8 `json:"r"`
		G uint8 `json:"g"`
		B uint8 `json:"b"`
		A uint8 `json:"a"`
	}

	type JSONCalibration struct {
		Name              string      `json:"name"`
		ProcessingPalette []JSONColor `json:"processing_palette"`
		DevicePalette     []JSONColor `json:"device_palette"`
	}

	jsonCal := JSONCalibration{
		Name:              cal.Name,
		ProcessingPalette: make([]JSONColor, len(cal.ProcessingPalette)),
		DevicePalette:     make([]JSONColor, len(cal.DevicePalette)),
	}

	for i, c := range cal.ProcessingPalette {
		r, g, b, a := c.RGBA()
		jsonCal.ProcessingPalette[i] = JSONColor{
			R: uint8(r >> 8),
			G: uint8(g >> 8),
			B: uint8(b >> 8),
			A: uint8(a >> 8),
		}
	}

	for i, c := range cal.DevicePalette {
		r, g, b, a := c.RGBA()
		jsonCal.DevicePalette[i] = JSONColor{
			R: uint8(r >> 8),
			G: uint8(g >> 8),
			B: uint8(b >> 8),
			A: uint8(a >> 8),
		}
	}

	return json.Marshal(jsonCal)
}

// Helper function to parse hex colors (reusing existing logic)
func parseHexColorsPalette(colorsStr string) (color.Palette, error) {
	// This would use the existing parseHexColorsPalette logic from image_processor.go
	// We'll move that logic here to avoid duplication
	processor := &ImageProcessor{}
	return processor.parseHexColorsPalette(colorsStr)
}
