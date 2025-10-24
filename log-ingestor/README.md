# Log Ingestor

high-throughput log ingestor that accepts structured json logs over http and stores them in postgres.

## features

- **high-throughput ingestion**: handles massive volumes of logs efficiently
- **in-memory batching**: configurable batch size and flush intervals
- **object pooling**: reuses logentry objects for better performance
- **sonic json**: fast json unmarshalling using sonic-json
- **bulk inserts**: uses postgres bulk insert for better performance
- **dead letter queue**: invalid logs are sent to dlq for analysis
- **async processing**: background workers handle batch processing
- **graceful shutdown**: handles shutdown signals properly
- **health checks**: monitoring endpoints for service health

## architecture

```
http request → validation → in-memory batcher → async worker → postgres bulk insert
                     ↓
                invalid logs → dlq
```

## configuration

the service uses yaml configuration files:

- `configs/local.yml` - development settings
- `configs/prod.yml` - production settings

### key configuration options

- **server**: host and port settings
- **postgres**: database connection and pool settings
- **batcher**: batch size, flush interval, memory limits
- **worker**: concurrency, retry settings
- **json**: sonic json enablement

## api endpoints

- `POST /logs` - ingest single log or bulk logs
- `GET /health` - health check endpoint
- `GET /stats` - batcher statistics

## log format

```json
{
  "level": "error",
  "message": "failed to connect to db",
  "resourceId": "server-1234",
  "timestamp": "2023-09-15t08:00:00z",
  "traceId": "abc-xyz-123",
  "spanId": "span-456",
  "commit": "5e5342f",
  "metadata": {
    "parentResourceId": "server-0987"
  }
}
```

## running locally

```bash
# start dependencies
docker-compose up -d postgres

# run the service
go run cmd/server/main.go
```

## running with docker

```bash
# build and start all services
docker-compose up --build
```

## performance optimizations

- **object pools**: reuses structs to reduce gc pressure
- **bulk inserts**: uses postgres copy from for faster writes
- **in-memory batching**: reduces database round trips
- **connection pooling**: reuses database connections
- **async processing**: non-blocking log processing
- **sonic json**: faster json parsing than standard library
