# Logito Makefile

.PHONY: up down restart logs health load reset clean test test-integration setup build truncate

# Start all services
up:
	docker-compose up -d

# Stop all services
down:
	docker-compose down

# Restart all services
restart: down up

# View all service logs
logs:
	docker-compose logs -f

# Check service health
health:
	@echo "Checking service health..."
	@echo "Log Ingestor:"
	@curl -s http://localhost:3000/health | jq . || echo "Service not responding"
	@echo "Query Interface:"
	@curl -s http://localhost:4000/health | jq . || echo "Service not responding"
	@echo "Elasticsearch:"
	@curl -s http://localhost:9200/_cluster/health | jq . || echo "Service not responding"

# Build all services
build:
	@echo "Building all services via Docker Compose..."
	docker-compose build
	@echo "Build complete!"

# Run load tests
load:
	cd loaddata && go run main.go

# Reset database and run migrations
reset:
	./scripts/reset.sh

# Clean up containers and volumes
clean:
	docker-compose down -v --remove-orphans
	@docker-compose rm -f

# Truncate all logs from PostgreSQL, Elasticsearch, and MongoDB DLQ
truncate:
	./scripts/truncate.sh

# Run integration tests only
test: test-integration

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	cd tests && go test -v


# Setup development environment
setup:
	@echo "Setting up development environment..."
	@echo "Installing dependencies..."
	go mod download
	cd tests && go mod download
	@echo "Starting services..."
	$(MAKE) up
	@echo "Waiting for services to be ready..."
	sleep 10
	@echo "Running migrations..."
	$(MAKE) reset
	@echo "Setup complete! Services are running at:"
	@echo "  Frontend: http://localhost:3030"
	@echo "  Query API: http://localhost:4000"
	@echo "  Log Ingestor: http://localhost:3000"
	@echo "  Elasticsearch: http://localhost:9200"


# Help
help:
	@echo "Available commands:"
	@echo "  up              - Start all services"
	@echo "  down            - Stop all services"
	@echo "  restart         - Restart all services"
	@echo "  logs            - View all service logs"
	@echo "  health          - Check service health"
	@echo "  build           - Build all services"
	@echo "  load            - Run load tests"
	@echo "  reset           - Reset database and run migrations"
	@echo "  clean           - Clean up containers and volumes"
	@echo "  truncate        - Truncate all logs from PostgreSQL, Elasticsearch, and MongoDB DLQ"
	@echo "  test-integration- Run integration tests"
	@echo "  setup           - Setup development environment"
	@echo "  help            - Show this help"