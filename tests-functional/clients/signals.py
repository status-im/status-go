import json
import logging
import time

import websocket


class SignalClient:

    def __init__(self, ws_url, await_signals):
        self.url = f"{ws_url}/signals"

        self.await_signals = await_signals
        self.received_signals = {
            signal: [] for signal in self.await_signals
        }

    def on_message(self, ws, signal):
        signal = json.loads(signal)
        if signal.get("type") in self.await_signals:
            self.received_signals[signal["type"]].append(signal)

    def wait_for_signal(self, signal_type, timeout=20):
        start_time = time.time()
        while not self.received_signals.get(signal_type):
            if time.time() - start_time >= timeout:
                raise TimeoutError(
                    f"Signal {signal_type} is not received in {timeout} seconds")
            time.sleep(0.2)
        logging.debug(f"Signal {signal_type} is received in {round(time.time() - start_time)} seconds")
        return self.received_signals[signal_type][0]

    def _on_error(self, ws, error):
        logging.error(f"Error: {error}")

    def _on_close(self, ws, close_status_code, close_msg):
        logging.info(f"Connection closed: {close_status_code}, {close_msg}")

    def _on_open(self, ws):
        logging.info("Connection opened")

    def _connect(self):
        ws = websocket.WebSocketApp(self.url,
                                    on_message=self.on_message,
                                    on_error=self._on_error,
                                    on_close=self._on_close)
        ws.on_open = self._on_open
        ws.run_forever()
