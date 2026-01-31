"""
Main entry point for Raphtory service

Starts both the FastAPI REST server and GraphQL server with UI
"""

import os
import sys
import structlog
import uvicorn
import threading
import time
from dotenv import load_dotenv

# Load environment variables
load_dotenv()

# Configure logging
structlog.configure(
    processors=[
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.add_log_level,
        structlog.processors.JSONRenderer()
    ]
)

logger = structlog.get_logger()


def run_fastapi_server():
    """Run the FastAPI REST server"""
    host = os.getenv("HOST", "0.0.0.0")
    port = int(os.getenv("PORT", "8000"))
    log_level = os.getenv("LOG_LEVEL", "info").lower()
    workers = int(os.getenv("WORKERS", "1"))
    reload = os.getenv("RELOAD", "false").lower() == "true"

    logger.info(
        "Starting FastAPI REST server",
        host=host,
        port=port,
        log_level=log_level,
        workers=workers,
        reload=reload
    )

    try:
        uvicorn.run(
            "api.server:app",
            host=host,
            port=port,
            log_level=log_level,
            workers=workers,
            reload=reload
        )
    except Exception as e:
        logger.error("FastAPI server error", error=str(e), exc_info=True)
        sys.exit(1)


def run_graphql_server():
    """Run the GraphQL server with UI"""
    from raphtory import graphql

    graphql_port = int(os.getenv("GRAPHQL_PORT", "1736"))
    work_dir = os.getenv("GRAPHQL_WORK_DIR", "/tmp/raphtory_graphql")

    try:
        # Wait for FastAPI to start and initialize graph_manager
        logger.info("Waiting for FastAPI to initialize...")
        time.sleep(5)

        # Import after waiting to ensure FastAPI has started
        from api import server as api_server

        # Create working directory
        os.makedirs(work_dir, exist_ok=True)

        logger.info(
            "Starting Raphtory GraphQL server with UI",
            port=graphql_port,
            work_dir=work_dir
        )

        # Create and start GraphQL server
        graphql_server = graphql.GraphServer(work_dir)
        handle = graphql_server.start(port=graphql_port)
        client = handle.get_client()

        logger.info(
            "GraphQL server started successfully",
            port=graphql_port,
            ui_url=f"http://localhost:{graphql_port}/",
            playground_url=f"http://localhost:{graphql_port}/playground"
        )

        # Periodic graph update loop
        update_interval = 10  # Update every 10 seconds
        while True:
            time.sleep(update_interval)

            # Get the current graph from the FastAPI server's graph_manager
            if api_server.graph_manager is not None and api_server.graph_manager.graph is not None:
                try:
                    # Send updated graph to GraphQL UI
                    client.send_graph(
                        path="usdt_transactions",
                        graph=api_server.graph_manager.graph,
                        overwrite=True
                    )

                    stats = api_server.graph_manager.get_statistics()
                    logger.debug(
                        "Graph updated in UI",
                        transactions=stats.get('transaction_count', 0),
                        nodes=stats.get('node_count', 0),
                        edges=stats.get('edge_count', 0)
                    )
                except Exception as e:
                    logger.error("Failed to update graph in UI", error=str(e))
            else:
                logger.debug("Graph manager not ready yet, skipping update")

    except Exception as e:
        logger.error(
            "GraphQL server error",
            error=str(e),
            exc_info=True
        )


def main():
    """Main function to start both servers"""
    logger.info("Starting Raphtory service with REST API and GraphQL UI")

    # Start GraphQL server in a separate thread
    graphql_thread = threading.Thread(target=run_graphql_server, daemon=True)
    graphql_thread.start()

    # Run FastAPI server in main thread
    try:
        run_fastapi_server()
    except KeyboardInterrupt:
        logger.info("Server stopped by user")
        sys.exit(0)


if __name__ == "__main__":
    main()
