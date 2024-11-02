import json
import logging
import time
from src.libs.common import write_signal_to_file

import websocket

from src.libs.custom_logger import get_custom_logger

logger = get_custom_logger(__name__)


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

    def wait_for_signal(self, signal_type, timeout=20):
        start_time = time.time()
        while not self.received_signals.get(signal_type):
            if time.time() - start_time >= timeout:
                raise TimeoutError(
                    f"Signal {signal_type} is not received in {timeout} seconds")
            time.sleep(0.2)
        logging.debug(f"Signal {signal_type} is received in {round(time.time() - start_time)} seconds")
        return self.received_signals[signal_type][0]

    def wait_for_complete_signal(self, signal_type, timeout=5):
        start_time = time.time()
        events = []

        while time.time() - start_time < timeout:
            if self.received_signals.get(signal_type):
                events.extend(self.received_signals[signal_type])
                self.received_signals[signal_type] = []
            time.sleep(0.2)

        if events:
            logging.debug(
                f"Collected {len(events)} events of type {signal_type} within {timeout} seconds")
            return events
        raise TimeoutError(f"No signals of type {signal_type} received in {timeout} seconds")

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
