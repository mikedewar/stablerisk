"""
FastAPI server for Raphtory temporal graph service
"""

from fastapi import FastAPI, HTTPException, Query, status
from fastapi.middleware.cors import CORSMiddleware
from typing import List, Optional
from datetime import datetime
import structlog

from api.models import (
    TransactionInput,
    TransactionResponse,
    NodeInfo,
    NeighborsResponse,
    PathsResponse,
    GraphStatistics,
    HealthResponse,
    ErrorResponse,
    SuccessResponse
)
from graph.graph_manager import GraphManager

# Initialize logger
structlog.configure(
    processors=[
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.add_log_level,
        structlog.processors.JSONRenderer()
    ]
)
logger = structlog.get_logger()

# Initialize FastAPI app
app = FastAPI(
    title="StableRisk Raphtory Service",
    description="Temporal graph service for USDT transaction analysis",
    version="0.1.0"
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # Configure appropriately for production
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Initialize graph manager
graph_manager: Optional[GraphManager] = None


@app.on_event("startup")
async def startup_event():
    """Initialize graph manager on startup"""
    global graph_manager
    graph_manager = GraphManager(persistent=False)
    logger.info("Raphtory service started")


@app.on_event("shutdown")
async def shutdown_event():
    """Cleanup on shutdown"""
    logger.info("Raphtory service shutting down")


@app.get("/")
async def root():
    """Redirect to API documentation"""
    from fastapi.responses import RedirectResponse
    return RedirectResponse(url="/docs")


@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Health check endpoint"""
    if graph_manager is None:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Graph manager not initialized"
        )

    stats = graph_manager.get_statistics()

    return HealthResponse(
        status="healthy",
        timestamp=datetime.utcnow().isoformat(),
        graph_stats=GraphStatistics(**stats)
    )


@app.post("/graph/transaction", response_model=SuccessResponse, status_code=status.HTTP_201_CREATED)
async def add_transaction(transaction: TransactionInput):
    """
    Add a transaction to the temporal graph

    Args:
        transaction: Transaction data

    Returns:
        Success response
    """
    if graph_manager is None:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Graph manager not initialized"
        )

    logger.info(
        "Adding transaction",
        tx_hash=transaction.tx_hash,
        from_addr=transaction.from_address[:10] + "...",
        to_addr=transaction.to_address[:10] + "..."
    )

    success = graph_manager.add_transaction(
        tx_hash=transaction.tx_hash,
        from_address=transaction.from_address,
        to_address=transaction.to_address,
        amount=transaction.amount,
        timestamp=transaction.timestamp,
        block_number=transaction.block_number,
        contract=transaction.contract
    )

    if not success:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to add transaction to graph"
        )

    return SuccessResponse(
        success=True,
        message="Transaction added successfully"
    )


@app.get("/graph/node/{address}", response_model=NodeInfo)
async def get_node_info(address: str):
    """
    Get information about a node (address)

    Args:
        address: The address to query

    Returns:
        Node information
    """
    if graph_manager is None:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Graph manager not initialized"
        )

    node_info = graph_manager.get_node_info(address)

    if node_info is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Node not found: {address}"
        )

    return NodeInfo(**node_info)


@app.get("/graph/window", response_model=List[TransactionResponse])
async def get_transactions_in_window(
    start: int = Query(..., description="Start timestamp (Unix seconds)"),
    end: int = Query(..., description="End timestamp (Unix seconds)"),
    limit: int = Query(1000, ge=1, le=10000, description="Maximum number of transactions")
):
    """
    Get transactions in a time window

    Args:
        start: Start timestamp
        end: End timestamp
        limit: Maximum number of transactions

    Returns:
        List of transactions
    """
    if graph_manager is None:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Graph manager not initialized"
        )

    if start >= end:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Start time must be before end time"
        )

    transactions = graph_manager.get_transactions_in_window(start, end, limit)

    return [
        TransactionResponse(
            from_address=tx["from"],
            to_address=tx["to"],
            amount=tx["amount"],
            tx_hash=tx["tx_hash"],
            block_number=tx["block_number"],
            timestamp=tx.get("timestamp")
        )
        for tx in transactions
    ]


@app.get("/graph/neighbors/{address}", response_model=NeighborsResponse)
async def get_neighbors(
    address: str,
    direction: str = Query("both", regex="^(in|out|both)$", description="Edge direction")
):
    """
    Get neighboring addresses

    Args:
        address: The address to query
        direction: "in", "out", or "both"

    Returns:
        List of neighbor addresses
    """
    if graph_manager is None:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Graph manager not initialized"
        )

    neighbors = graph_manager.get_neighbors(address, direction)

    return NeighborsResponse(
        address=address,
        neighbors=neighbors,
        count=len(neighbors)
    )


@app.get("/graph/paths", response_model=PathsResponse)
async def find_paths(
    from_address: str = Query(..., alias="from", description="Start address"),
    to_address: str = Query(..., alias="to", description="End address"),
    max_depth: int = Query(3, ge=1, le=10, description="Maximum path length")
):
    """
    Find paths between two addresses

    Args:
        from_address: Start address
        to_address: End address
        max_depth: Maximum path length

    Returns:
        List of paths
    """
    if graph_manager is None:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Graph manager not initialized"
        )

    paths = graph_manager.find_paths(from_address, to_address, max_depth)

    return PathsResponse(
        from_address=from_address,
        to_address=to_address,
        paths=paths,
        count=len(paths)
    )


@app.get("/graph/statistics", response_model=GraphStatistics)
async def get_graph_statistics():
    """Get graph statistics"""
    if graph_manager is None:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Graph manager not initialized"
        )

    stats = graph_manager.get_statistics()
    return GraphStatistics(**stats)


@app.post("/graph/snapshot", response_model=SuccessResponse)
async def save_snapshot(filename: Optional[str] = None):
    """
    Save graph snapshot to disk

    Args:
        filename: Optional filename for snapshot

    Returns:
        Success response
    """
    if graph_manager is None:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Graph manager not initialized"
        )

    success = graph_manager.save_snapshot(filename)

    if not success:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to save snapshot"
        )

    return SuccessResponse(
        success=True,
        message="Snapshot saved successfully"
    )


@app.delete("/graph/clear", response_model=SuccessResponse)
async def clear_graph():
    """Clear the graph (for testing only)"""
    if graph_manager is None:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Graph manager not initialized"
        )

    graph_manager.clear()

    return SuccessResponse(
        success=True,
        message="Graph cleared successfully"
    )


# Error handlers
@app.exception_handler(HTTPException)
async def http_exception_handler(request, exc):
    """Handle HTTP exceptions"""
    logger.error(
        "HTTP exception",
        status_code=exc.status_code,
        detail=exc.detail,
        path=request.url.path
    )
    return ErrorResponse(
        error=exc.detail or "Internal server error",
        detail=str(exc.status_code)
    )


@app.exception_handler(Exception)
async def general_exception_handler(request, exc):
    """Handle general exceptions"""
    logger.error(
        "Unhandled exception",
        error=str(exc),
        path=request.url.path,
        exc_info=True
    )
    return ErrorResponse(
        error="Internal server error",
        detail=str(exc)
    )
