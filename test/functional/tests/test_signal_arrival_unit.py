"""
Unit tests for the arrival time SignalClient stamps on each signal.

These push frames straight into the client, so they need no status-go backend.
"""

import json
import time

import pytest

from clients.signals import SignalClient, SignalType

# Wide enough to absorb scheduler jitter on a loaded CI box, narrow enough that a stamp taken
# at read time (both stamps then ~equal) cannot pass.
SLEEP_S = 0.3
TOLERANCE_S = 0.15


def _push(client: SignalClient, signal_type: SignalType, event: dict) -> None:
    client.on_message(None, json.dumps({"type": signal_type.value, "event": event}))


def _new_message(message_id: str) -> dict:
    return {"messages": [{"id": message_id}]}


@pytest.mark.rpc  # needs no backend; the marker only puts it in the lane CI actually runs
class TestSignalArrivalTime:
    def test_stamp_is_taken_on_arrival_not_on_read(self):
        client = SignalClient("ws://localhost:0")

        _push(client, SignalType.MESSAGES_NEW, _new_message("first"))
        time.sleep(SLEEP_S)
        _push(client, SignalType.MESSAGES_NEW, _new_message("second"))
        # Read well after both arrived; a read-time stamp would collapse the gap to ~0.
        time.sleep(SLEEP_S)
        read_at = time.monotonic()

        with client.expect_signal(SignalType.MESSAGES_NEW, count=2, start="beginning", timeout=1) as exp:
            pass

        assert exp.arrival_times is not None and len(exp.arrival_times) == 2
        gap = exp.arrival_times[1] - exp.arrival_times[0]
        assert abs(gap - SLEEP_S) <= TOLERANCE_S, f"stamped gap {gap:.3f}s, slept {SLEEP_S}s"
        assert read_at - exp.arrival_times[1] >= SLEEP_S - TOLERANCE_S, "second stamp moved towards read time"
        assert exp.arrived_at == exp.arrival_times[0]

    def test_filtered_match_keeps_its_own_stamp(self):
        client = SignalClient("ws://localhost:0")

        _push(client, SignalType.MESSAGES_NEW, _new_message("first"))
        time.sleep(SLEEP_S)
        _push(client, SignalType.MESSAGES_NEW, _new_message("second"))

        with client.expect_signal(SignalType.MESSAGES_NEW, pattern="second", start="beginning", timeout=1) as exp:
            pass

        snapshot = client.received_with_arrival(SignalType.MESSAGES_NEW)
        assert [s["event"]["messages"][0]["id"] for _, s in snapshot] == ["first", "second"]
        assert exp.result == snapshot[1][1]
        assert exp.arrived_at == snapshot[1][0]
        assert snapshot[1][0] - snapshot[0][0] >= SLEEP_S - TOLERANCE_S

    def test_payload_shape_is_unchanged(self):
        client = SignalClient("ws://localhost:0")
        _push(client, SignalType.MESSAGES_NEW, _new_message("only"))

        assert client.received_signals[SignalType.MESSAGES_NEW] == [{"type": "messages.new", "event": _new_message("only")}]
