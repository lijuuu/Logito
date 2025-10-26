# Configuration Guide

## Default Specifications

The system is configured with the following default specifications:

| Service | CPU | Memory | Purpose |
|---------|-----|--------|---------|
| PostgreSQL | 2.0 cores | 2GB | Database operations |
| Elasticsearch | 2.0 cores | 2GB | Search and indexing |
| Log Ingestor | 1.0 cores | 512MB | High-throughput ingestion |
| Query Interface | 1.0 cores | 512MB | API and indexing |
| Frontend | 0.5 cores | 512MB | Web interface |

**Total System Requirements**: 6.5 cores, 5.5GB RAM

## Changing Specifications

⚠️ **Important**: Changing resource specifications requires additional tuning in configuration files.

### 1. Update Docker Compose Resources

Edit `docker-compose.yml`:
```yaml
postgres:
  deploy:
    resources:
      limits:
        cpus: "4.0"      # Double CPU
        memory: "4G"     # Double memory

elasticsearch:
  environment:
    ES_JAVA_OPTS: "-Xms2g -Xmx2g"  # Double heap
  deploy:
    resources:
      limits:
        cpus: "4.0"      # Double CPU
        memory: "4G"     # Double memory
```

### 2. Tune PostgreSQL Configuration

Edit `.configs/postgresql.conf`:
```conf
# For 4GB RAM
shared_buffers = 1GB              # 25% of RAM
work_mem = 8MB                    # Increase per connection
maintenance_work_mem = 256MB      # Increase for maintenance
max_connections = 200             # Increase connection limit
```

### 3. Tune Service Configurations

#### Log Ingestor (`.configs/config.yaml` - log-ingestor section)
```yaml
log-ingestor:
  postgres:
    maxOpenConns: 200    # Increase for more connections
    maxIdleConns: 50     # Increase for connection reuse

  batcher:
    maxBatchSize: 4000   # Increase for higher throughput
    maxBatchCount: 1000  # Increase queue size

  worker:
    concurrency: 40      # Increase worker threads
```

#### Query Interface (`.configs/config.yaml` - query-interface section)
```yaml
query-interface:
  postgres:
    maxOpenConns: 50     # Increase for more connections
    maxIdleConns: 20     # Increase for connection reuse

  indexer:
    workerCount: 24      # Increase indexing workers
    batchSize: 4000      # Increase batch size
```

## Performance Tuning Guidelines

### Memory Scaling
- **2GB → 4GB**: Double shared_buffers, work_mem
- **4GB → 8GB**: Quadruple shared_buffers, increase connections
- **8GB+**: Consider connection pooling, read replicas

### CPU Scaling
- **2 cores → 4 cores**: Double worker concurrency
- **4 cores → 8 cores**: Quadruple worker concurrency
- **8+ cores**: Consider horizontal scaling

### Storage Scaling
- **<1M rows**: Default settings work fine
- **1M-10M rows**: Increase shared_buffers to 512MB
- **10M-100M rows**: Increase shared_buffers to 1GB+
- **100M+ rows**: Consider partitioning, read replicas

## Environment Variables

### Log Ingestor
```bash
ENV=local
GOMAXPROCS=2          # Match CPU cores
GOGC=50               # Garbage collection tuning
GOMEMLIMIT=512MiB     # Memory limit
```

### Query Interface
```bash
ENV=local
GOMAXPROCS=2          # Match CPU cores
GOMEMLIMIT=512MiB     # Memory limit
```

## Monitoring Configuration

### Health Checks
All services include health checks:
- **PostgreSQL**: `pg_isready` every 10s
- **Elasticsearch**: Cluster health every 30s
- **Services**: HTTP health endpoints every 30s

### Logging
- **Service Logs**: Available via `make logs`
- **Database Logs**: Available via `docker compose logs postgres`
- **Search Logs**: Available via `docker compose logs elasticsearch`
