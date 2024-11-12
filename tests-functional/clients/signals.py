import json
import logging
import time

import websocket


class SignalClient:

    def __init__(self, ws_url, await_signals):
        self.url = f"{ws_url}/signals"

        self.await_signals = await_signals
        self.received_signals = {
            # For each signal type, store:
            # - list of received signals
            # - expected received event delta count (resets to 1 after each wait_for_event call)
            # - expected received event count
            # - a function that takes the received signal as an argument and returns True if the signal is accepted (counted) or discarded
            signal: {
                "received": [],
                "delta_count": 1,
                "expected_count": 1,
                "accept_fn": None
            } for signal in self.await_signals
        }

    def on_message(self, ws, signal):
        signal = json.loads(signal)
        signal_type = signal.get("type")
        if signal_type in self.await_signals:
            accept_fn = self.received_signals[signal_type]["accept_fn"]
            if not accept_fn or accept_fn(signal):
                self.received_signals[signal_type]["received"].append(signal)

    def check_signal_type(self, signal_type):
        if signal_type not in self.await_signals:
            raise ValueError(f"Signal type {signal_type} is not in the list of awaited signals")

    # Used to set up how many instances of a signal to wait for, before triggering the actions
    # that cause them to be emitted.
    def prepare_wait_for_signal(self, signal_type, delta_count, accept_fn=None):
        self.check_signal_type(signal_type)

        if delta_count < 1:
            raise ValueError("delta_count must be greater than 0")
        self.received_signals[signal_type]["delta_count"] = delta_count
        self.received_signals[signal_type]["expected_count"] = len(self.received_signals[signal_type]["received"]) + delta_count
        self.received_signals[signal_type]["accept_fn"] = accept_fn

    def wait_for_signal(self, signal_type, timeout=20):
        self.check_signal_type(signal_type)

        start_time = time.time()
        received_signals = self.received_signals.get(signal_type)
        while (not received_signals) or len(received_signals["received"]) < received_signals["expected_count"]:
            if time.time() - start_time >= timeout:
                raise TimeoutError(
                    f"Signal {signal_type} is not received in {timeout} seconds")
            time.sleep(0.2)
        logging.debug(f"Signal {signal_type} is received in {round(time.time() - start_time)} seconds")
        delta_count = received_signals["delta_count"]
        self.prepare_wait_for_signal(signal_type, 1)
        if delta_count == 1:
            return self.received_signals[signal_type]["received"][-1]
        return self.received_signals[signal_type]["received"][-delta_count:]

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
