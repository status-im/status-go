import logging
import pytest
import threading

from clients.signals import SignalType


@pytest.mark.rpc
class TestAppGeneral:

    def test_discovery(self, backend_new_profile):
        nodes_count = 2
        peers_count = [0] * nodes_count
        stop_event = threading.Event()

        # logging.info("✅ backends ready")

        def wait_for_peerstats(node_index, stop_event):
            """Function to run in each thread - waits for wakuv2.peerstats signal"""
            node = backend_new_profile(f"node_{node_index}")
            logging.info(f"✅ backend {node_index} ready")

            while not stop_event.is_set():
                try:
                    signal = node.wait_for_signal(SignalType.WAKUV2_PEERSTATS.value, timeout=60)
                    count = len(signal["event"]["peers"])
                    peers_count[node_index] = count
                    logging.info(f"peers count updated: {peers_count}")

                except KeyboardInterrupt:
                    stop_event.set()
                    return
                except Exception as e:
                    logging.info(f"❌Node {node_index} failed to receive signal: {e}")
                    return

        # Run threads, each waiting for wakuv2.peerstats signal
        logging.info("⚠️ Starting threads to wait for wakuv2.peerstats signals...")

        threads = []
        try:
            # Create and start threads
            for i in range(nodes_count):
                thread = threading.Thread(target=wait_for_peerstats, args=(i, stop_event))
                thread.daemon = True
                thread.start()
                threads.append(thread)

            # Wait for all threads to complete
            for thread in threads:
                thread.join()

        except KeyboardInterrupt:
            logging.info("⚠️ Received KeyboardInterrupt, stopping threads...")
            stop_event.set()
            # Wait for threads to finish gracefully
            for thread in threads:
                thread.join(timeout=5)

        logging.info("✅ All threads completed")
