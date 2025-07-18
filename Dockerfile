# Use the official Go image as build stage
FROM golang:1.24-alpine AS builder

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

# Define port as build argument with default value for EXPOSE
ARG PORT=8080

# Expose the port the app runs on
EXPOSE ${PORT}

# Command to run the application with required port argument
# To use a different port, override with: docker run <image> ./main --port <your-port>
CMD ["./main", "--port", "8080"] 