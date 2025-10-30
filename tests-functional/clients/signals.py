import json
import logging
import os
import threading
import time
from datetime import datetime
from enum import Enum
from pathlib import Path

import websocket

from resources.constants import SIGNALS_DIR, LOG_SIGNALS_TO_FILE


# Only signals defined in SignalType are processed by SignalClient
class SignalType(Enum):
    MESSAGES_NEW = "messages.new"
    MESSAGE_DELIVERED = "message.delivered"
    NODE_READY = "node.ready"
    NODE_STARTED = "node.started"
    NODE_LOGIN = "node.login"
    NODE_STOPPED = "node.stopped"
    MEDIASERVER_STARTED = "mediaserver.started"
    WALLET = "wallet"
    WALLET_SUGGESTED_ROUTES = "wallet.suggested.routes"
    WALLET_ROUTER_SIGN_TRANSACTIONS = "wallet.router.sign-transactions"
    WALLET_ROUTER_SENDING_TRANSACTIONS_STARTED = "wallet.router.sending-transactions-started"
    WALLET_ROUTER_TRANSACTIONS_SENT = "wallet.router.transactions-sent"
    LOCAL_PAIRING = "localPairing"
    DB_REENCRYPTION_STARTED = "db.reEncryption.started"
    DB_REENCRYPTION_FINISHED = "db.reEncryption.finished"
    CONNECTOR_SEND_REQUEST_ACCOUNTS = "connector.sendRequestAccounts"
    CONNECTOR_SEND_TRANSACTION = "connector.sendTransaction"
    CONNECTOR_SIGN = "connector.sign"
    CONNECTOR_DAPP_PERMISSION_GRANTED = "connector.dAppPermissionGranted"
    CONNECTOR_DAPP_PERMISSION_REVOKED = "connector.dAppPermissionRevoked"
    CONNECTOR_DAPP_CHAIN_ID_SWITCHED = "connector.dAppChainIdSwitched"


class WalletEventType(Enum):
    WALLET_ACTIVITY_FILTERING_DONE = "wallet-activity-filtering-done"
    WALLET_ACTIVITY_FILTERING_ENTRIES_UPDATED = "wallet-activity-filtering-entries-updated"
    WALLET_ACTIVITY_SESSION_UPDATED = "wallet-activity-session-updated"
    TRANSACTIONS_PENDING_TRANSACTION_UPDATE = "pending-transaction-update"
    TRANSACTIONS_PENDING_TRANSACTION_STATUS_CHANGED = "pending-transaction-status-changed"
    WALLET_TICK_RELOAD = "wallet-tick-reload"


class LocalPairingEventType(Enum):
    # Both Sender and Receiver
    EVENT_PEER_DISCOVERED = "peer-discovered"
    EVENT_CONNECTION_ERROR = "connection-error"
    EVENT_CONNECTION_SUCCESS = "connection-success"
    EVENT_TRANSFER_ERROR = "transfer-error"
    EVENT_TRANSFER_SUCCESS = "transfer-success"
    EVENT_RECEIVED_INSTALLATION = "received-installation"
    # Only Receiver side
    EVENT_RECEIVED_ACCOUNT = "received-account"
    EVENT_PROCESS_SUCCESS = "process-success"
    EVENT_PROCESS_ERROR = "process-error"
    EVENT_RECEIVED_KEYSTORE_FILES = "received-keystore-files"


class LocalPairingEventAction(Enum):
    ACTION_CONNECT = 1
    ACTION_PAIRING_ACCOUNT = 2
    ACTION_SYNC_DEVICE = 3
    ACTION_PAIRING_INSTALLATION = 4
    ACTION_PEER_DISCOVERY = 5
    ACTION_KEYSTORE_FILES_TRANSFER = 6


class SignalClient:
    def __init__(self, ws_url):
        self.url = f"{ws_url}/signals"

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
                "accept_fn": None,
            }
            for signal in SignalType
        }
        if LOG_SIGNALS_TO_FILE:
            self.signal_file_path = os.path.join(
                SIGNALS_DIR,
                f"signal_{ws_url.split(':')[-1]}_{datetime.now().strftime('%H%M%S')}.log",
            )
            Path(SIGNALS_DIR).mkdir(parents=True, exist_ok=True)

    def on_message(self, ws, signal):
        signal_data = json.loads(signal)
        if LOG_SIGNALS_TO_FILE:
            self.write_signal_to_file(signal_data)

        signal_type = signal_data.get("type")
        try:
            signal_type = self._convert_signal_type(signal_type)
        except ValueError:
            # Ignore unregistered signal types
            return

        if signal_type not in self.received_signals:
            # This should never happen, as we register all signal types from SignalType enum
            raise ValueError(f"Signal type {signal_type} is not registered")

        accept_fn = self.received_signals[signal_type]["accept_fn"]
        if not accept_fn or accept_fn(signal_data):
            self.received_signals[signal_type]["received"].append(signal_data)

    # TODO: This is a temporary workaround until all tests are migrated to use SignalType enum
    @staticmethod
    def _convert_signal_type(signal_type: SignalType | str) -> SignalType:
        if isinstance(signal_type, SignalType):
            return signal_type
        if isinstance(signal_type, str):
            return SignalType(signal_type)

    # Used to set up how many instances of a signal to wait for, before triggering the actions
    # that cause them to be emitted.
    def prepare_wait_for_signal(self, signal_type: SignalType, delta_count: int, accept_fn=None):
        signal_type = self._convert_signal_type(signal_type)

        if delta_count < 1:
            raise ValueError("delta_count must be greater than 0")
        self.received_signals[signal_type]["delta_count"] = delta_count
        self.received_signals[signal_type]["expected_count"] = len(self.received_signals[signal_type]["received"]) + delta_count
        self.received_signals[signal_type]["accept_fn"] = accept_fn

    def wait_for_signal(self, signal_type: SignalType | str, timeout: int | None = 20):
        signal_type = self._convert_signal_type(signal_type)

        start_time = time.time()
        received_signals = self.received_signals.get(signal_type)
        while (not received_signals) or len(received_signals["received"]) < received_signals["expected_count"]:
            if timeout is not None and time.time() - start_time >= timeout:
                raise TimeoutError(f"Signal {signal_type} is not received in {timeout} seconds")
            time.sleep(0.2)
        logging.debug(f"Signal {signal_type} is received in {round(time.time() - start_time)} seconds")
        delta_count = received_signals["delta_count"]
        self.prepare_wait_for_signal(signal_type, 1)
        if delta_count == 1:
            return self.received_signals[signal_type]["received"][-1]
        return self.received_signals[signal_type]["received"][-delta_count:]

    def wait_for_signal_predicate(self, signal_type: SignalType | str, predicate=lambda signal: True, timeout=20):
        signal_type = self._convert_signal_type(signal_type)
        start_time = time.time()
        while True:
            elapsed_time = time.time() - start_time
            if elapsed_time >= timeout:
                break
            remaining_time = int(timeout - elapsed_time)
            signal = self.wait_for_signal(signal_type, remaining_time)
            try:
                if predicate(signal):
                    return signal
            except Exception as ex:
                logging.warning(f"Could not filter signal by predicate because of error: {str(ex)}")
                continue
        raise TimeoutError(f"Signal {signal_type} satisfying the predicate is not received in {timeout} seconds")

    def wait_for_logout(self):
        signal = self.wait_for_signal(SignalType.NODE_STOPPED)
        return signal

    def find_signal_containing_pattern(self, signal_type: SignalType | str, event_pattern, timeout=20):
        signal_type = self._convert_signal_type(signal_type)

        start_time = time.time()
        while True:
            if time.time() - start_time >= timeout:
                raise TimeoutError(f"Signal {signal_type} containing {event_pattern} is not received in {timeout} seconds")
            if not self.received_signals.get(signal_type):
                time.sleep(0.2)
                continue
            for event in self.received_signals[signal_type]["received"]:
                if event_pattern in json.dumps(event):
                    logging.debug(f"Signal {signal_type} containing {event_pattern} is received in {round(time.time() - start_time)} seconds")
                    return event
            time.sleep(0.2)

    def get_all_events(self, signal_type: SignalType | str):
        signal_type = self._convert_signal_type(signal_type)
        signals = self.received_signals.get(signal_type, {}).get("received", [])
        return [signal.get("event") for signal in signals]

    def _on_error(self, ws, error):
        logging.error(f"SignalClient [{self.url}]: websocket error: {error}")

    def _on_close(self, ws, close_status_code, close_msg):
        logging.debug(f"SignalClient [{self.url}]: websocket connection closed: {close_status_code}, {close_msg}")

    def _on_open(self, ws):
        logging.debug(f"SignalClient [{self.url}]: websocket connection opened")

    def _connect(self):
        self.wsapp = websocket.WebSocketApp(
            url=self.url,
            on_message=self.on_message,
            on_error=self._on_error,
            on_open=self._on_open,
            on_close=self._on_close,
        )
        self.wsapp.run_forever()

    def connect(self):
        websocket_thread = threading.Thread(target=self._connect)
        websocket_thread.daemon = True
        websocket_thread.start()

    def disconnect(self):
        if hasattr(self, "wsapp") and self.wsapp is not None:
            self.wsapp.close()

    def write_signal_to_file(self, signal_data):
        with open(self.signal_file_path, "a+") as file:
            json.dump(signal_data, file)
            file.write("\n")
