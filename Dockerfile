# Use the official Go image as build stage
FROM golang:1.24-alpine AS builder

LABEL dokku.proxy.port-map=http:80:8080,https:443:8080

# Set the working directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Use a minimal image for the final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Set the working directory
WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/main .

# Expose port for local development (Dokku will handle dynamic port mapping)
EXPOSE 8080

# Command to run the application with required port argument
# Use PORT environment variable from Dokku, fallback to 8080 for local development
CMD sh -c './main --port ${PORT:-8080}' 