import threading
from src.libs.custom_logger import get_custom_logger
from tests.clients.signals import SignalClient

logger = get_custom_logger(__name__)

class WebSocketManager:
    def __init__(self, port, signals):
        self.ws_url = f"ws://localhost:{port}"
        self.signals = signals
        self.signal_client = SignalClient(self.ws_url, self.signals)

    def start_client(self):
        """Start the WebSocket client in a separate thread."""
        websocket_thread = threading.Thread(target=self.signal_client._connect)
        websocket_thread.daemon = True
        websocket_thread.start()
        logger.info("WebSocket client started and subscribed to signals.")
