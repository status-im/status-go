import logging
import threading
import time

import pytest

from clients.status_backend import StatusBackend

DISCOVERY_TIMEOUT_SEC = 30


def _all_nodes_discovered(nodes: dict[str, StatusBackend], known_peers: dict[str, str]) -> bool:
    """Check if all nodes discovered each other"""
    for peer_id, node in nodes.items():
        if node is None:
            return False
        peers = node.wakuext_service.peers()

        # Use shorter names for logging
        peer_names = [known_peers.get(peer, peer[-5:]) for peer in peers]
        logging.info(f"Node {known_peers[peer_id]} peers: {peer_names}")

        for known_node in known_peers.keys():
            if known_node == peer_id:
                continue
            if known_node not in peers:
                return False
    return True


@pytest.mark.rpc
class TestDiscovery:

    # @pytest.mark.parametrize("waku_light_client", [False, True], indirect=True,
    #                          ids=["wakuV2LightClient_False", "wakuV2LightClient_True"])
    def test_discovery(self, backend_new_profile):
        nodes_count = 2
        nodes: dict[str, StatusBackend] = {}

        known_nodes = {
            "16Uiu2HAm3vFYHkGRURyJ6F7bwDyzMLtPEuCg4DU89T7km2u8Fjyb": "boot-1",
            "16Uiu2HAmCDqxtfF1DwBqs7UJ4TgSnjoh6j1RtE1hhQxLLao84jLi": "store",
        }

        def create_node(node_index: int):
            """Function to run in each thread - waits for wakuv2.peerstats signal"""
            backend = backend_new_profile(f"node_{node_index}", waku_light_client=True)
            peer_id = backend.wakuext_service.peer_id()
            known_nodes[peer_id] = f"backend_{node_index}"
            nodes[peer_id] = backend
            info = f"Peer ID: {peer_id}"
            info += f", URL: {backend.url}"
            if backend.container:
                info += f", Container: {backend.container.short_id()}"
            logging.info(f"Backend {node_index} ready. {info}")

        # Run threads, each waiting for wakuv2.peerstats signal
        logging.info("Starting threads to wait for wakuv2.peerstats signals...")
        threads = []

        for i in range(nodes_count):
            thread = threading.Thread(target=create_node, args=(i,))
            thread.daemon = True
            thread.start()
            threads.append(thread)

        for thread in threads:
            thread.join()

        assert len(nodes.keys()) == nodes_count, "Not all nodes created"

        # Wait for all nodes to discover each other
        start_time = time.time()
        while time.time() - start_time < DISCOVERY_TIMEOUT_SEC:
            if _all_nodes_discovered(nodes, known_nodes):
                return
            time.sleep(0.5)

        assert False, f"Nodes failed to discover each other within {DISCOVERY_TIMEOUT_SEC} seconds"
