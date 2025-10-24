# build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# copy go mod files
COPY go.mod go.sum ./

# download dependencies
RUN go mod download

# copy source code
COPY . .

# build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o query-interface ./cmd/server

# final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# copy the binary from builder stage
COPY --from=builder /app/query-interface .

# copy config files
COPY --from=builder /app/configs ./configs

# expose port
EXPOSE 4000

# run the application
CMD ["./query-interface"]
