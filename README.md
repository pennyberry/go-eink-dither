# Go E-Ink Dithering Server

Converts images to indexed color bitmaps optimized for e-ink displays. Downloads images from URLs, resizes them, and
applies optional dithering.

## Features

- Downloads JPEG images from URLs
- Resizes images to fit within specified dimensions while maintaining aspect ratio
- Processes both colored and grayscale images
- Custom color palettes via hex color lists
- Floyd-Steinberg dithering with perceptually-weighted color matching for colored images
- Optional dithering - can be enabled/disabled via query parameter
- Returns indexed color BMP format optimized for e-ink displays
- HTTP caching support with ETags

## How to Run

```bash
go mod tidy
go build -o go-eink-dither
./go-eink-dither --port 8080
```

## Usage

### Process Image

```
GET /process?url=<image_url>&width=<width>&height=<height>&dither=<true|false>&colors=<hex_colors>
```

**Parameters:**

- `url`: Image URL to process (required)
- `width`: Maximum width in pixels (required)
- `height`: Maximum height in pixels (required)
- `dither`: Apply dithering (optional, defaults to `true`)
- `colors`: Custom palette as comma-separated hex colors (optional, defaults to 4-color grayscale)

**Examples:**

Default 4-color grayscale processing:

```
http://localhost:8080/process?url=https://picsum.photos/500/500&width=400&height=300
```

Spectra E6 color palette:

```
http://locahost:8080/process?url=https://blog.shvn.dev/posts/2025-welcome/images/lighthouse_hu_2abee84ca1903cbc.jpg&width=800&height=480&dither=true&colors=000000,ffffff,e6e600,cc0000,0033cc,00cc00
```

Without dithering:

```
http://localhost:8080/process?url=https://picsum.photos/500/500&width=400&height=300&dither=false
```

### Health Check

```
GET /health
```

Returns "OK" if server is running.

## Output

Returns a BMP image using either:

- **Default**: 4-color grayscale palette (Black, Dark Gray, Light Gray, White)
- **Custom**: Your specified hex colors via the `colors` parameter

Images maintain aspect ratio and use indexed colors optimized for e-ink displays.
