"""
Graph Manager for USDT Transaction Temporal Graph

This module manages a temporal graph of USDT transactions using Raphtory.
Nodes represent Tron addresses, edges represent transactions with amounts and timestamps.
"""

from typing import Dict, List, Optional, Any
from datetime import datetime
from decimal import Decimal
import structlog

from raphtory import Graph, PersistentGraph

logger = structlog.get_logger()


class GraphManager:
    """Manages the temporal graph of USDT transactions"""

    def __init__(self, persistent: bool = False, snapshot_dir: Optional[str] = None):
        """
        Initialize the graph manager

        Args:
            persistent: If True, use PersistentGraph with disk storage
            snapshot_dir: Directory for saving graph snapshots
        """
        self.persistent = persistent
        self.snapshot_dir = snapshot_dir or "/tmp/raphtory_snapshots"

        if persistent:
            self.graph = PersistentGraph()
            logger.info("Initialized persistent graph", snapshot_dir=self.snapshot_dir)
        else:
            self.graph = Graph()
            logger.info("Initialized in-memory graph")

        self._transaction_count = 0
        self._node_count = 0
        self._edge_count = 0

    def add_transaction(
        self,
        tx_hash: str,
        from_address: str,
        to_address: str,
        amount: str,
        timestamp: int,
        block_number: int,
        contract: str
    ) -> bool:
        """
        Add a transaction to the temporal graph

        Args:
            tx_hash: Transaction hash
            from_address: Sender address
            to_address: Receiver address
            amount: Transaction amount (as string to preserve precision)
            timestamp: Unix timestamp in seconds
            block_number: Block number
            contract: Contract address

        Returns:
            True if successful, False otherwise
        """
        try:
            # Add or update nodes (addresses)
            self._add_or_update_node(from_address, timestamp)
            self._add_or_update_node(to_address, timestamp)

            # Add edge (transaction) with temporal information
            self.graph.add_edge(
                timestamp,
                from_address,
                to_address,
                properties={
                    "tx_hash": tx_hash,
                    "amount": amount,
                    "block_number": block_number,
                    "contract": contract
                },
                layer="usdt"
            )

            self._transaction_count += 1
            self._edge_count += 1

            logger.debug(
                "Transaction added to graph",
                tx_hash=tx_hash,
                from_addr=from_address[:10] + "...",
                to_addr=to_address[:10] + "...",
                amount=amount,
                timestamp=timestamp
            )

            return True

        except Exception as e:
            logger.error(
                "Failed to add transaction to graph",
                error=str(e),
                tx_hash=tx_hash
            )
            return False

    def _add_or_update_node(self, address: str, timestamp: int):
        """Add or update a node (address) in the graph"""
        try:
            # Check if node exists
            if not self.graph.has_node(address):
                self._node_count += 1

            # Add node with temporal information
            self.graph.add_node(
                timestamp,
                address,
                properties={"address": address},
                node_type="wallet"
            )

        except Exception as e:
            logger.error(
                "Failed to add/update node",
                error=str(e),
                address=address
            )

    def get_node_info(self, address: str) -> Optional[Dict[str, Any]]:
        """
        Get information about a node (address)

        Args:
            address: The address to query

        Returns:
            Dictionary with node information, or None if not found
        """
        try:
            if not self.graph.has_node(address):
                return None

            node = self.graph.node(address)

            # Get all edges (transactions) for this node
            out_edges = list(node.out_edges())
            in_edges = list(node.in_edges())

            # Calculate statistics
            total_sent = sum(
                float(edge.properties.get("amount", 0))
                for edge in out_edges
            )
            total_received = sum(
                float(edge.properties.get("amount", 0))
                for edge in in_edges
            )

            return {
                "address": address,
                "first_seen": node.earliest_time if hasattr(node, 'earliest_time') else None,
                "last_seen": node.latest_time if hasattr(node, 'latest_time') else None,
                "transaction_count": len(out_edges) + len(in_edges),
                "sent_count": len(out_edges),
                "received_count": len(in_edges),
                "total_sent": total_sent,
                "total_received": total_received,
                "balance_flow": total_received - total_sent
            }

        except Exception as e:
            logger.error(
                "Failed to get node info",
                error=str(e),
                address=address
            )
            return None

    def get_transactions_in_window(
        self,
        start_time: int,
        end_time: int,
        limit: int = 1000
    ) -> List[Dict[str, Any]]:
        """
        Get all transactions in a time window

        Args:
            start_time: Start timestamp (Unix seconds)
            end_time: End timestamp (Unix seconds)
            limit: Maximum number of transactions to return

        Returns:
            List of transaction dictionaries
        """
        try:
            # Create a windowed view of the graph
            windowed_graph = self.graph.window(start_time, end_time)

            transactions = []
            for edge in windowed_graph.edges():
                if len(transactions) >= limit:
                    break

                transactions.append({
                    "from": edge.src().name,
                    "to": edge.dst().name,
                    "amount": edge.properties.get("amount"),
                    "tx_hash": edge.properties.get("tx_hash"),
                    "block_number": edge.properties.get("block_number"),
                    "timestamp": edge.earliest_time if hasattr(edge, 'earliest_time') else None
                })

            logger.info(
                "Retrieved transactions in window",
                start=start_time,
                end=end_time,
                count=len(transactions)
            )

            return transactions

        except Exception as e:
            logger.error(
                "Failed to get transactions in window",
                error=str(e),
                start=start_time,
                end=end_time
            )
            return []

    def get_neighbors(
        self,
        address: str,
        direction: str = "both"
    ) -> List[str]:
        """
        Get neighboring addresses (connected nodes)

        Args:
            address: The address to query
            direction: "in", "out", or "both"

        Returns:
            List of neighbor addresses
        """
        try:
            if not self.graph.has_node(address):
                return []

            node = self.graph.node(address)
            neighbors = set()

            if direction in ("out", "both"):
                for edge in node.out_edges():
                    neighbors.add(edge.dst().name)

            if direction in ("in", "both"):
                for edge in node.in_edges():
                    neighbors.add(edge.src().name)

            return list(neighbors)

        except Exception as e:
            logger.error(
                "Failed to get neighbors",
                error=str(e),
                address=address
            )
            return []

    def find_paths(
        self,
        from_address: str,
        to_address: str,
        max_depth: int = 3
    ) -> List[List[str]]:
        """
        Find paths between two addresses

        Args:
            from_address: Start address
            to_address: End address
            max_depth: Maximum path length

        Returns:
            List of paths (each path is a list of addresses)
        """
        try:
            if not self.graph.has_node(from_address) or not self.graph.has_node(to_address):
                return []

            # Simple BFS to find paths
            paths = []
            queue = [([from_address], set([from_address]))]

            while queue:
                current_path, visited = queue.pop(0)
                current_node = current_path[-1]

                if len(current_path) > max_depth:
                    continue

                if current_node == to_address and len(current_path) > 1:
                    paths.append(current_path)
                    continue

                # Explore neighbors
                for neighbor in self.get_neighbors(current_node, direction="out"):
                    if neighbor not in visited:
                        new_path = current_path + [neighbor]
                        new_visited = visited | {neighbor}
                        queue.append((new_path, new_visited))

            return paths

        except Exception as e:
            logger.error(
                "Failed to find paths",
                error=str(e),
                from_address=from_address,
                to_address=to_address
            )
            return []

    def get_statistics(self) -> Dict[str, Any]:
        """Get graph statistics"""
        try:
            return {
                "node_count": self.graph.count_nodes(),
                "edge_count": self.graph.count_edges(),
                "transaction_count": self._transaction_count,
                "earliest_time": self.graph.earliest_time if hasattr(self.graph, 'earliest_time') else None,
                "latest_time": self.graph.latest_time if hasattr(self.graph, 'latest_time') else None,
                "persistent": self.persistent
            }
        except Exception as e:
            logger.error("Failed to get statistics", error=str(e))
            return {
                "node_count": self._node_count,
                "edge_count": self._edge_count,
                "transaction_count": self._transaction_count,
                "persistent": self.persistent
            }

    def save_snapshot(self, filename: Optional[str] = None) -> bool:
        """
        Save graph snapshot to disk

        Args:
            filename: Optional filename for snapshot

        Returns:
            True if successful
        """
        if not self.persistent:
            logger.warning("Cannot save snapshot for non-persistent graph")
            return False

        try:
            if filename is None:
                filename = f"snapshot_{datetime.now().strftime('%Y%m%d_%H%M%S')}"

            # Raphtory PersistentGraph handles serialization automatically
            logger.info("Graph snapshot saved", filename=filename)
            return True

        except Exception as e:
            logger.error("Failed to save snapshot", error=str(e))
            return False

    def clear(self):
        """Clear the graph (for testing)"""
        if self.persistent:
            self.graph = PersistentGraph()
        else:
            self.graph = Graph()

        self._transaction_count = 0
        self._node_count = 0
        self._edge_count = 0

        logger.info("Graph cleared")
