import json
import logging
import time

import websocket
import os
from pathlib import Path
from constants import SIGNALS_DIR, LOG_SIGNALS_TO_FILE
from datetime import datetime
from enum import Enum

class SignalType(Enum):
    MESSAGES_NEW = "messages.new"
    MESSAGE_DELIVERED = "message.delivered"
    NODE_READY = "node.ready"
    NODE_STARTED = "node.started"
    NODE_LOGIN = "node.login"
    MEDIASERVER_STARTED = "mediaserver.started"
    WALLET_SUGGESTED_ROUTES = "wallet.suggested.routes"
    WALLET_ROUTER_SIGN_TRANSACTIONS = "wallet.router.sign-transactions"
    WALLET_ROUTER_SENDING_TRANSACTIONS_STARTED = "wallet.router.sending-transactions-started"
    WALLET_TRANSACTION_STATUS_CHANGED = "wallet.transaction.status-changed"
    WALLET_ROUTER_TRANSACTIONS_SENT = "wallet.router.transactions-sent"

class SignalClient:
    def __init__(self, ws_url, await_signals):
        self.url = f"{ws_url}/signals"

        self.await_signals = await_signals
        self.received_signals = {
            signal: [] for signal in self.await_signals
        }
        if LOG_SIGNALS_TO_FILE:
            self.signal_file_path = os.path.join(SIGNALS_DIR, f"signal_{ws_url.split(':')[-1]}_{datetime.now().strftime('%H%M%S')}.log")
            Path(SIGNALS_DIR).mkdir(parents=True, exist_ok=True)

    def on_message(self, ws, signal):
        signal_data = json.loads(signal)
        if LOG_SIGNALS_TO_FILE:
            self.write_signal_to_file(signal_data)

        signal_type = signal_data.get("type")
        if signal_type in self.await_signals:
            self.received_signals[signal_type].append(signal_data)

    def wait_for_signal(self, signal_type, timeout=20, event_pattern=None):
        start_time = time.time()
        while True:
            if not self.received_signals.get(signal_type):
                if time.time() - start_time >= timeout:
                    raise TimeoutError(
                        f"Signal {signal_type} containing {event_pattern} is not received in {timeout} seconds"
                    )
                time.sleep(0.2)
                continue
            for event in self.received_signals[signal_type]:
                if event_pattern is None or event_pattern in str(event):
                    logging.info(
                        f"Signal {signal_type} containing {event_pattern} is received in {round(time.time() - start_time)} seconds"
                    )
                    return event
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