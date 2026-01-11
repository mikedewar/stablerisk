# Raphtory Service

Temporal graph service for USDT transaction analysis using Raphtory.

## Features

- Temporal graph storage of USDT transactions
- FastAPI REST API for graph operations
- Node and edge queries with temporal filtering
- Path finding between addresses
- Neighbor discovery
- Graph statistics and analytics

## Installation

```bash
# Install dependencies
pip install -r requirements.txt

# Or using Docker
docker build -t raphtory-service .
```

## Running

### Local Development

```bash
# Set environment variables
export HOST=0.0.0.0
export PORT=8000
export LOG_LEVEL=info

# Run the service
python main.py
```

### Using Docker

```bash
# Build image
docker build -t raphtory-service .

# Run container
docker run -p 8000:8000 raphtory-service
```

### Using Docker Compose

From the project root:

```bash
docker-compose -f deployments/docker-compose.yml up raphtory
```

## API Endpoints

### Health Check

```
GET /health
```

Returns service health status and graph statistics.

### Add Transaction

```
POST /graph/transaction
```

Add a transaction to the temporal graph.

**Request Body:**
```json
{
  "tx_hash": "0xabc123",
  "from": "TFromAddress",
  "to": "TToAddress",
  "amount": "100.50",
  "timestamp": 1704067200,
  "block_number": 12345,
  "contract": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
}
```

### Get Node Information

```
GET /graph/node/{address}
```

Get information about a node (address).

**Response:**
```json
{
  "address": "TAddress",
  "first_seen": 1704067200,
  "last_seen": 1704067500,
  "transaction_count": 10,
  "sent_count": 5,
  "received_count": 5,
  "total_sent": 500.0,
  "total_received": 450.0,
  "balance_flow": -50.0
}
```

### Get Transactions in Time Window

```
GET /graph/window?start=1704067000&end=1704070600&limit=1000
```

Get transactions within a time window.

### Get Neighbors

```
GET /graph/neighbors/{address}?direction=both
```

Get neighboring addresses. Direction can be "in", "out", or "both".

### Find Paths

```
GET /graph/paths?from=TAddr1&to=TAddr2&max_depth=3
```

Find paths between two addresses.

### Get Statistics

```
GET /graph/statistics
```

Get graph statistics (node count, edge count, etc.).

### Save Snapshot

```
POST /graph/snapshot?filename=snapshot_name
```

Save graph snapshot to disk (only for persistent graphs).

### Clear Graph

```
DELETE /graph/clear
```

Clear the graph (for testing only).

## Testing

```bash
# Install test dependencies
pip install pytest pytest-asyncio pytest-cov

# Run tests
pytest

# Run with coverage
pytest --cov=. --cov-report=html
```

## Configuration

Configuration via environment variables:

- `HOST`: Server host (default: 0.0.0.0)
- `PORT`: Server port (default: 8000)
- `LOG_LEVEL`: Logging level (default: info)
- `WORKERS`: Number of worker processes (default: 1)
- `RELOAD`: Enable auto-reload (default: false)
- `PERSISTENT_GRAPH`: Use persistent graph storage (default: false)
- `SNAPSHOT_DIR`: Directory for snapshots (default: /tmp/raphtory_snapshots)

## Development

### Project Structure

```
raphtory-service/
├── api/
│   ├── __init__.py
│   ├── models.py          # Pydantic models
│   └── server.py          # FastAPI app
├── graph/
│   ├── __init__.py
│   └── graph_manager.py   # Raphtory graph manager
├── tests/
│   ├── __init__.py
│   ├── conftest.py
│   ├── test_api.py
│   └── test_graph_manager.py
├── main.py                # Entry point
├── requirements.txt       # Dependencies
├── Dockerfile             # Container image
└── README.md
```

### Adding New Features

1. Add graph operations to `graph/graph_manager.py`
2. Add API models to `api/models.py`
3. Add endpoints to `api/server.py`
4. Add tests to `tests/`

## Performance

- In-memory graph for fast queries
- Temporal indexing for efficient time-based queries
- Configurable limits on query results
- Optional persistent storage for large graphs

## License

[Specify license]
