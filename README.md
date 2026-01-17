# StableRisk - USDT Transaction Monitoring Service

A production-ready Go-based service for monitoring USDT (TRC20) stablecoin transactions on the Tron blockchain in real-time, detecting statistical anomalies, and exposing insights via REST API and web dashboard.

## Features

- Real-time USDT transaction monitoring on Tron blockchain via TronGrid WebSocket
- Temporal graph analysis using Raphtory for pattern detection
- Statistical anomaly detection (Z-score and IQR methods)
- Graph-based pattern detection (circulation, fan-out, fan-in, dormant awakening, velocity)
- RESTful API with JWT authentication and RBAC
- Real-time WebSocket updates for GUI
- Modern web dashboard with graph visualizations
- Comprehensive testing (unit, contract, integration)
- Docker containerization with multi-stage builds
- ISO27001 and PCI-DSS compliant security controls

## Architecture

```
Tron Blockchain → Monitor Service → Raphtory Graph → Detection Engine → API → Web Dashboard
```

### Components

- **Monitor Service** (Go): Connects to TronGrid WebSocket, parses USDT transactions, sends to Raphtory
- **Raphtory Service** (Python): Temporal graph database with REST API
- **API Service** (Go): Anomaly detection, REST API, WebSocket hub, authentication
- **Web Dashboard** (Svelte): Real-time visualization and monitoring interface
- **PostgreSQL**: Audit logs, user management, outlier cache
- **Nginx**: Reverse proxy with TLS termination

## Prerequisites

- Docker and Docker Compose
- Go 1.23+ (for local development)
- Python 3.11+ (for Raphtory service development)
- Node.js 20+ (for web dashboard development)
- TronGrid API Key (get from https://www.trongrid.io/)

## Quick Start

### 1. Clone and Configure

```bash
# Clone the repository
git clone <repository-url>
cd stablerisk

# Copy environment template
cp .env.example .env

# Edit .env and set required secrets:
# - TRONGRID_API_KEY
# - POSTGRES_PASSWORD
# - JWT_SECRET
# - ENCRYPTION_KEY
# - HMAC_KEY
nano .env
```

### 2. Generate Secrets

```bash
# Generate encryption keys
openssl rand -base64 32  # Use for ENCRYPTION_KEY
openssl rand -base64 32  # Use for HMAC_KEY
openssl rand -base64 32  # Use for JWT_SECRET
```

### 3. Start Services

```bash
# Build and start all services
docker-compose -f deployments/docker-compose.yml up -d

# Check service health
docker-compose -f deployments/docker-compose.yml ps

# View logs
docker-compose -f deployments/docker-compose.yml logs -f
```

### 4. Access Services

- **Web Dashboard**: http://localhost:3000
- **API**: http://localhost:8080
- **API Documentation**: http://localhost:8080/api/v1/docs (coming soon)
- **Metrics**: http://localhost:9090/metrics
- **Health Check**: http://localhost:8080/health

### 5. Default Users

For development purposes, default users are created:

| Username | Password | Role |
|----------|----------|------|
| admin | admin123456 | admin |
| analyst | analyst123456 | analyst |
| viewer | viewer123456 | viewer |

**IMPORTANT**: Change these passwords in production!

## Development

### Project Structure

```
stablerisk/
├── cmd/                    # Application entry points
│   ├── api/               # API server
│   └── monitor/           # Blockchain monitor
├── internal/              # Private application code
│   ├── blockchain/        # TronGrid integration
│   ├── detection/         # Anomaly detection
│   ├── api/              # REST API handlers
│   ├── security/          # Security and compliance
│   └── ...
├── pkg/                   # Public libraries
├── raphtory-service/      # Python Raphtory service
├── web/                   # Svelte dashboard
├── tests/                 # Test suites
├── migrations/            # Database migrations
└── deployments/           # Docker and K8s configs
```

### Local Development Setup

#### Backend (Go)

```bash
# Install dependencies
go mod download

# Run tests
go test ./...

# Run with coverage
go test -cover -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Build API service
go build -o bin/api ./cmd/api

# Build monitor service
go build -o bin/monitor ./cmd/monitor

# Run locally (requires PostgreSQL and Raphtory)
./bin/api
./bin/monitor
```

#### Raphtory Service (Python)

```bash
cd raphtory-service

# Create virtual environment
python3 -m venv venv
source venv/bin/activate

# Install dependencies
pip install -r requirements.txt

# Run service
python main.py

# Or with uvicorn
uvicorn api.server:app --reload --host 0.0.0.0 --port 8000
```

#### Web Dashboard (Svelte)

```bash
cd web

# Install dependencies
npm install

# Run development server
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview
```

### Running Tests

```bash
# Unit tests
go test ./internal/...

# Integration tests
go test ./tests/integration/...

# Contract tests
go test ./tests/contract/...

# All tests with coverage
./scripts/test.sh

# Specific package
go test -v ./internal/detection/

# Docker Compose integration test (verifies full stack)
./tests/integration/docker-compose-test.sh
```

**Docker Compose Integration Test**: This test verifies that all services build, start, and become healthy. It's recommended to run this test before deploying or after making changes to Docker configurations. See `tests/integration/README.md` for details.

## API Documentation

### Authentication

All API endpoints (except `/health`) require JWT authentication.

```bash
# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123456"}'

# Response includes JWT token
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "...",
  "expires_in": 3600
}

# Use token in subsequent requests
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/v1/outliers
```

### Key Endpoints

#### Outliers

```bash
# List outliers
GET /api/v1/outliers?page=1&limit=20&type=zscore&severity=high

# Get outlier details
GET /api/v1/outliers/:id

# Acknowledge outlier
POST /api/v1/outliers/:id/acknowledge
```

#### Transactions

```bash
# Query transactions
GET /api/v1/transactions?address=TR7...&from=2024-01-01&to=2024-01-31

# Get transaction details
GET /api/v1/transactions/:hash
```

#### Statistics

```bash
# Get summary statistics
GET /api/v1/stats/summary

# Get top addresses by volume
GET /api/v1/stats/addresses?limit=100
```

#### WebSocket

```bash
# Connect to WebSocket (requires token in query or header)
ws://localhost:8080/api/v1/ws?token=<jwt_token>

# Message format
{
  "type": "outlier",
  "data": { ... },
  "timestamp": "2024-01-01T00:00:00Z"
}
```

## Configuration

Configuration is managed via:
1. `internal/config/config.yaml` (default values)
2. Environment variables (override config file)

Environment variables use the prefix `STABLERISK_` and follow the config structure:

```bash
# Example: Override API port
STABLERISK_SERVER_API_PORT=8080

# Example: Override detection threshold
STABLERISK_DETECTION_ZSCORE_THRESHOLD=3.5
```

See `.env.example` for all available configuration options.

## Security

### Compliance

StableRisk implements security controls for:
- **ISO27001**: Access control, cryptography, operations security
- **PCI-DSS**: Data protection, encryption, authentication, audit logging

See `docs/compliance/` for detailed control mappings.

### Security Features

- **Encryption**: AES-256-GCM for data at rest, TLS 1.3 for data in transit
- **Authentication**: JWT with short-lived tokens (1 hour) and refresh tokens (7 days)
- **Authorization**: Role-based access control (RBAC)
- **Audit Logging**: Tamper-proof logs with HMAC signatures
- **Password Hashing**: bcrypt with cost factor 12
- **Rate Limiting**: Prevents brute force attacks
- **Input Validation**: Strict validation on all inputs

### Key Rotation

Rotate secrets every 90 days:

```bash
# Generate new keys
openssl rand -base64 32

# Update .env or secrets manager
# Restart services
docker-compose restart
```

## Monitoring

### Prometheus Metrics

Metrics are exposed at `http://localhost:9090/metrics`:

- `stablerisk_api_requests_total` - Total API requests
- `stablerisk_api_request_duration_seconds` - Request duration histogram
- `stablerisk_transactions_processed_total` - Transactions processed
- `stablerisk_outliers_detected_total` - Outliers detected by type
- `stablerisk_websocket_connections` - Active WebSocket connections

### Health Checks

```bash
# API health
curl http://localhost:8080/health

# Response
{
  "status": "healthy",
  "timestamp": "2024-01-01T00:00:00Z",
  "services": {
    "database": "healthy",
    "raphtory": "healthy",
    "trongrid": "connected"
  }
}
```

### Logs

All services use structured JSON logging:

```bash
# View API logs
docker-compose logs -f api

# View monitor logs
docker-compose logs -f monitor

# View all logs
docker-compose logs -f
```

## Deployment

### Docker Compose (Development)

```bash
docker-compose -f deployments/docker-compose.yml up -d
```

### Kubernetes (Production)

```bash
# Apply manifests
kubectl apply -f deployments/kubernetes/

# Check status
kubectl get pods -n stablerisk
kubectl get svc -n stablerisk

# View logs
kubectl logs -f deployment/stablerisk-api -n stablerisk
```

### Production Checklist

- [ ] Change all default passwords
- [ ] Generate and set strong secrets (JWT, encryption, HMAC keys)
- [ ] Enable TLS (set `TLS_ENABLED=true`, provide certificates)
- [ ] Configure database backups
- [ ] Set up monitoring and alerting
- [ ] Review and adjust detection thresholds
- [ ] Configure log retention policies
- [ ] Set up secret rotation schedule
- [ ] Whitelist Prometheus metrics endpoint
- [ ] Configure CORS for production domains
- [ ] Review and lock down network policies

## Troubleshooting

### Monitor Service Not Connecting

```bash
# Check TronGrid API key
docker-compose logs monitor | grep "TronGrid"

# Verify WebSocket connection
docker-compose exec monitor curl -v wss://api.trongrid.io
```

### Database Connection Issues

```bash
# Check PostgreSQL status
docker-compose ps postgres

# Connect to database
docker-compose exec postgres psql -U stablerisk -d stablerisk

# Check migrations
docker-compose exec postgres psql -U stablerisk -d stablerisk -c "SELECT * FROM audit_logs WHERE action='migration';"
```

### Raphtory Service Issues

```bash
# Check Raphtory logs
docker-compose logs raphtory

# Test Raphtory API
curl http://localhost:8000/health
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

[Specify your license here]

## Support

For issues, questions, or contributions:
- GitHub Issues: [Repository issues page]
- Documentation: `docs/`
- Email: [Contact email]

## Acknowledgments

- TronGrid API for Tron blockchain access
- Raphtory for temporal graph analysis
- Go, Python, and Svelte communities
