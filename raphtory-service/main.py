"""
Main entry point for Raphtory service
"""

import os
import sys
import structlog
import uvicorn
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


def main():
    """Main function to start the server"""
    # Get configuration from environment
    host = os.getenv("HOST", "0.0.0.0")
    port = int(os.getenv("PORT", "8000"))
    log_level = os.getenv("LOG_LEVEL", "info").lower()
    workers = int(os.getenv("WORKERS", "1"))
    reload = os.getenv("RELOAD", "false").lower() == "true"

    logger.info(
        "Starting Raphtory service",
        host=host,
        port=port,
        log_level=log_level,
        workers=workers,
        reload=reload
    )

    # Run server
    try:
        uvicorn.run(
            "api.server:app",
            host=host,
            port=port,
            log_level=log_level,
            workers=workers,
            reload=reload
        )
    except KeyboardInterrupt:
        logger.info("Server stopped by user")
        sys.exit(0)
    except Exception as e:
        logger.error("Server error", error=str(e), exc_info=True)
        sys.exit(1)


if __name__ == "__main__":
    main()
