# Services Overview
## Service Details

### 1. Log Ingestor
**Port**: 3000  
**Purpose**: High-throughput log ingestion and processing

**Features**:
- Batch processing for high throughput
- Connection pooling for database efficiency
- Worker pool for parallel processing
- Health monitoring endpoint

**Configuration**: `log-ingestor/configs/local.yml`

### 2. Query Interface
**Port**: 4000  
**Purpose**: REST API for log queries and search

**Features**:
- RESTful API endpoints
- Elasticsearch integration
- Database querying
- Real-time indexing status

**Configuration**: `query-interface/configs/local.yml`

### 3. Frontend
**Port**: 3030  
**Purpose**: Web interface for log management

**Features**:
- Modern React-based UI
- Real-time log viewing
- Advanced filtering and search
- Performance metrics dashboard

**Configuration**: Environment variables in `docker-compose.yml`

### 4. PostgreSQL
**Port**: 5433  
**Purpose**: Primary database for log storage

**Features**:
- Optimized for high-throughput writes
- Connection pooling
- Health monitoring
- Persistent data storage

**Configuration**: `.configs/postgresql.conf`

### 5. Elasticsearch
**Port**: 9200  
**Purpose**: Search and indexing engine

**Features**:
- Full-text search capabilities
- Real-time indexing
- Cluster health monitoring
- Optimized for log data

**Configuration**: Environment variables in `docker-compose.yml`

## Service Dependencies

```
Frontend → Query Interface → PostgreSQL
                    ↓
              Elasticsearch
                    ↑
            Log Ingestor → PostgreSQL
```

## Health Monitoring

All services provide health check endpoints:

- **Log Ingestor**: `GET /health`
- **Query Interface**: `GET /health`
- **PostgreSQL**: `pg_isready` command
- **Elasticsearch**: `GET /_cluster/health`
