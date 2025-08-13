package main

import (
	"errors"
	"fmt"
	"image"
	"log"
	"net/http"
	"strconv"

	"github.com/spf13/cobra"
	"golang.org/x/image/bmp"
)

var (
	port string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "go-eink-dither",
		Short: "A web service for processing images with dithering for e-ink displays",
		Long: `go-eink-dither is a web service that downloads images from URLs,
resizes them, converts to grayscale, and applies Floyd-Steinberg dithering
optimized for e-ink displays.`,
		Run: runServer,
	}

	rootCmd.Flags().StringVarP(&port, "port", "p", "", "Port to run the server on")
	rootCmd.MarkFlagRequired("port")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runServer(cmd *cobra.Command, args []string) {
	processor := NewImageProcessor()

	http.HandleFunc("/process", func(w http.ResponseWriter, r *http.Request) {
		imageURL := r.URL.Query().Get("url")
		widthStr := r.URL.Query().Get("width")
		heightStr := r.URL.Query().Get("height")
		ditherStr := r.URL.Query().Get("dither")
		normalizeStr := r.URL.Query().Get("normalize")
		colorsStr := r.URL.Query().Get("colors")

		etag := r.Header.Get("If-None-Match")

		if imageURL == "" {
			http.Error(w, "Missing required parameter: url", http.StatusBadRequest)
			return
		}
		if widthStr == "" {
			http.Error(w, "Missing required parameter: width", http.StatusBadRequest)
			return
		}
		if heightStr == "" {
			http.Error(w, "Missing required parameter: height", http.StatusBadRequest)
			return
		}

		width, err := strconv.ParseUint(widthStr, 10, 32)
		if err != nil {
			http.Error(w, "Invalid width parameter: must be a positive integer", http.StatusBadRequest)
			return
		}
		height, err := strconv.ParseUint(heightStr, 10, 32)
		if err != nil {
			http.Error(w, "Invalid height parameter: must be a positive integer", http.StatusBadRequest)
			return
		}

		enableDither := true
		if ditherStr != "" {
			enableDither, err = strconv.ParseBool(ditherStr)
			if err != nil {
				http.Error(w, "Invalid dither parameter: must be true or false", http.StatusBadRequest)
				return
			}
		}

		enableNormalize := true
		if normalizeStr != "" {
			enableNormalize, err = strconv.ParseBool(normalizeStr)
			if err != nil {
				http.Error(w, "Invalid normalize parameter: must be true or false", http.StatusBadRequest)
				return
			}
		}

		if width == 0 || height == 0 {
			http.Error(w, "Width and height must be greater than 0", http.StatusBadRequest)
			return
		}

		var processedImage image.Image
		var responseETag string

		if colorsStr != "" {
			// Legacy mode: use old custom colors method (for backward compatibility)
			processedImage, responseETag, err = processor.ProcessImage(imageURL, uint(width), uint(height), enableDither, enableNormalize, colorsStr, etag)
			if err != nil {
				var notModifiedErr ErrNotModified
				if errors.As(err, &notModifiedErr) {
					w.WriteHeader(http.StatusNotModified)
					return
				}
				log.Printf("Error processing image: %v", err)
				http.Error(w, fmt.Sprintf("Error processing image: %v", err), http.StatusInternalServerError)
				return
			}
		} else {
			// Default mode: use Spectra 6 calibrated processing for optimal results
			processedImage, responseETag, err = processor.ProcessImageWithSpectra6Calibration(
				imageURL, uint(width), uint(height), enableDither, enableNormalize, etag)
			if err != nil {
				var notModifiedErr ErrNotModified
				if errors.As(err, &notModifiedErr) {
					w.WriteHeader(http.StatusNotModified)
					return
				}
				log.Printf("Error processing image with Spectra 6 calibration: %v", err)
				http.Error(w, fmt.Sprintf("Error processing image: %v", err), http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "image/bmp")
		w.Header().Set("Cache-Control", "no-cache")

		// Add calibration info to headers
		if colorsStr == "" {
			w.Header().Set("X-Calibration-Used", "spectra6")
		}

		if responseETag != "" {
			w.Header().Set("ETag", responseETag)
		}

		if err := bmp.Encode(w, processedImage); err != nil {
			log.Printf("Error encoding BMP: %v", err)
			http.Error(w, "Error encoding BMP", http.StatusInternalServerError)
			return
		}
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	fmt.Printf("Server starting on port %s\n", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
