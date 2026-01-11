"""
Tests for GraphManager
"""

import pytest
from graph.graph_manager import GraphManager


@pytest.fixture
def graph_manager():
    """Create a fresh graph manager for each test"""
    return GraphManager(persistent=False)


def test_add_transaction(graph_manager):
    """Test adding a transaction to the graph"""
    success = graph_manager.add_transaction(
        tx_hash="0xabc123",
        from_address="TFromAddress123",
        to_address="TToAddress456",
        amount="100.50",
        timestamp=1704067200,  # 2024-01-01 00:00:00
        block_number=12345,
        contract="TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
    )

    assert success is True
    stats = graph_manager.get_statistics()
    assert stats["transaction_count"] == 1
    assert stats["node_count"] == 2
    assert stats["edge_count"] == 1


def test_add_multiple_transactions(graph_manager):
    """Test adding multiple transactions"""
    transactions = [
        {
            "tx_hash": "0xabc123",
            "from_address": "TAddr1",
            "to_address": "TAddr2",
            "amount": "100",
            "timestamp": 1704067200,
            "block_number": 12345,
            "contract": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
        },
        {
            "tx_hash": "0xdef456",
            "from_address": "TAddr2",
            "to_address": "TAddr3",
            "amount": "50",
            "timestamp": 1704067260,  # +60 seconds
            "block_number": 12346,
            "contract": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
        },
        {
            "tx_hash": "0xghi789",
            "from_address": "TAddr3",
            "to_address": "TAddr1",
            "amount": "25",
            "timestamp": 1704067320,  # +120 seconds
            "block_number": 12347,
            "contract": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
        }
    ]

    for tx in transactions:
        success = graph_manager.add_transaction(**tx)
        assert success is True

    stats = graph_manager.get_statistics()
    assert stats["transaction_count"] == 3
    assert stats["node_count"] == 3
    assert stats["edge_count"] == 3


def test_get_node_info(graph_manager):
    """Test getting node information"""
    # Add transaction
    graph_manager.add_transaction(
        tx_hash="0xabc123",
        from_address="TFromAddr",
        to_address="TToAddr",
        amount="100.5",
        timestamp=1704067200,
        block_number=12345,
        contract="TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
    )

    # Get info for sender
    from_info = graph_manager.get_node_info("TFromAddr")
    assert from_info is not None
    assert from_info["address"] == "TFromAddr"
    assert from_info["sent_count"] == 1
    assert from_info["received_count"] == 0
    assert from_info["total_sent"] == 100.5

    # Get info for receiver
    to_info = graph_manager.get_node_info("TToAddr")
    assert to_info is not None
    assert to_info["address"] == "TToAddr"
    assert to_info["sent_count"] == 0
    assert to_info["received_count"] == 1
    assert to_info["total_received"] == 100.5


def test_get_node_info_nonexistent(graph_manager):
    """Test getting info for nonexistent node"""
    info = graph_manager.get_node_info("TNonexistent")
    assert info is None


def test_get_neighbors(graph_manager):
    """Test getting neighbors"""
    # Add transactions to create a graph: A -> B -> C
    graph_manager.add_transaction(
        tx_hash="0x1",
        from_address="TAddrA",
        to_address="TAddrB",
        amount="100",
        timestamp=1704067200,
        block_number=12345,
        contract="TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
    )
    graph_manager.add_transaction(
        tx_hash="0x2",
        from_address="TAddrB",
        to_address="TAddrC",
        amount="50",
        timestamp=1704067260,
        block_number=12346,
        contract="TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
    )

    # Test outgoing neighbors of A
    neighbors_a_out = graph_manager.get_neighbors("TAddrA", direction="out")
    assert "TAddrB" in neighbors_a_out
    assert len(neighbors_a_out) == 1

    # Test incoming neighbors of C
    neighbors_c_in = graph_manager.get_neighbors("TAddrC", direction="in")
    assert "TAddrB" in neighbors_c_in
    assert len(neighbors_c_in) == 1

    # Test both directions for B
    neighbors_b = graph_manager.get_neighbors("TAddrB", direction="both")
    assert "TAddrA" in neighbors_b
    assert "TAddrC" in neighbors_b
    assert len(neighbors_b) == 2


def test_find_paths(graph_manager):
    """Test finding paths between nodes"""
    # Create path: A -> B -> C -> D
    transactions = [
        ("0x1", "TAddrA", "TAddrB", 1704067200),
        ("0x2", "TAddrB", "TAddrC", 1704067260),
        ("0x3", "TAddrC", "TAddrD", 1704067320),
    ]

    for tx_hash, from_addr, to_addr, timestamp in transactions:
        graph_manager.add_transaction(
            tx_hash=tx_hash,
            from_address=from_addr,
            to_address=to_addr,
            amount="100",
            timestamp=timestamp,
            block_number=12345,
            contract="TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
        )

    # Find path from A to D
    paths = graph_manager.find_paths("TAddrA", "TAddrD", max_depth=5)
    assert len(paths) > 0
    assert ["TAddrA", "TAddrB", "TAddrC", "TAddrD"] in paths


def test_get_transactions_in_window(graph_manager):
    """Test getting transactions in a time window"""
    # Add transactions with different timestamps
    transactions = [
        ("0x1", 1704067200),  # 2024-01-01 00:00:00
        ("0x2", 1704067260),  # 2024-01-01 00:01:00
        ("0x3", 1704067320),  # 2024-01-01 00:02:00
        ("0x4", 1704067380),  # 2024-01-01 00:03:00
    ]

    for tx_hash, timestamp in transactions:
        graph_manager.add_transaction(
            tx_hash=tx_hash,
            from_address="TFrom",
            to_address="TTo",
            amount="100",
            timestamp=timestamp,
            block_number=12345,
            contract="TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
        )

    # Query first 2 minutes
    txs = graph_manager.get_transactions_in_window(
        start_time=1704067200,
        end_time=1704067320,
        limit=100
    )

    # Should get transactions at 00:00 and 00:01, but not 00:02
    assert len(txs) >= 2


def test_clear_graph(graph_manager):
    """Test clearing the graph"""
    # Add transaction
    graph_manager.add_transaction(
        tx_hash="0xabc",
        from_address="TFrom",
        to_address="TTo",
        amount="100",
        timestamp=1704067200,
        block_number=12345,
        contract="TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
    )

    # Verify it was added
    stats = graph_manager.get_statistics()
    assert stats["transaction_count"] == 1

    # Clear the graph
    graph_manager.clear()

    # Verify it's empty
    stats = graph_manager.get_statistics()
    assert stats["transaction_count"] == 0
    assert stats["node_count"] == 0


def test_get_statistics(graph_manager):
    """Test getting graph statistics"""
    stats = graph_manager.get_statistics()

    assert "node_count" in stats
    assert "edge_count" in stats
    assert "transaction_count" in stats
    assert "persistent" in stats
    assert stats["persistent"] is False
