# Go E-Ink Dithering Server

A simple HTTP server that downloads, resizes, and applies Floyd-Steinberg dithering to JPEG images, returning indexed color bitmaps optimized for e-ink displays.

## Features

- Downloads JPEG images from URLs
- Resizes images to fit within specified dimensions while maintaining aspect ratio
- Applies Floyd-Steinberg dithering with a fixed 4-color grayscale palette
- Returns indexed color bitmaps (.bmp format) with predefined color palette
- Optional dithering - can be enabled/disabled via query parameter
- Custom Floyd-Steinberg implementation for accurate color palette preservation
- Optimized for e-ink displays and low-color output devices
- Fast processing with efficient algorithms

## Dependencies

- [golang.org/x/image/bmp](https://pkg.go.dev/golang.org/x/image/bmp) - Official Go BMP encoder/decoder
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
GET /process?url=<image_url>&width=<width>&height=<height>&dither=<true|false>&display=<display_type>
```

**Parameters:**
- `url`: URL of the JPEG image to process (required)
- `width`: Maximum width in pixels (required)
- `height`: Maximum height in pixels (required)
- `dither`: Enable/disable Floyd-Steinberg dithering (optional, defaults to `true`)
- `display`: Target display type (optional):
  - `spectra-e6`: Use color calibration optimized for Spectra 6 e-ink displays
  - If not specified: Use original 4-color grayscale processing
- `colors`: Custom hex color palette (optional, comma-separated, e.g., `000000,ffffff,ff0000`)
- `normalize`: Enable/disable histogram normalization (optional, defaults to `true`)

**Response:**
- Returns a BMP image with indexed colors from the fixed 4-color palette
- Content-Type: `image/bmp`

**Examples:**

Process image with default 4-color grayscale:
```
http://localhost:8080/process?url=https://picsum.photos/500/500&width=400&height=300
```

Process image optimized for Spectra 6 e-ink display:
```
http://localhost:8080/process?url=https://picsum.photos/500/500&width=400&height=300&display=spectra-e6
```

Process image with custom colors:
```
http://localhost:8080/process?url=https://picsum.photos/500/500&width=400&height=300&colors=000000,ffffff,ff0000,00ff00
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
3. **Process**: Depending on the `display` parameter:
   
   **Default Mode (4-color grayscale):**
   - Applies Floyd-Steinberg dithering with a fixed 4-color grayscale palette:
     - Black: `#000000`, Dark Gray: `#555555`, Light Gray: `#AAAAAA`, White: `#FFFFFF`
   
   **Spectra 6 Mode (`display=spectra-e6`):**
   - Uses color calibration system for optimal e-ink results
   - Dithers with realistic colors that match actual Spectra 6 appearance
   - Outputs device-compatible colors for hardware: Black, White, Blue, Green, Red, Yellow
   - Produces significantly better color gradients and transitions on real displays

4. **Return**: Returns the processed image as an indexed color bitmap (.bmp)

## Indexed Color Bitmap Output

The server generates indexed color bitmaps with the following characteristics:
- **Format**: BMP (bitmap) with 8-bit indexed color
- **Palette**: Fixed 4-color grayscale palette optimized for e-ink displays
- **Color Depth**: 8 bits per pixel (256 possible colors, but only 4 are used)
- **Compression**: None (raw bitmap data)
- **Compatibility**: Standard BMP format compatible with all major image viewers and applications

## Floyd-Steinberg Dithering Algorithm

The server implements a custom Floyd-Steinberg dithering algorithm that:

- Processes each pixel individually
- Finds the closest color in the fixed 4-color grayscale palette
- Maps pixels to color indices in the palette for indexed color output
- Calculates the quantization error between the original and quantized pixel
- Distributes this error to neighboring pixels using the standard Floyd-Steinberg weights:
  - Right pixel: 7/16 of the error
  - Bottom-left pixel: 3/16 of the error
  - Bottom pixel: 5/16 of the error
  - Bottom-right pixel: 1/16 of the error

This creates smooth transitions between the four gray levels, producing indexed color bitmaps optimized for e-ink displays and other low-color devices.

## Implementation Details

- Custom Floyd-Steinberg dithering implementation for accurate palette preservation
- Uses Go's standard library for image processing (`image`, `image/color`, `net/http`, etc.)
- Utilizes the official `golang.org/x/image/bmp` package for bitmap encoding
- Implements custom nearest neighbor interpolation algorithm for resizing
- Creates indexed color images with predefined color palette
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
- Bitmap encoding errors

## Use Cases

Perfect for:
- E-ink display applications requiring indexed color bitmaps
- Low-color output devices with specific palette requirements
- Embedded systems with limited color support
- Retro image processing applications
- Artistic dithering effects with fixed palettes
- Applications requiring small, optimized bitmap files with known color sets

## Technical Notes

- The output bitmap uses an indexed color format where each pixel stores a color index (0-3) rather than RGB values
- The 4-color palette is embedded in the bitmap file header
- This approach ensures consistent color reproduction across different devices
- The indexed format is more efficient for storage and processing on e-ink displays
- Compatible with standard BMP viewers and image processing libraries