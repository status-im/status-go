import logging
import threading

import pytest
from tenacity import retry, stop_after_delay, wait_fixed

from clients.status_backend import StatusBackend


@pytest.mark.rpc
class TestDiscovery:
    def test_discovery(self, backend_new_profile):
        # Desired topology: 3 full nodes, 2 light nodes
        full_count = 3
        light_count = 2
        total_nodes = full_count + light_count

        nodes: dict[str, StatusBackend] = {}
        full_ids: set[str] = set()

        known_nodes = {
            # Boot nodes available in the fleet
            "16Uiu2HAm3vFYHkGRURyJ6F7bwDyzMLtPEuCg4DU89T7km2u8Fjyb": "boot-1",
            "16Uiu2HAmCDqxtfF1DwBqs7UJ4TgSnjoh6j1RtE1hhQxLLao84jLi": "store",
        }

        def create_node(node_index: int, waku_light_client: bool):
            backend = backend_new_profile(
                f"node_{node_index}",
                # We mix full and light clients in a single test.
                # Light clients (edge nodes) will not discover each,
                # but they must discover full clients.
                waku_light_client=waku_light_client,
                # Force the container to only have the docker compose network.
                # Otherwise, advertised ENRs may contain the wrong IP address.
                bridge_network=False,
            )
            peer_id = backend.wakuext_service.peer_id()
            known_nodes[peer_id] = f"backend_{node_index}"
            if not waku_light_client:
                full_ids.add(peer_id)
            nodes[peer_id] = backend
            info = f"Peer ID: {peer_id}"
            info += f", URL: {backend.url}"
            if backend.container:
                info += f", Container: {backend.container.short_id()}"
            info += f", Type: {'full' if not waku_light_client else 'light'}"
            logging.info(f"Backend {node_index} ready. {info}")

        # Start threads to create nodes
        logging.info("Starting threads to create full and light clients...")
        threads = []

        # First start full nodes
        for i in range(full_count):
            t = threading.Thread(target=create_node, args=(i, False))
            t.daemon = True
            t.start()
            threads.append(t)

        # Then start light nodes
        for i in range(full_count, total_nodes):
            t = threading.Thread(target=create_node, args=(i, True))
            t.daemon = True
            t.start()
            threads.append(t)

        for t in threads:
            t.join()

        assert len(nodes.keys()) == total_nodes, "Not all nodes created"

        # Build sets for expectations
        all_node_ids = set(nodes.keys())
        boot_ids = {k for k, v in known_nodes.items() if v in ("boot-1", "store")}

        # Pre-build expected peers per node (static expectations)
        expected_by_node: dict[str, set[str]] = {}
        for pid in nodes.keys():
            if pid in full_ids:
                # Full clients should discover all other clients and boot nodes
                expected = (all_node_ids | boot_ids) - {pid}
            else:
                # Light clients should discover only full clients and boot nodes
                expected = boot_ids | full_ids
            expected_by_node[pid] = expected

        # Check using tenacity.retry, logs peers on each iteration
        @retry(stop=stop_after_delay(120), wait=wait_fixed(1), reraise=True)
        def assert_discovery():
            all_ok = True

            for peer_id, node in nodes.items():
                # Fetch peers once per node per attempt
                peers = set(node.wakuext_service.peers())

                # Log peer names
                peer_names = [known_nodes.get(peer, peer[-5:]) for peer in peers]
                logging.info(f"Checking node {known_nodes[peer_id]} peers: {peer_names}")

                if peers >= expected_by_node[peer_id]:
                    logging.info(f"Node {known_nodes[peer_id]} discovered peers as expected")
                    nodes.pop(peer_id)
                    continue

                all_ok = False

            assert all_ok, "Not all nodes discovered as expected"

        assert_discovery()
