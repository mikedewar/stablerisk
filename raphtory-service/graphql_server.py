"""
GraphQL Server for Raphtory UI

This module starts the Raphtory GraphQL server with built-in UI
for interactive graph exploration and visualization.
"""

import os
import structlog
from raphtory import graphql

logger = structlog.get_logger()


def start_graphql_server(graph_manager):
    """
    Start the Raphtory GraphQL server with UI

    Args:
        graph_manager: The GraphManager instance containing the graph
    """
    port = int(os.getenv("GRAPHQL_PORT", "1736"))
    work_dir = os.getenv("GRAPHQL_WORK_DIR", "/tmp/raphtory_graphql")

    logger.info(
        "Starting Raphtory GraphQL server with UI",
        port=port,
        work_dir=work_dir
    )

    try:
        # Create working directory if it doesn't exist
        os.makedirs(work_dir, exist_ok=True)

        # Create GraphQL server with working directory
        # The UI will be available at http://localhost:{port}/
        # The GraphQL playground will be at http://localhost:{port}/playground
        server = graphql.GraphServer(work_dir)

        # Start the server (non-blocking) and get client
        handle = server.start(port=port)
        client = handle.get_client()

        # Send the graph to the server
        # This makes it available in the UI
        if graph_manager and graph_manager.graph:
            client.send_graph(
                name="usdt_transactions",
                graph=graph_manager.graph,
                overwrite=True
            )
            logger.info(
                "Graph sent to GraphQL server",
                name="usdt_transactions"
            )
        else:
            logger.warning("Graph manager or graph not available, UI will be empty")

        logger.info(
            "GraphQL server started successfully",
            port=port,
            ui_url=f"http://localhost:{port}/",
            playground_url=f"http://localhost:{port}/playground"
        )

        # Keep the thread alive
        import time
        while True:
            time.sleep(60)
            # Periodically update the graph in case new transactions arrived
            if graph_manager and graph_manager.graph:
                client.send_graph(
                    name="usdt_transactions",
                    graph=graph_manager.graph,
                    overwrite=True
                )

    except Exception as e:
        logger.error(
            "Failed to start GraphQL server",
            error=str(e),
            exc_info=True
        )
        raise
