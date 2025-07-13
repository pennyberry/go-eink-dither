# Go Image Processing Server

A simple HTTP server that downloads and resizes JPEG images while maintaining aspect ratio.

## Features

- Downloads JPEG images from URLs
- Resizes images to fit within specified dimensions
- Maintains aspect ratio (no stretching)
- Uses only Go standard library (no external dependencies)
- Nearest neighbor interpolation for fast processing

## Usage

### Build and Run

```bash
go build -o image-server
./image-server
```

The server will start on port 8080.

### API Endpoints

#### Process Image
```
GET /process?url=<image_url>&width=<width>&height=<height>
```

**Parameters:**
- `url`: URL of the JPEG image to process (required)
- `width`: Maximum width in pixels (required)
- `height`: Maximum height in pixels (required)

**Example:**
```bash
curl "http://localhost:8080/process?url=https://example.com/image.jpg&width=800&height=600" -o resized_image.jpg
```

#### Health Check
```
GET /health
```

Returns "OK" if the server is running.

## How It Works

1. **Download**: The server downloads the image from the provided URL
2. **Resize**: Calculates the appropriate scale factor to fit the image within the specified dimensions while maintaining aspect ratio
3. **Process**: Uses nearest neighbor interpolation to resize the image pixel by pixel
4. **Return**: Returns the processed image as a JPEG with 90% quality

## Implementation Details

- Uses Go's standard library only (`image`, `image/jpeg`, `net/http`, etc.)
- Implements custom nearest neighbor interpolation algorithm
- Handles aspect ratio preservation automatically
- Includes proper error handling and validation

## Error Handling

The server handles various error cases:
- Missing or invalid parameters
- Invalid image URLs
- Network errors during download
- Invalid JPEG images
- Image processing errors 