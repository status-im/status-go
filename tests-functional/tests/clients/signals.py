
import websocket
import time
import json
import logging


class SignalClient:

    def __init__(self, ws_url, await_signals):
        self.ws_url = ws_url

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
            time_passed = time.time() - start_time
            if time_passed >= timeout:
                raise TimeoutError(
                    f"Signal {signal_type} is not  received in {timeout} seconds")
            time.sleep(0.2)
        return self.received_signals[signal_type][0]

    def _on_error(self, ws, error):
        logging.info(f"Error: {error}")

    def _on_close(self, ws, close_status_code, close_msg):
        logging.info(f"Connection closed: {close_status_code}, {close_msg}")

    def _on_open(self, ws):
        logging.info("Connection opened")

    def _connect(self):
        self.url = f"{self.ws_url}/signals"
        ws = websocket.WebSocketApp(self.url,
                                    on_message=self.on_message,
                                    on_error=self._on_error,
                                    on_close=self._on_close)
        ws.on_open = self._on_open
        ws.run_forever()
