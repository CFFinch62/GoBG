# GoBG - Go Backgammon Engine
# Multi-stage build for minimal final image

# ============================================
# Stage 1: Build the Go binary
# ============================================
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /build

# Copy go mod files first (better layer caching)
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary with optimizations
# CGO_ENABLED=0 produces a static binary
# -ldflags="-s -w" strips debug info for smaller binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /build/bgserver \
    ./cmd/bgserver

# ============================================
# Stage 2: Create minimal runtime image
# ============================================
FROM alpine:3.19

# Add ca-certificates for HTTPS and tzdata for timezones
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user for security
RUN addgroup -g 1000 gobg && \
    adduser -u 1000 -G gobg -s /bin/sh -D gobg

# Set working directory
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/bgserver /app/bgserver

# Copy data files
COPY data/gnubg.weights /app/data/gnubg.weights
COPY data/gnubg_os0.bd /app/data/gnubg_os0.bd
COPY data/gnubg_ts.bd /app/data/gnubg_ts.bd
COPY data/g11.xml /app/data/g11.xml

# Set ownership
RUN chown -R gobg:gobg /app

# Switch to non-root user
USER gobg

# Expose the API port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Default command
# Use 0.0.0.0 to accept connections from outside the container
CMD ["/app/bgserver", "-host", "0.0.0.0", "-port", "8080"]

