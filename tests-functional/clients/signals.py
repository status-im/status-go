import json
import logging
import time

import websocket
import os
from pathlib import Path
from constants import SIGNALS_DIR
from datetime import datetime


class SignalClient:
    def __init__(self, ws_url, await_signals):
        self.url = f"{ws_url}/signals"

        self.await_signals = await_signals
        self.received_signals = {
            signal: [] for signal in self.await_signals
        }
        self.signal_file_path = os.path.join(SIGNALS_DIR, f"sig_log_{ws_url.split(":")[-1]}_{datetime.now().strftime("%H%M%S")}.json")
        Path(SIGNALS_DIR).mkdir(parents=True, exist_ok=True)

    def on_message(self, ws, signal):
        signal_data = json.loads(signal)
        self.write_signal_to_file(signal_data)

        signal_type = signal_data.get("type")
        if signal_type in self.await_signals:
            self.received_signals[signal_type].append(signal_data)

    def wait_for_signal(self, signal_type, timeout=20, event_contains=None):
        start_time = time.time()
        while True:
            if self.received_signals.get(signal_type):
                for event in self.received_signals[signal_type]:
                    if event_contains is None or event_contains in str(event):
                        logging.info(
                            f"Signal {signal_type} containing {event_contains} is received in {round(time.time() - start_time)} seconds")
                        return event
            if time.time() - start_time >= timeout:
                raise TimeoutError(
                    f"Signal {signal_type} containing {event_contains} is not received in {timeout} seconds")
            time.sleep(0.2)

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

    def write_signal_to_file(self, signal_data):
        with open(self.signal_file_path, "a+") as file:
            json.dump(signal_data, file)
            file.write("\n")