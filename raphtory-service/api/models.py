"""
API models for Raphtory service
"""

from pydantic import BaseModel, Field
from typing import List, Optional, Dict, Any
from datetime import datetime


class TransactionInput(BaseModel):
    """Input model for adding a transaction"""
    tx_hash: str = Field(..., description="Transaction hash")
    from_address: str = Field(..., alias="from", description="Sender address")
    to_address: str = Field(..., alias="to", description="Receiver address")
    amount: str = Field(..., description="Transaction amount")
    timestamp: int = Field(..., description="Unix timestamp in seconds")
    block_number: int = Field(..., description="Block number")
    contract: str = Field(..., description="Contract address")

    class Config:
        populate_by_name = True


class TransactionResponse(BaseModel):
    """Response model for a transaction"""
    from_address: str = Field(..., alias="from")
    to_address: str = Field(..., alias="to")
    amount: str
    tx_hash: str
    block_number: int
    timestamp: Optional[int] = None

    class Config:
        populate_by_name = True


class NodeInfo(BaseModel):
    """Information about a graph node (address)"""
    address: str
    first_seen: Optional[int] = None
    last_seen: Optional[int] = None
    transaction_count: int = 0
    sent_count: int = 0
    received_count: int = 0
    total_sent: float = 0.0
    total_received: float = 0.0
    balance_flow: float = 0.0


class NeighborsResponse(BaseModel):
    """Response for neighbor query"""
    address: str
    neighbors: List[str]
    count: int


class PathsResponse(BaseModel):
    """Response for path finding query"""
    from_address: str = Field(..., alias="from")
    to_address: str = Field(..., alias="to")
    paths: List[List[str]]
    count: int

    class Config:
        populate_by_name = True


class GraphStatistics(BaseModel):
    """Graph statistics"""
    node_count: int
    edge_count: int
    transaction_count: int
    earliest_time: Optional[int] = None
    latest_time: Optional[int] = None
    persistent: bool


class HealthResponse(BaseModel):
    """Health check response"""
    status: str
    timestamp: str
    graph_stats: Optional[GraphStatistics] = None


class ErrorResponse(BaseModel):
    """Error response"""
    error: str
    detail: Optional[str] = None


class SuccessResponse(BaseModel):
    """Generic success response"""
    success: bool
    message: str
