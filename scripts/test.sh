#!/bin/bash

# Logito Test Runner Script
# This script runs all tests for the Logito log management system

set -e

echo "🧪 Logito Test Runner"
echo "===================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if services are running
check_services() {
    print_status "Checking if services are running..."
    
    local services_running=true
    
    # Check log ingestor
    if ! curl -s http://localhost:3000/health > /dev/null 2>&1; then
        print_warning "Log ingestor service is not running on port 3000"
        services_running=false
    fi
    
    # Check query interface
    if ! curl -s http://localhost:4000/health > /dev/null 2>&1; then
        print_warning "Query interface service is not running on port 4000"
        services_running=false
    fi
    
    # Check elasticsearch
    if ! curl -s http://localhost:9200/_cluster/health > /dev/null 2>&1; then
        print_warning "Elasticsearch service is not running on port 9200"
        services_running=false
    fi
    
    if [ "$services_running" = false ]; then
        print_warning "Some services are not running. Integration tests will be skipped."
        print_status "To start services, run: docker-compose up -d"
        return 1
    fi
    
    print_success "All services are running!"
    return 0
}

# Run unit tests
run_unit_tests() {
    print_status "Running unit tests..."
    
    # Log ingestor unit tests
    print_status "Running log ingestor unit tests..."
    cd log-ingestor
    if go test ./internal/ingest/... -v; then
        print_success "Log ingestor unit tests passed"
    else
        print_error "Log ingestor unit tests failed"
        return 1
    fi
    cd ..
    
    # Query interface unit tests
    print_status "Running query interface unit tests..."
    cd query-interface
    if go test ./internal/api/... -v; then
        print_success "Query interface unit tests passed"
    else
        print_error "Query interface unit tests failed"
        return 1
    fi
    cd ..
    
    print_success "All unit tests passed!"
    return 0
}

# Run integration tests
run_integration_tests() {
    print_status "Running integration tests..."
    
    cd tests
    if go test -v; then
        print_success "Integration tests passed"
    else
        print_error "Integration tests failed"
        return 1
    fi
    cd ..
    
    return 0
}

# Main execution
main() {
    local run_integration=true
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --unit-only)
                run_integration=false
                shift
                ;;
            --integration-only)
                # Skip unit tests, only run integration
                shift
                ;;
            --help)
                echo "Usage: $0 [OPTIONS]"
                echo "Options:"
                echo "  --unit-only        Run only unit tests"
                echo "  --integration-only Run only integration tests"
                echo "  --help             Show this help message"
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                echo "Use --help for usage information"
                exit 1
                ;;
        esac
    done
    
    # Run unit tests
    if ! run_unit_tests; then
        print_error "Unit tests failed. Exiting."
        exit 1
    fi
    
    # Run integration tests if requested and services are running
    if [ "$run_integration" = true ]; then
        if check_services; then
            if ! run_integration_tests; then
                print_error "Integration tests failed."
                exit 1
            fi
        else
            print_warning "Skipping integration tests due to missing services"
        fi
    fi
    
    print_success "All tests completed successfully! 🎉"
}

# Run main function
main "$@"
