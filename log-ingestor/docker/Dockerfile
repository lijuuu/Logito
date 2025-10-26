FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy configs first
COPY .configs/ ./.configs/

COPY log-ingestor/go.mod log-ingestor/go.sum ./

RUN go mod download

COPY log-ingestor/ .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags='-w -s -extldflags "-static"' \
  -a -installsuffix cgo \
  -o log-ingestor \
  ./cmd/server

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -g 1001 -S appgroup && \
  adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/log-ingestor .

# Copy config from builder stage
COPY --from=builder /app/.configs/config.yaml ./config.yaml

COPY --from=builder /app/internal/migration ./migration

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 3000 6060

HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:3000/health || exit 1

ENV GIN_MODE=release
ENV GOMAXPROCS=1
ENV GOGC=50
ENV GOMEMLIMIT=800MiB

CMD ["./log-ingestor"]
