# Go E-Ink Dithering Server

A simple HTTP server that downloads, resizes, and applies Floyd-Steinberg dithering to JPEG images, optimized for e-ink displays.

## Features

- Downloads JPEG images from URLs
- Resizes images to fit within specified dimensions while maintaining aspect ratio
- Applies Floyd-Steinberg dithering with a 4-color grayscale palette (black, dark gray, light gray, white)
- Optional dithering - can be enabled/disabled via query parameter
- Custom Floyd-Steinberg implementation for accurate color palette preservation
- Optimized for e-ink displays and low-color output devices
- Fast processing with efficient algorithms

## Dependencies

- [github.com/esimov/dithergo](https://github.com/esimov/dithergo) - Dithering algorithms library
- Go standard library

## Usage

### Build and Run

```bash
go mod tidy
go build -o go-eink-dither
./go-eink-dither
```

The server will start on port 8080.

### API Endpoints

#### Process Image
```
GET /process?url=<image_url>&width=<width>&height=<height>&dither=<true|false>
```

**Parameters:**
- `url`: URL of the JPEG image to process (required)
- `width`: Maximum width in pixels (required)
- `height`: Maximum height in pixels (required)
- `dither`: Enable/disable Floyd-Steinberg dithering (optional, defaults to `true`)

**Examples:**

Process image with dithering (default):
```
http://localhost:8080/process?url=https://picsum.photos/500/500&width=400&height=300
```

Process image with dithering explicitly enabled:
```
http://localhost:8080/process?url=https://picsum.photos/500/500&width=400&height=300&dither=true
```

Process image without dithering:
```
http://localhost:8080/process?url=https://picsum.photos/500/500&width=400&height=300&dither=false
```

#### Health Check
```
GET /health
```

Returns "OK" if the server is running.

## How It Works

1. **Download**: The server downloads the image from the provided URL
2. **Resize**: Calculates the appropriate scale factor to fit the image within the specified dimensions while maintaining aspect ratio
3. **Dither** (optional): Applies Floyd-Steinberg dithering with a 4-color grayscale palette:
   - Black: `#000000`
   - Dark Gray: `#555555`
   - Light Gray: `#AAAAAA`
   - White: `#FFFFFF`
4. **Return**: Returns the processed image as a JPEG with 90% quality

## Floyd-Steinberg Dithering Algorithm

The server implements a custom Floyd-Steinberg dithering algorithm that:

- Processes each pixel individually
- Finds the closest color in the 4-color grayscale palette
- Calculates the quantization error between the original and quantized pixel
- Distributes this error to neighboring pixels using the standard Floyd-Steinberg weights:
  - Right pixel: 7/16 of the error
  - Bottom-left pixel: 3/16 of the error
  - Bottom pixel: 5/16 of the error
  - Bottom-right pixel: 1/16 of the error

This creates smooth transitions between the four gray levels, producing output optimized for e-ink displays and other low-color devices.

## Implementation Details

- Custom Floyd-Steinberg dithering implementation for accurate palette preservation
- Uses Go's standard library for image processing (`image`, `image/jpeg`, `net/http`, etc.)
- Implements custom nearest neighbor interpolation algorithm for resizing
- Handles aspect ratio preservation automatically
- Includes proper error handling and validation
- Optimized for e-ink display characteristics

## Error Handling

The server handles various error cases:
- Missing or invalid parameters
- Invalid image URLs
- Network errors during download
- Invalid JPEG images
- Image processing errors
- Dithering algorithm errors

## Use Cases

Perfect for:
- E-ink display applications
- Low-color output devices
- Retro image processing
- Artistic dithering effects
- Bandwidth-constrained applications requiring small, optimized images