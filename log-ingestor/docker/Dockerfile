# Multi-stage build for optimized log ingestor
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags='-w -s -extldflags "-static"' \
  -a -installsuffix cgo \
  -o log-ingestor \
  ./cmd/server

# Final stage - minimal runtime image
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user for security
RUN addgroup -g 1001 -S appgroup && \
  adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/log-ingestor .

# Copy configuration files
COPY --from=builder /app/configs ./configs

# Copy migration files
COPY --from=builder /app/internal/migration ./migration

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose ports
EXPOSE 3000 6060

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:3000/health || exit 1

# Set environment variables
ENV GIN_MODE=release
ENV GOMAXPROCS=1
ENV GOGC=50
ENV GOMEMLIMIT=800MiB

# Run the application
CMD ["./log-ingestor"]
