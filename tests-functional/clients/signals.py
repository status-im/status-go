import json
import logging
import os
import threading
import time
from datetime import datetime
from enum import Enum
from pathlib import Path
from typing import Callable, Literal

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
    COMMUNITY_MEMBER_REEVALUATION_STATUS = "community.memberReevaluationStatus"


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


class SignalExpectation:
    """Context manager for expecting signals.

    By default waits only for signals that arrive AFTER entering the context (race-safe).
    If you need to match already received signals, use `start="beginning"` or `start=<index>`.
    """

    def __init__(
        self,
        signal_client: "SignalClient",
        signal_type: SignalType,
        *,
        count: int = 1,
        accept_fn: Callable[[dict], bool] | None = None,
        pattern: str | None = None,
        predicate: Callable[[dict], bool] | None = None,
        timeout: float = 20,
        start: Literal["now", "beginning"] | int = "now",
    ):
        self.signal_client = signal_client
        self.signal_type = signal_type
        self.count = count
        self.timeout = timeout
        self.start = start

        self.result: dict | list[dict] | None = None
        self.results: list[dict] | None = None
        self._start_index = 0

        filters_set = sum(1 for v in (accept_fn, pattern, predicate) if v is not None)
        if filters_set > 1:
            raise ValueError("Only one of accept_fn, pattern, predicate can be specified")

        if pattern is not None:
            self.accept_fn: Callable[[dict], bool] | None = lambda s: pattern in json.dumps(s)
        elif predicate is not None:
            self.accept_fn = predicate
        else:
            self.accept_fn = accept_fn

        if self.count < 1:
            raise ValueError("count must be >= 1")

    def __enter__(self):
        with self.signal_client._cond:
            received = self.signal_client._received_by_type[self.signal_type]
            if self.start == "now":
                self._start_index = len(received)
            elif self.start == "beginning":
                self._start_index = 0
            elif isinstance(self.start, int):
                if self.start < 0:
                    raise ValueError("start index must be >= 0")
                self._start_index = self.start
            else:
                raise ValueError(f"Unsupported start mode: {self.start}")
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        if exc_type is not None:
            return False

        deadline = time.time() + float(self.timeout)
        with self.signal_client._cond:
            while True:
                received = self.signal_client._received_by_type[self.signal_type]
                candidates = received[self._start_index :]

                if self.accept_fn is not None:
                    candidates = [s for s in candidates if self.accept_fn(s)]

                if len(candidates) >= self.count:
                    selected = candidates[: self.count]
                    self.results = selected
                    self.result = selected[0] if self.count == 1 else selected
                    return False

                remaining = deadline - time.time()
                if remaining <= 0:
                    raise TimeoutError(
                        f"Expected {self.count} signal(s) of type {self.signal_type}, " f"but got {len(candidates)} in {self.timeout} seconds"
                    )

                self.signal_client._cond.wait(timeout=remaining)


class SignalClient:
    def __init__(self, ws_url):
        self.url = f"{ws_url}/signals"

        self._cond = threading.Condition()
        self._seq = 0
        self._received_by_type: dict[SignalType, list[dict]] = {signal: [] for signal in SignalType}
        # Global ordered stream: (seq, signal_type, signal_dict)
        self._received_all: list[tuple[int, SignalType, dict]] = []

        # Public attribute for debugging/inspection in tests if needed.
        # Do NOT mutate it directly.
        self.received_signals = self._received_by_type
        if LOG_SIGNALS_TO_FILE:
            self.signal_file_path = os.path.join(
                SIGNALS_DIR,
                f"signal_{ws_url.split(':')[-1]}_{datetime.now().strftime('%H%M%S')}.log",
            )
            Path(SIGNALS_DIR).mkdir(parents=True, exist_ok=True)

    def on_message(self, ws, signal):
        try:
            signal_data = json.loads(signal)
        except Exception:
            logging.exception(f"SignalClient [{self.url}]: failed to parse signal ({len(signal)} bytes)")
            return

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

        with self._cond:
            self._seq += 1
            seq = self._seq
            self._received_by_type[signal_type].append(signal_data)
            self._received_all.append((seq, signal_type, signal_data))
            self._cond.notify_all()

    # TODO: This is a temporary workaround until all tests are migrated to use SignalType enum
    @staticmethod
    def _convert_signal_type(signal_type: SignalType | str) -> SignalType:
        if isinstance(signal_type, SignalType):
            return signal_type
        if isinstance(signal_type, str):
            return SignalType(signal_type)

    def get_all_events(self, signal_type: SignalType | str):
        signal_type = self._convert_signal_type(signal_type)
        with self._cond:
            signals = self._received_by_type.get(signal_type, [])
            return [signal.get("event") for signal in signals]

    def _on_error(self, ws, error):
        logging.error(f"SignalClient [{self.url}]: websocket error: {error}")

    def _on_close(self, ws, close_status_code, close_msg):
        logging.error(f"SignalClient [{self.url}]: websocket connection closed: {close_status_code}, {close_msg}")

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
        self.wsapp.run_forever(ping_interval=30, ping_timeout=10)
        logging.error(f"SignalClient [{self.url}]: run_forever() exited — websocket is dead")

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

    def expect_signal(
        self,
        signal_type,
        count: int = 1,
        accept_fn: Callable[[dict], bool] | None = None,
        pattern: str | None = None,
        predicate: Callable[[dict], bool] | None = None,
        timeout: float = 20,
        start: Literal["now", "beginning"] | int = "now",
    ):
        """
        Create a context manager for expecting signals.

        By default (`start="now"`) it waits only for signals that arrive AFTER entering the context.
        This is the recommended race-safe usage.

        If you need to match signals that could have already arrived (e.g. startup signals),
        pass `start="beginning"` (or an explicit index).

        Args:
            signal_type: The type of signal to expect (SignalType enum or string)
            count: Number of signals to expect (default: 1)
            accept_fn: Optional filter function that takes signal and returns True if accepted
            pattern: Optional string pattern to search for in signal JSON (alternative to accept_fn)
            predicate: Optional predicate function (alternative to accept_fn)
            timeout: Maximum time to wait for signals in seconds (default: 20)
            start: Where to start searching in the per-type signal list:
                - "now" (default): from current end (only new signals)
                - "beginning": from index 0 (includes already received)
                - int: explicit start index

        Returns:
            SignalExpectation context manager

        Example:
            with backend.expect_signal(SignalType.MESSAGES_NEW) as exp:
                sender.send_message(...)
            signal = exp.result
        """
        signal_type = self._convert_signal_type(signal_type)
        return SignalExpectation(
            self,
            signal_type,
            count=count,
            accept_fn=accept_fn,
            pattern=pattern,
            predicate=predicate,
            timeout=timeout,
            start=start,
        )

    def expect_signals_sequence(self, signal_types, timeout: float = 60):
        """
        Create a context manager for expecting multiple signals from a single action.

        This is a convenience method for cases where one action triggers multiple
        sequential signals. It's equivalent to nested expect_signal contexts but
        more readable.

        Args:
            signal_types: List of signal types to expect in sequence
            timeout: Maximum time to wait for ALL signals (default: 60)

        Returns:
            Context manager that waits for all signals

        Example:
            with backend.expect_signals_sequence([
                SignalType.DB_REENCRYPTION_STARTED,
                SignalType.DB_REENCRYPTION_FINISHED,
                SignalType.NODE_STOPPED,
                SignalType.NODE_STARTED,
                SignalType.NODE_READY
            ]):
                backend.change_password(old, new)
        """

        class SequenceExpectation:
            def __init__(self, signal_client, signal_types, timeout):
                self.signal_client = signal_client
                self.signal_types = [signal_client._convert_signal_type(st) for st in signal_types]
                self.timeout = timeout
                self.results: list[dict] = []
                self._start_pos = 0

            def __enter__(self):
                with self.signal_client._cond:
                    self._start_pos = len(self.signal_client._received_all)
                return self

            def __exit__(self, exc_type, exc_val, exc_tb):
                if exc_type is not None:
                    return False

                deadline = time.time() + float(self.timeout)
                pos = self._start_pos
                expected = list(self.signal_types)

                with self.signal_client._cond:
                    for expected_type in expected:
                        while True:
                            # Scan global ordered stream from current position
                            stream = self.signal_client._received_all
                            found = None
                            for i in range(pos, len(stream)):
                                _seq, st, data = stream[i]
                                if st == expected_type:
                                    found = (i, data)
                                    break

                            if found is not None:
                                i, data = found
                                self.results.append(data)
                                pos = i + 1
                                break

                            remaining = deadline - time.time()
                            if remaining <= 0:
                                raise TimeoutError(
                                    f"Expected signal sequence {expected} but did not receive {expected_type} " f"within {self.timeout} seconds"
                                )
                            self.signal_client._cond.wait(timeout=remaining)

                return False

        return SequenceExpectation(self, signal_types, timeout)
