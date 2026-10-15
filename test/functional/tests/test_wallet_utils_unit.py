import json

import pytest

from clients.signals import SignalType, WalletEventType
from clients.wallet_pending_tx_status import WalletPendingTxStatus
from utils.wallet_utils import _record_terminal_status, send_router_transactions_with_signatures


def _status_signal(tx_hash: str, status: str) -> dict:
    return {
        "event": {
            "type": WalletEventType.TRANSACTIONS_PENDING_TRANSACTION_STATUS_CHANGED.value,
            "message": json.dumps({"hash": tx_hash, "status": status}),
        }
    }


class _SignalExpectation:
    def __init__(self, signal: dict | None, predicate):
        self.signal = signal
        self.predicate = predicate
        self.result = None

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        if exc_type is not None:
            return False
        if self.signal is None or not self.predicate(self.signal):
            raise TimeoutError("signal did not match")
        self.result = self.signal
        return False


class _FailingRouterClient:
    def __init__(self, transaction_uuid: str):
        self.transaction_uuid = transaction_uuid
        self.wallet_service = self

    def expect_signal(self, signal_type, *, accept_fn=None, **_kwargs):
        if signal_type == SignalType.WALLET_ROUTER_TRANSACTIONS_SENT:
            signal = {
                "event": {
                    "sendDetails": {
                        "uuid": self.transaction_uuid,
                        "errorResponse": {"message": "send failed"},
                    },
                    "sentTransactions": [],
                }
            }
        else:
            signal = None
        return _SignalExpectation(signal, accept_fn)

    def send_router_transactions_with_signatures(self, _uuid, _signatures):
        return None


@pytest.mark.rpc
def test_terminal_status_waits_for_every_sent_transaction():
    sent_hashes = ["0xapproval", "0xexecution"]
    statuses = {}

    assert not _record_terminal_status(
        _status_signal(sent_hashes[0], WalletPendingTxStatus.SUCCESS),
        sent_hashes,
        statuses,
    )
    assert _record_terminal_status(
        _status_signal(sent_hashes[1], WalletPendingTxStatus.SUCCESS),
        sent_hashes,
        statuses,
    )


@pytest.mark.rpc
def test_terminal_status_records_failure_only_after_every_transaction_finishes():
    sent_hashes = ["0xapproval", "0xexecution"]
    statuses = {}

    assert not _record_terminal_status(
        _status_signal(sent_hashes[1], WalletPendingTxStatus.FAILED),
        sent_hashes,
        statuses,
    )
    assert _record_terminal_status(
        _status_signal(sent_hashes[0], WalletPendingTxStatus.SUCCESS),
        sent_hashes,
        statuses,
    )
    assert statuses[sent_hashes[1]]["status"] == WalletPendingTxStatus.FAILED


@pytest.mark.rpc
def test_router_send_error_is_reported_without_waiting_for_transaction_status():
    transaction_uuid = "route-id"
    client = _FailingRouterClient(transaction_uuid)

    with pytest.raises(AssertionError, match="Router failed to send transactions"):
        send_router_transactions_with_signatures(client, transaction_uuid, {})
