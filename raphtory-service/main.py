"""
Main entry point for Raphtory service

Starts both the FastAPI REST server and GraphQL server with UI
"""

import os
import sys
import structlog
import uvicorn
import threading
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
    # Import here to avoid circular dependencies
    from api.server import graph_manager
    from graphql_server import start_graphql_server

    try:
        # Give FastAPI a moment to start and initialize graph_manager
        import time
        time.sleep(2)

        if graph_manager is None:
            logger.warning("Graph manager not initialized yet, starting GraphQL server without graph")

        start_graphql_server(graph_manager)
    except Exception as e:
        logger.error("GraphQL server error", error=str(e), exc_info=True)


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
