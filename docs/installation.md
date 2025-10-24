# Installation Guide

## Prerequisites

- Docker 20.10+ with Docker Compose 2.0+
- Make (optional, for convenience commands)
- 8GB RAM minimum (16GB recommended)
- 4 CPU cores minimum (8 cores recommended)

## Quick Start

### 1. Clone and Setup
```bash
git clone <repository-url>
cd Logito
```

### 2. Start All Services
```bash
# Using Docker Compose directly
docker compose up -d

# Or using Make commands
make up
```

### 3. Verify Installation
```bash
# Check service status
docker compose ps

# Check service health
make health

# View logs
make logs
```

## Make Commands

| Command | Description |
|---------|-------------|
| `make up` | Start all services |
| `make down` | Stop all services |
| `make restart` | Restart all services |
| `make logs` | View all service logs |
| `make health` | Check service health |
| `make clean` | Remove all containers and volumes |
| `make mock` | Run load tests |
| `make reset` | Reset database and run migrations |

## Service Endpoints

| Service | URL | Description |
|---------|-----|-------------|
| Frontend | http://localhost:3030 | Web interface |
| Query API | http://localhost:4000 | REST API |
| Log Ingestor | http://localhost:3000 | Log ingestion |
| PostgreSQL | localhost:5433 | Database |
| Elasticsearch | http://localhost:9200 | Search engine |

## Troubleshooting

### Services Not Starting
```bash
# Check logs
make logs

# Restart services
make restart

# Clean restart
make clean && make up
```

### Port Conflicts
If ports are already in use, modify `docker-compose.yml`:
```yaml
ports:
  - "3031:3030"  # Change frontend port
  - "3001:3000"  # Change ingestor port
```

### Memory Issues
- Ensure Docker has at least 8GB RAM allocated
- Check system memory: `docker stats`
- Reduce concurrent load if needed

## Next Steps

1. **Configure Services**: See [Configuration Guide](configuration.md)
2. **Run Load Tests**: `make mock`
3. **Access Web Interface**: http://localhost:3030
4. **Ingest Logs**: Send POST requests to http://localhost:3000/logs
