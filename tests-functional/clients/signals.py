import json
import logging
import time
from src.libs.common import write_signal_to_file

import websocket

logger = logging.getLogger(__name__)

class SignalClient:

    def __init__(self, ws_url, await_signals):
        self.url = f"{ws_url}/signals"
        self.await_signals = await_signals
        self.received_signals = {signal: [] for signal in self.await_signals}

    def on_message(self, ws, signal):
        signal_data = json.loads(signal)
        signal_type = signal_data.get("type")

        write_signal_to_file(signal_data)

        if signal_type in self.await_signals:
            self.received_signals[signal_type].append(signal_data)

    def wait_for_signal(self, signal_type, expected_event=None, timeout=20):
        start_time = time.time()
        while time.time() - start_time < timeout:
            if self.received_signals.get(signal_type):
                received_signal = self.received_signals[signal_type][0]
                if expected_event:
                    event = received_signal.get("event", {})
                    if all(event.get(k) == v for k, v in expected_event.items()):
                        logger.info(f"Signal {signal_type} with event {expected_event} received and matched.")
                        return received_signal
                    else:
                        logger.debug(
                            f"Signal {signal_type} received but event did not match expected event: {expected_event}. Received event: {event}")
                else:
                    logger.info(f"Signal {signal_type} received without specific event validation.")
                    return received_signal
            time.sleep(0.2)

        raise TimeoutError(f"Signal {signal_type} with event {expected_event} not received in {timeout} seconds")

    def _on_error(self, ws, error):
        logger.error(f"WebSocket error: {error}")

    def _on_close(self, ws, close_status_code, close_msg):
        logger.info(f"WebSocket connection closed: {close_status_code}, {close_msg}")

    def _on_open(self, ws):
        logger.info("WebSocket connection opened")

    def _connect(self):
        ws = websocket.WebSocketApp(
            self.url,
            on_message=self.on_message,
            on_error=self._on_error,
            on_close=self._on_close
        )
        ws.on_open = self._on_open
        ws.run_forever()
