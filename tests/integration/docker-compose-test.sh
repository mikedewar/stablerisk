#!/bin/bash
#
# Docker Compose Integration Test
# Tests that all services build, start, and become healthy
#
# Usage: ./docker-compose-test.sh
# Exit codes: 0 = success, 1 = failure

set -e  # Exit on error
set -o pipefail  # Catch errors in pipes

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
COMPOSE_FILE="deployments/docker-compose.yml"
MAX_WAIT_TIME=300  # 5 minutes max wait for services
CHECK_INTERVAL=5   # Check every 5 seconds
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Change to project root
cd "$PROJECT_ROOT"

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Track test results
TESTS_PASSED=0
TESTS_FAILED=0

# Cleanup function
cleanup() {
    local exit_code=$?
    log_info "Cleaning up Docker resources..."

    # Stop and remove containers, networks, volumes
    docker-compose -f "$COMPOSE_FILE" down -v --remove-orphans 2>/dev/null || true

    # Remove any dangling images from this test
    docker image prune -f >/dev/null 2>&1 || true

    if [ $exit_code -eq 0 ]; then
        log_success "Cleanup completed"
        log_success "Tests passed: $TESTS_PASSED, Tests failed: $TESTS_FAILED"
    else
        log_error "Cleanup completed after failure"
        log_error "Tests passed: $TESTS_PASSED, Tests failed: $TESTS_FAILED"
    fi

    exit $exit_code
}

# Register cleanup on exit
trap cleanup EXIT INT TERM

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed or not in PATH"
        exit 1
    fi

    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose is not installed or not in PATH"
        exit 1
    fi

    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running"
        exit 1
    fi

    if [ ! -f "$COMPOSE_FILE" ]; then
        log_error "Docker Compose file not found: $COMPOSE_FILE"
        exit 1
    fi

    if [ ! -f ".env" ]; then
        log_warning ".env file not found, creating from .env.example"
        cp .env.example .env
    fi

    log_success "Prerequisites check passed"
}

# Build all services
build_services() {
    log_info "Building Docker images (this may take several minutes)..."

    if docker-compose -f "$COMPOSE_FILE" build --no-cache 2>&1 | tee /tmp/docker-build.log; then
        log_success "All services built successfully"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        log_error "Docker build failed. Check /tmp/docker-build.log for details"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Start all services
start_services() {
    log_info "Starting services..."

    if docker-compose -f "$COMPOSE_FILE" up -d 2>&1; then
        log_success "Services started"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        log_error "Failed to start services"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Wait for a service to become healthy
wait_for_service() {
    local service_name=$1
    local check_command=$2
    local elapsed=0

    log_info "Waiting for $service_name to become healthy..."

    while [ $elapsed -lt $MAX_WAIT_TIME ]; do
        if eval "$check_command" &>/dev/null; then
            log_success "$service_name is healthy (${elapsed}s)"
            return 0
        fi

        sleep $CHECK_INTERVAL
        elapsed=$((elapsed + CHECK_INTERVAL))

        # Show progress
        if [ $((elapsed % 30)) -eq 0 ]; then
            log_info "Still waiting for $service_name... (${elapsed}s elapsed)"
        fi
    done

    log_error "$service_name did not become healthy within ${MAX_WAIT_TIME}s"
    log_info "Container status:"
    docker-compose -f "$COMPOSE_FILE" ps
    log_info "Recent logs:"
    docker-compose -f "$COMPOSE_FILE" logs --tail=50 "$service_name" || true
    return 1
}

# Test PostgreSQL
test_postgres() {
    log_info "Testing PostgreSQL..."

    if wait_for_service "postgres" \
        "docker-compose -f $COMPOSE_FILE exec -T postgres pg_isready -U stablerisk"; then

        # Additional connection test
        if docker-compose -f "$COMPOSE_FILE" exec -T postgres \
            psql -U stablerisk -d stablerisk -c "SELECT 1" &>/dev/null; then
            log_success "PostgreSQL connection test passed"
            TESTS_PASSED=$((TESTS_PASSED + 1))
            return 0
        else
            log_error "PostgreSQL connection failed"
            TESTS_FAILED=$((TESTS_FAILED + 1))
            return 1
        fi
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Test Raphtory
test_raphtory() {
    log_info "Testing Raphtory service..."

    if wait_for_service "raphtory" \
        "curl -sf http://localhost:8000/health"; then

        # Verify health endpoint response
        response=$(curl -s http://localhost:8000/health)
        if echo "$response" | grep -q "healthy\|ok\|running"; then
            log_success "Raphtory health check passed"
            TESTS_PASSED=$((TESTS_PASSED + 1))
            return 0
        else
            log_warning "Raphtory health endpoint returned unexpected response: $response"
            # Still count as success if endpoint responds
            TESTS_PASSED=$((TESTS_PASSED + 1))
            return 0
        fi
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Test Monitor service
test_monitor() {
    log_info "Testing Monitor service..."

    # Monitor doesn't have a health endpoint, so check if container is running
    if wait_for_service "monitor" \
        "docker-compose -f $COMPOSE_FILE ps monitor | grep -q 'Up'"; then

        # Check logs for any immediate crashes
        sleep 5
        if docker-compose -f "$COMPOSE_FILE" ps monitor | grep -q "Up"; then
            log_success "Monitor service is running"
            TESTS_PASSED=$((TESTS_PASSED + 1))
            return 0
        else
            log_error "Monitor service crashed after startup"
            TESTS_FAILED=$((TESTS_FAILED + 1))
            return 1
        fi
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Test API service
test_api() {
    log_info "Testing API service..."

    if wait_for_service "api" \
        "curl -sf http://localhost:8080/health"; then

        # Verify health endpoint response
        response=$(curl -s http://localhost:8080/health)
        if echo "$response" | grep -q "healthy\|ok"; then
            log_success "API health check passed"
            log_info "API response: $response"
            TESTS_PASSED=$((TESTS_PASSED + 1))
            return 0
        else
            log_error "API health endpoint returned unexpected response: $response"
            TESTS_FAILED=$((TESTS_FAILED + 1))
            return 1
        fi
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Test Web service
test_web() {
    log_info "Testing Web service..."

    # Web service runs on port 3000
    if wait_for_service "web" \
        "curl -sf http://localhost:3000"; then
        log_success "Web service is responding"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Main test execution
main() {
    log_info "=== Docker Compose Integration Test ==="
    log_info "Starting at $(date)"
    echo

    # Run checks
    check_prerequisites
    echo

    # Build services
    if ! build_services; then
        log_error "Build failed, aborting tests"
        exit 1
    fi
    echo

    # Start services
    if ! start_services; then
        log_error "Failed to start services, aborting tests"
        exit 1
    fi
    echo

    # Test each service
    test_postgres
    echo

    test_raphtory
    echo

    test_monitor
    echo

    test_api
    echo

    test_web
    echo

    # Final summary
    log_info "=== Test Summary ==="
    log_info "Total tests: $((TESTS_PASSED + TESTS_FAILED))"
    log_success "Passed: $TESTS_PASSED"

    if [ $TESTS_FAILED -gt 0 ]; then
        log_error "Failed: $TESTS_FAILED"
        log_error "Some tests failed!"
        exit 1
    else
        log_success "All tests passed!"
        exit 0
    fi
}

# Run main function
main "$@"
