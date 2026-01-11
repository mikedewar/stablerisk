"""
Tests for FastAPI endpoints
"""

import pytest
from fastapi.testclient import TestClient
from api.server import app


@pytest.fixture
def client():
    """Create test client"""
    return TestClient(app)


def test_health_check(client):
    """Test health check endpoint"""
    response = client.get("/health")
    assert response.status_code == 200
    data = response.json()
    assert data["status"] == "healthy"
    assert "timestamp" in data
    assert "graph_stats" in data


def test_add_transaction(client):
    """Test adding a transaction"""
    transaction = {
        "tx_hash": "0xabc123",
        "from": "TFromAddress123",
        "to": "TToAddress456",
        "amount": "100.50",
        "timestamp": 1704067200,
        "block_number": 12345,
        "contract": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
    }

    response = client.post("/graph/transaction", json=transaction)
    assert response.status_code == 201
    data = response.json()
    assert data["success"] is True
    assert "message" in data


def test_get_node_info(client):
    """Test getting node information"""
    # First add a transaction
    transaction = {
        "tx_hash": "0xabc",
        "from": "TTestFrom",
        "to": "TTestTo",
        "amount": "100",
        "timestamp": 1704067200,
        "block_number": 12345,
        "contract": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
    }
    client.post("/graph/transaction", json=transaction)

    # Get node info
    response = client.get("/graph/node/TTestFrom")
    assert response.status_code == 200
    data = response.json()
    assert data["address"] == "TTestFrom"
    assert "transaction_count" in data
    assert "total_sent" in data


def test_get_node_info_not_found(client):
    """Test getting info for nonexistent node"""
    response = client.get("/graph/node/TNonexistent")
    assert response.status_code == 404


def test_get_neighbors(client):
    """Test getting neighbors"""
    # Add transactions
    transactions = [
        {
            "tx_hash": "0x1",
            "from": "TAddr1",
            "to": "TAddr2",
            "amount": "100",
            "timestamp": 1704067200,
            "block_number": 12345,
            "contract": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
        },
        {
            "tx_hash": "0x2",
            "from": "TAddr2",
            "to": "TAddr3",
            "amount": "50",
            "timestamp": 1704067260,
            "block_number": 12346,
            "contract": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
        }
    ]

    for tx in transactions:
        client.post("/graph/transaction", json=tx)

    # Get neighbors
    response = client.get("/graph/neighbors/TAddr2?direction=both")
    assert response.status_code == 200
    data = response.json()
    assert data["address"] == "TAddr2"
    assert len(data["neighbors"]) == 2
    assert "TAddr1" in data["neighbors"]
    assert "TAddr3" in data["neighbors"]


def test_find_paths(client):
    """Test finding paths"""
    # Add path: A -> B -> C
    transactions = [
        {
            "tx_hash": "0x1",
            "from": "TAddrA",
            "to": "TAddrB",
            "amount": "100",
            "timestamp": 1704067200,
            "block_number": 12345,
            "contract": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
        },
        {
            "tx_hash": "0x2",
            "from": "TAddrB",
            "to": "TAddrC",
            "amount": "50",
            "timestamp": 1704067260,
            "block_number": 12346,
            "contract": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
        }
    ]

    for tx in transactions:
        client.post("/graph/transaction", json=tx)

    # Find paths
    response = client.get("/graph/paths?from=TAddrA&to=TAddrC&max_depth=3")
    assert response.status_code == 200
    data = response.json()
    assert data["from"] == "TAddrA"
    assert data["to"] == "TAddrC"
    assert len(data["paths"]) > 0


def test_get_window_transactions(client):
    """Test getting transactions in window"""
    # Add transactions
    transaction = {
        "tx_hash": "0xwindow",
        "from": "TFrom",
        "to": "TTo",
        "amount": "100",
        "timestamp": 1704067200,
        "block_number": 12345,
        "contract": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
    }
    client.post("/graph/transaction", json=transaction)

    # Query window
    response = client.get("/graph/window?start=1704067000&end=1704067300")
    assert response.status_code == 200
    data = response.json()
    assert isinstance(data, list)


def test_get_window_invalid_range(client):
    """Test invalid time range"""
    response = client.get("/graph/window?start=1704067300&end=1704067000")
    assert response.status_code == 400


def test_get_statistics(client):
    """Test getting graph statistics"""
    response = client.get("/graph/statistics")
    assert response.status_code == 200
    data = response.json()
    assert "node_count" in data
    assert "edge_count" in data
    assert "transaction_count" in data


def test_clear_graph(client):
    """Test clearing the graph"""
    # Add a transaction first
    transaction = {
        "tx_hash": "0xtest",
        "from": "TFrom",
        "to": "TTo",
        "amount": "100",
        "timestamp": 1704067200,
        "block_number": 12345,
        "contract": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
    }
    client.post("/graph/transaction", json=transaction)

    # Clear
    response = client.delete("/graph/clear")
    assert response.status_code == 200
    data = response.json()
    assert data["success"] is True

    # Verify it's cleared
    stats = client.get("/graph/statistics").json()
    assert stats["transaction_count"] == 0
