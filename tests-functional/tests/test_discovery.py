import logging
import time
import pytest
import threading

from clients.status_backend import StatusBackend

known_nodes = {
    "16Uiu2HAm3vFYHkGRURyJ6F7bwDyzMLtPEuCg4DU89T7km2u8Fjyb": "boot-1",
    "16Uiu2HAm5TrV8hjUQaqJ8VaHC1EL3EX8RdqP1gTrLxSCecWVN3xc": "boot-2",
    "16Uiu2HAmGqYFBfbKCQmhUagDFVo43dPBoEe4mcp2TThH93AAf55c": "node",
    "16Uiu2HAmCDqxtfF1DwBqs7UJ4TgSnjoh6j1RtE1hhQxLLao84jLi": "store",
}


@pytest.mark.rpc
class TestAppGeneral:

    def test_discovery(self, backend_new_profile):
        nodes_count = 2
        nodes: list[StatusBackend | None] = [None] * nodes_count
        stop_event = threading.Event()

        def create_node(node_index, stop_event):
            """Function to run in each thread - waits for wakuv2.peerstats signal"""
            backend = backend_new_profile(f"node_{node_index}")
            peer_id = backend.wakuext_service.peer_id()
            known_nodes[peer_id] = f"backend_{node_index}"
            nodes[node_index] = backend
            logging.info(
                f"✅ backend {node_index} ready."
                # + f"Container ID: {backend.container.short_id()}."
                # + f"API Port: {backend.container.host_port}."
                + f"Peer ID: {peer_id}"
            )

        # Run threads, each waiting for wakuv2.peerstats signal
        logging.info("⚠️ Starting threads to wait for wakuv2.peerstats signals...")
        threads = []

        for i in range(nodes_count):
            thread = threading.Thread(target=create_node, args=(i, stop_event))
            thread.daemon = True
            thread.start()
            threads.append(thread)

        for thread in threads:
            thread.join()

        logging.info("✅ All threads completed")

        for i, node in enumerate(nodes):
            assert node is not None, f"Node {i} is None"

        try:
            while True:
                logging.info("--- updating peers")
                for i, node in enumerate(nodes):
                    if node is None:
                        continue
                    response = node.wakuext_service.peers(enable_logging=False)
                    peers = response["result"]
                    peers = [known_nodes.get(peer, peer[-5:]) for peer in peers]
                    logging.info(f"node {i}. Total peers count: {len(peers)}, peers: {peers}")
                time.sleep(5)
        except KeyboardInterrupt:
            logging.warning("Keyboard interrupt received")
