# makefile for logito project

.PHONY: help build up down logs clean test reset truncate mock

# default target
help:
	@echo "available targets:"
	@echo "  build    - build docker images"
	@echo "  up       - start all services"
	@echo "  down     - stop all services"
	@echo "  logs     - show logs from all services"
	@echo "  clean    - remove containers and volumes"
	@echo "  test     - run tests"
	@echo "  reset    - reset entire system"
	@echo "  truncate - clear all logs"
	@echo "  mock     - generate mock data"

# build docker images
build:
	docker-compose build

# start all services
up:
	docker-compose up -d

# stop all services
down:
	docker-compose down

# show logs
logs:
	docker-compose logs -f

# clean up containers and volumes
clean:
	docker-compose down -v
	docker system prune -f

# run tests
test:
	cd log-ingestor && go test ./...
	cd query-interface && go test ./...

# reset entire system
reset:
	chmod +x scripts/reset.sh && ./scripts/reset.sh

# truncate all logs
truncate:
	chmod +x scripts/truncate.sh && ./scripts/truncate.sh

# generate mock data
mock:
	cd mockdata && go run main.go
