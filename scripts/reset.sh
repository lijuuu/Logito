#!/bin/bash

# Force Refresh Script - Complete wipe of PostgreSQL and Elasticsearch
# This script will completely remove all data from both databases and recreate them

set -e

echo "🔄 Force Refresh - Complete Database Wipe"
echo "=========================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    print_error "Docker is not running. Please start Docker first."
    exit 1
fi

# Check if containers are running
print_status "Checking if containers are running..."

if ! docker-compose ps | grep -q "logito-postgres-1.*Up"; then
    print_error "PostgreSQL container is not running. Please start the services first with: docker-compose up -d"
    exit 1
fi

if ! docker-compose ps | grep -q "logito-elasticsearch-1.*Up"; then
    print_error "Elasticsearch container is not running. Please start the services first with: docker-compose up -d"
    exit 1
fi

print_warning "This will PERMANENTLY DELETE ALL DATA from both PostgreSQL and Elasticsearch!"
print_warning "This action cannot be undone!"
echo ""
echo -n "Are you absolutely sure you want to continue? Type 'FORCE' to proceed: "
read -r confirmation

if [ "$confirmation" != "FORCE" ]; then
    print_status "Operation cancelled."
    exit 0
fi

print_step "Starting complete database wipe..."

# 1. Stop all services to ensure clean state
print_step "Stopping all services..."
docker-compose down

# 2. Remove all volumes to completely wipe data
print_step "Removing all volumes (this will delete ALL data)..."
docker-compose down -v

# 3. Remove any orphaned containers and networks
print_step "Cleaning up orphaned containers and networks..."
docker-compose down --remove-orphans

# 4. Start services fresh
print_step "Starting services with fresh volumes..."
docker-compose up -d

# 5. Wait for services to be ready
print_step "Waiting for services to be ready..."
sleep 10

# 6. Wait for PostgreSQL to be ready
print_step "Waiting for PostgreSQL to be ready..."
max_attempts=30
attempt=0
while [ $attempt -lt $max_attempts ]; do
    if docker-compose exec -T postgres pg_isready -U loguser -d logs > /dev/null 2>&1; then
        print_status "PostgreSQL is ready"
        break
    fi
    attempt=$((attempt + 1))
    echo -n "."
    sleep 2
done

if [ $attempt -eq $max_attempts ]; then
    print_error "PostgreSQL failed to start within expected time"
    exit 1
fi

# 7. Wait for Elasticsearch to be ready
print_step "Waiting for Elasticsearch to be ready..."
max_attempts=30
attempt=0
while [ $attempt -lt $max_attempts ]; do
    if docker-compose exec -T elasticsearch curl -s "localhost:9200/_cluster/health" > /dev/null 2>&1; then
        print_status "Elasticsearch is ready"
        break
    fi
    attempt=$((attempt + 1))
    echo -n "."
    sleep 2
done

if [ $attempt -eq $max_attempts ]; then
    print_error "Elasticsearch failed to start within expected time"
    exit 1
fi

# 8. Run database migrations
print_step "Running database migrations..."
if [ -f "log-ingestor/internal/migration/migration.sql" ]; then
    docker-compose exec -T postgres psql -U loguser -d logs -f /dev/stdin < log-ingestor/internal/migration/migration.sql
    if [ $? -eq 0 ]; then
        print_status "Database migrations completed successfully"
    else
        print_error "Database migrations failed"
        exit 1
    fi
else
    print_warning "Migration file not found, skipping database setup"
fi

# 9. Create Elasticsearch index with proper mapping
print_step "Creating Elasticsearch index with proper mapping..."

# Delete the index if it exists
docker-compose exec -T elasticsearch curl -X DELETE "localhost:9200/logs?pretty" > /dev/null 2>&1 || true

# Create the index with proper mapping
if [ -f "query-interface/scripts/init-es-index.json" ]; then
    MAPPING=$(cat query-interface/scripts/init-es-index.json)
    docker-compose exec -T elasticsearch curl -X PUT "localhost:9200/logs?pretty" \
        -H "Content-Type: application/json" \
        -d "$MAPPING" > /dev/null 2>&1
    
    if [ $? -eq 0 ]; then
        print_status "Elasticsearch index created successfully"
    else
        print_error "Failed to create Elasticsearch index"
        exit 1
    fi
else
    print_warning "Elasticsearch mapping file not found, using default mapping"
    # Create index with basic mapping
    docker-compose exec -T elasticsearch curl -X PUT "localhost:9200/logs?pretty" \
        -H "Content-Type: application/json" \
        -d '{
            "mappings": {
                "properties": {
                    "id": {"type": "long"},
                    "level": {"type": "keyword"},
                    "message": {"type": "text"},
                    "resourceId": {"type": "keyword"},
                    "timestamp": {"type": "date"},
                    "traceId": {"type": "keyword"},
                    "spanId": {"type": "keyword"},
                    "commit": {"type": "keyword"},
                    "metadata": {"type": "object"},
                    "indexed": {"type": "boolean"},
                    "processingAt": {"type": "date"}
                }
            },
            "settings": {
                "number_of_shards": 1,
                "number_of_replicas": 0,
                "index": {
                    "refresh_interval": "30s",
                    "max_result_window": 10000000
                }
            }
        }' > /dev/null 2>&1
fi

# 10. Verify the setup
print_step "Verifying the setup..."

# Check PostgreSQL
PG_COUNT=$(docker-compose exec -T postgres psql -U loguser -d logs -t -c "SELECT COUNT(*) FROM logs;" 2>/dev/null | tr -d ' \n' || echo "0")
print_status "PostgreSQL logs count: $PG_COUNT"

# Check Elasticsearch
ES_COUNT=$(docker-compose exec -T elasticsearch curl -s "localhost:9200/logs/_count" 2>/dev/null | grep -o '"count":[0-9]*' | cut -d':' -f2 || echo "0")
print_status "Elasticsearch logs count: $ES_COUNT"

# Check if index exists
ES_INDEX_EXISTS=$(docker-compose exec -T elasticsearch curl -s "localhost:9200/logs" 2>/dev/null | grep -o '"logs"' || echo "")
if [ -n "$ES_INDEX_EXISTS" ]; then
    print_status "Elasticsearch index 'logs' exists"
else
    print_warning "Elasticsearch index 'logs' not found"
fi

# 11. Final status
echo ""
print_status "🎉 Force refresh completed successfully!"
print_status "Both PostgreSQL and Elasticsearch have been completely wiped and recreated"
print_status "The system is now clean and ready for new logs"
echo ""
print_status "You can now:"
print_status "  - Start the log-ingestor service"
print_status "  - Start the query-interface service"
print_status "  - Begin ingesting new logs"
echo ""
print_status "Force refresh completed!"
