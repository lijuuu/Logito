# build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy configs first
COPY .configs/ ./.configs/

# copy go mod files
COPY query-interface/go.mod query-interface/go.sum ./

# download dependencies
RUN go mod download

# copy source code
COPY query-interface/ .

# build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o query-interface ./cmd/server

# final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# copy the binary from builder stage
COPY --from=builder /app/query-interface .

# copy shared config file from builder stage
COPY --from=builder /app/.configs/config.yaml ./config.yaml

# expose port
EXPOSE 4000

# match log-ingestor's runtime tuning; GOMAXPROCS/GOMEMLIMIT should track
# the container's own cpu/memory limits (1 cpu / 512M in docker-compose.yml)
ENV GIN_MODE=release
ENV GOMAXPROCS=1
ENV GOGC=50
ENV GOMEMLIMIT=400MiB

# run the application
CMD ["./query-interface"]
