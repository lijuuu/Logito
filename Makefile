# Logito Makefile

.PHONY: up down restart logs health mock reset clean test test-unit test-integration

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

# Run load tests
mock:
	cd mockdata && go run main.go

# Reset database and run migrations
reset:
	./scripts/reset.sh

# Clean up containers and volumes
clean:
	docker-compose down -v --remove-orphans
	docker system prune -f

# Run all tests
test: test-unit test-integration

# Run unit tests
test-unit:
	@echo "Running log ingestor unit tests..."
	cd log-ingestor && go test ./internal/ingest/... -v
	@echo "Running query interface unit tests..."
	cd query-interface && go test ./internal/api/... -v

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	cd tests && go test -v

# Build all services
build:
	docker-compose build

# Start services in development mode
dev:
	docker-compose -f docker-compose.yml -f docker-compose.dev.yml up

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	cd log-ingestor && go test ./internal/ingest/... -coverprofile=coverage.out
	cd query-interface && go test ./internal/api/... -coverprofile=coverage.out
	cd tests && go test -v

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Install dependencies
deps:
	go mod download
	cd tests && go mod download

# Setup development environment
setup: deps
	@echo "Setting up development environment..."
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

# Quick test of the system
quick-test:
	@echo "Running quick system test..."
	@echo "1. Testing log ingestion..."
	curl -X POST http://localhost:3000/logs \
		-H "Content-Type: application/json" \
		-d '{"level":"info","message":"Quick test log","resourceId":"test-server","timestamp":"'$(shell date -u +%Y-%m-%dT%H:%M:%SZ)'"}' \
		| jq .
	@echo "2. Waiting for indexing..."
	sleep 5
	@echo "3. Testing search..."
	curl -s "http://localhost:4000/search?limit=5" | jq .
	@echo "Quick test completed!"

# Help
help:
	@echo "Available commands:"
	@echo "  up              - Start all services"
	@echo "  down            - Stop all services"
	@echo "  restart         - Restart all services"
	@echo "  logs            - View all service logs"
	@echo "  health          - Check service health"
	@echo "  mock            - Run load tests"
	@echo "  reset           - Reset database and run migrations"
	@echo "  clean           - Clean up containers and volumes"
	@echo "  test            - Run all tests"
	@echo "  test-unit       - Run unit tests"
	@echo "  test-integration- Run integration tests"
	@echo "  build           - Build all services"
	@echo "  dev             - Start services in development mode"
	@echo "  test-coverage   - Run tests with coverage"
	@echo "  fmt             - Format code"
	@echo "  lint            - Lint code"
	@echo "  deps            - Install dependencies"
	@echo "  setup           - Setup development environment"
	@echo "  quick-test      - Run quick system test"
	@echo "  help            - Show this help"