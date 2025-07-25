package main

import (
	"errors"
	"fmt"
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
	// Create image processor instance
	processor := NewImageProcessor()

	// Create HTTP handler for the image processing endpoint
	http.HandleFunc("/process", func(w http.ResponseWriter, r *http.Request) {
		// Parse query parameters
		imageURL := r.URL.Query().Get("url")
		widthStr := r.URL.Query().Get("width")
		heightStr := r.URL.Query().Get("height")
		ditherStr := r.URL.Query().Get("dither")
		normalizeStr := r.URL.Query().Get("normalize")

		// Get ETag from request headers
		etag := r.Header.Get("If-None-Match")

		// Validate required parameters
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

		// Parse width and height
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

		// Parse dither parameter (optional, defaults to true)
		enableDither := true
		if ditherStr != "" {
			enableDither, err = strconv.ParseBool(ditherStr)
			if err != nil {
				http.Error(w, "Invalid dither parameter: must be true or false", http.StatusBadRequest)
				return
			}
		}

		// Parse normalize parameter (optional, defaults to true)
		enableNormalize := true
		if normalizeStr != "" {
			enableNormalize, err = strconv.ParseBool(normalizeStr)
			if err != nil {
				http.Error(w, "Invalid normalize parameter: must be true or false", http.StatusBadRequest)
				return
			}
		}

		// Validate dimensions
		if width == 0 || height == 0 {
			http.Error(w, "Width and height must be greater than 0", http.StatusBadRequest)
			return
		}

		// Process the image
		processedImage, err := processor.ProcessImage(imageURL, uint(width), uint(height), enableDither, enableNormalize, etag)
		if err != nil {
			// Check if it's a "not modified" error
			var notModifiedErr ErrNotModified
			if errors.As(err, &notModifiedErr) {
				w.WriteHeader(http.StatusNotModified)
				return
			}
			log.Printf("Error processing image: %v", err)
			http.Error(w, fmt.Sprintf("Error processing image: %v", err), http.StatusInternalServerError)
			return
		}

		// Set response headers for bitmap
		w.Header().Set("Content-Type", "image/bmp")
		w.Header().Set("Cache-Control", "no-cache")

		// Encode and send the image as BMP using the official package
		if err := bmp.Encode(w, processedImage); err != nil {
			log.Printf("Error encoding BMP: %v", err)
			http.Error(w, "Error encoding BMP", http.StatusInternalServerError)
			return
		}
	})

	// Create a simple health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	fmt.Printf("Server starting on port %s\n", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
