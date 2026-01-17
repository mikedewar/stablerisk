# Integration Tests

This directory contains integration tests for the StableRisk project.

## Docker Compose Integration Test

The `docker-compose-test.sh` script verifies that all services build and run correctly using Docker Compose.

### What It Tests

1. **Prerequisites**: Verifies Docker and Docker Compose are installed and running
2. **Build**: Builds all services from scratch (no cache)
3. **Startup**: Starts all services in detached mode
4. **Health Checks**: Verifies each service becomes healthy:
   - PostgreSQL: Connection and query execution
   - Raphtory: HTTP health endpoint on port 8000
   - Monitor: Container stays running
   - API: HTTP health endpoint on port 8080
   - Web: HTTP response on port 3000
5. **Cleanup**: Removes all containers, networks, and volumes

### Prerequisites

- Docker installed and running
- Docker Compose installed
- `.env` file (will be created from `.env.example` if missing)
- At least 8GB of available disk space
- At least 4GB of available RAM

### Usage

From the project root:

```bash
# Run the test
./tests/integration/docker-compose-test.sh
```

The script will:
- Print colored output (info in blue, success in green, errors in red)
- Show progress for long-running operations
- Display logs if any service fails
- Clean up all resources when done (success or failure)
- Return exit code 0 on success, 1 on failure

### Expected Duration

- First run (with build): 5-10 minutes
- Subsequent runs (with cached images): 2-5 minutes

### Troubleshooting

If the test fails:

1. **Check Docker resources**: Ensure Docker has enough memory/disk
   ```bash
   docker system df
   docker system prune
   ```

2. **View detailed logs**: Check `/tmp/docker-build.log` for build errors

3. **Check service logs**:
   ```bash
   docker-compose -f deployments/docker-compose.yml logs [service-name]
   ```

4. **Manual cleanup** (if script cleanup fails):
   ```bash
   docker-compose -f deployments/docker-compose.yml down -v
   docker system prune -f
   ```

5. **Common issues**:
   - **Port conflicts**: Another service using 3000, 5432, 8000, or 8080
   - **Memory**: Services fail to start due to insufficient RAM
   - **Network**: TronGrid API key invalid or network issues

### CI/CD Integration

This test is designed to run in CI/CD pipelines:

```yaml
# Example GitHub Actions
- name: Run integration tests
  run: ./tests/integration/docker-compose-test.sh
```

```yaml
# Example GitLab CI
test:integration:
  script:
    - ./tests/integration/docker-compose-test.sh
```

### Exit Codes

- `0`: All tests passed
- `1`: One or more tests failed

### Test Output

```
[INFO] === Docker Compose Integration Test ===
[INFO] Starting at Mon Jan 11 13:45:00 UTC 2026

[INFO] Checking prerequisites...
[SUCCESS] Prerequisites check passed

[INFO] Building Docker images (this may take several minutes)...
[SUCCESS] All services built successfully

[INFO] Starting services...
[SUCCESS] Services started

[INFO] Testing PostgreSQL...
[INFO] Waiting for postgres to become healthy...
[SUCCESS] postgres is healthy (10s)
[SUCCESS] PostgreSQL connection test passed

[INFO] Testing Raphtory service...
[INFO] Waiting for raphtory to become healthy...
[SUCCESS] raphtory is healthy (15s)
[SUCCESS] Raphtory health check passed

...

[INFO] === Test Summary ===
[INFO] Total tests: 7
[SUCCESS] Passed: 7
[SUCCESS] All tests passed!
```

## Adding More Integration Tests

To add new integration tests:

1. Create a new `.sh` script in this directory
2. Follow the pattern in `docker-compose-test.sh`:
   - Proper error handling with `set -e`
   - Cleanup trap
   - Colored logging functions
   - Clear success/failure reporting
3. Make it executable: `chmod +x your-test.sh`
4. Document it in this README
5. Add to CI/CD pipeline

## Notes

- These tests are destructive (tear down services after completion)
- Not suitable for running against production environments
- Use a development/test environment only
- Tests run with `--no-cache` to ensure clean builds
