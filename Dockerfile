# Build stage
FROM golang:1.26.0-alpine AS builder

# Install CA certificates for HTTPS
RUN apk add --no-cache ca-certificates git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies with direct mode to bypass proxy issues
RUN GOPROXY=direct go mod download

# Copy source code
COPY *.go ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o webhook-server

# Final stage
FROM scratch

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/webhook-server .

# Expose port (default 8080, can be overridden)
EXPOSE 8080

# Run the application
CMD ["./webhook-server"]
