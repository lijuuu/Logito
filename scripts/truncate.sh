#!/bin/bash

# Script to clear all logs from PostgreSQL and Elasticsearch
# This will remove all data from both databases

set -e

echo "🗑️  Clearing all logs from PostgreSQL and Elasticsearch..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
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

print_warning "This will permanently delete ALL logs from both PostgreSQL and Elasticsearch!"
echo -n "Are you sure you want to continue? (yes/no): "
read -r confirmation

if [ "$confirmation" != "yes" ]; then
    print_status "Operation cancelled."
    exit 0
fi

print_status "Starting cleanup process..."

# 1. Clear PostgreSQL logs
print_status "Clearing PostgreSQL logs..."
docker-compose exec -T postgres psql -U loguser -d logs -c "
DELETE FROM logs;
"

if [ $? -eq 0 ]; then
    print_status "✅ PostgreSQL logs cleared successfully"
else
    print_error "❌ Failed to clear PostgreSQL logs"
    exit 1
fi

# 2. Clear Elasticsearch logs
print_status "Clearing Elasticsearch logs..."

# Delete the entire logs index
docker-compose exec -T elasticsearch curl -X DELETE "localhost:9200/logs?pretty" || true

# Recreate the index with proper mapping
print_status "Recreating Elasticsearch index with proper mapping..."

# Get the mapping from the migration directory
MAPPING=$(cat query-interface/internal/migration/mapping.json)

docker-compose exec -T elasticsearch curl -X PUT "localhost:9200/logs?pretty" \
    -H "Content-Type: application/json" \
    -d "$MAPPING"

if [ $? -eq 0 ]; then
    print_status "✅ Elasticsearch index recreated successfully"
else
    print_error "❌ Failed to recreate Elasticsearch index"
    exit 1
fi

# 3. Verify cleanup
print_status "Verifying cleanup..."

# Check PostgreSQL
PG_COUNT=$(docker-compose exec -T postgres psql -U loguser -d logs -t -c "SELECT COUNT(*) FROM logs;" | tr -d ' \n')
print_status "PostgreSQL logs count: $PG_COUNT"

# Check Elasticsearch
ES_COUNT=$(docker-compose exec -T elasticsearch curl -s "localhost:9200/logs/_count" | grep -o '"count":[0-9]*' | cut -d':' -f2)
print_status "Elasticsearch logs count: $ES_COUNT"

if [ "$PG_COUNT" = "0" ] && [ "$ES_COUNT" = "0" ]; then
    print_status "🎉 All logs have been successfully cleared from both databases!"
    print_status "The system is now clean and ready for new logs."
else
    print_warning "⚠️  Some logs may still remain. Please check manually."
fi

print_status "Cleanup completed!"
