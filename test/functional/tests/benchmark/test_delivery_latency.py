"""Time 1:1 messages from the sender's RPC return to the receiver's messages.new signal.

Five rows: sender mode x receiver mode (full/light), plus full->full with the receiver paused at
send time. Each row writes a ``latency`` block into the same metrics JSON the nightly publishes.

A lost or slow message never fails the row: it shows in the numbers and the counts, and an online
row that measures nothing publishes nulls. Setup (login, contact request, wait_for_online) can still fail it,
as in test_basic_benchmark. p50 is the median; ``max`` is the slowest sample, which is what a
nearest-rank p95 also returns at n=11, so it is named for what it is.

Two deliberate timers sit inside every figure. About 1 s of each send-to-receive sample is the
receiver's debounce (``retrieveMessagesDebounceInterval``, recorded as ``debounce_interval_ms``);
the send-to-delivered figure carries that plus the sender's own debounce, and because message i's
ack is drained when message i+1 re-arms the sender's timer, it moves with ``idle_after_receive_s``
and must not be compared across cadences.

The offline row is a batch shape, not a sample set: the receiver drains what was queued during the
pause in a few ``messages.new`` signals (one or two in the runs so far; ``batches`` records the
count), so it reports the batches rather than a per-message percentile.
"""

import json
import logging
import os
import statistics
import time
from uuid import uuid4

import pytest

from clients.signals import SignalType
from clients.status_backend import StatusBackend
from steps import messenger
from utils import fake
from utils.config import Config

MESSAGES_PER_ROW = 12  # the first is reported on its own; the rest are the samples
IDLE_AFTER_RECEIVE_S = 1.0  # one message in flight at a time, so a sample is latency, not queue drain
RECEIVE_TIMEOUT_S = 20  # against a 1.4-3 s floor a 20 s miss is a miss
OFFLINE_FIRST_WAIT_S = 60  # recovery after a pause may legitimately take longer than one receipt
MAX_CONSECUTIVE_TIMEOUTS = 3  # a dead path costs three timeouts, not twelve
DELIVERED_DRAIN_S = 30  # bounded wait for acks still missing after the last receive
OFFLINE_PAUSE_S = 30
MIN_SAMPLES_FOR_STATS = 5
DEBOUNCE_INTERVAL_MS = 1000  # internal/protocol/messenger.go retrieveMessagesDebounceInterval, a build constant

ROWS = [
    pytest.param(False, False, False, id="full_to_full"),
    pytest.param(True, False, False, id="light_to_full"),
    pytest.param(False, True, False, id="full_to_light"),
    pytest.param(True, True, False, id="light_to_light"),
    pytest.param(False, False, True, id="full_to_full_offline"),
]


def _carries_message(signal: dict, message_id: str) -> bool:
    return any(m.get("id") == message_id for m in signal.get("event", {}).get("messages") or [])


def _acks_message(signal: dict, message_id: str) -> bool:
    return signal.get("event", {}).get("messageID") == message_id


def _ms(later: float, earlier: float) -> float:
    return (later - earlier) * 1000


@pytest.mark.benchmark
class TestDeliveryLatency:

    def _new_aut(self, backend_factory, waku_light_client: bool) -> StatusBackend:
        aut = backend_factory("AUT", pprof_enabled=True)
        aut.start_performance_monitoring()
        aut.init_status_backend()
        aut.events.append("CreateAccountAndLogin")
        aut.create_account_and_login(password=fake.profile_password(), waku_light_client=waku_light_client)
        aut.wait_for_login()
        aut.events.append("Logged in")
        aut.wakuext_service.start_messenger()
        return aut

    def _finalize(self, aut: StatusBackend, test_name: str, latency: dict):
        filename = f"{Config.benchmark_results_dir}/{test_name}-{time.strftime('%Y%m%d-%H%M%S')}"
        report = aut.gather_metrics().to_dict()
        report["metrics"]["latency"] = latency
        os.makedirs(Config.benchmark_results_dir, exist_ok=True)
        with open(f"{filename}.json", "w") as f:
            json.dump(report, f, indent=2)
        logging.info(f"Latency report saved to {filename}.json: {latency}")

    def _send(self, sender: StatusBackend, receiver: StatusBackend) -> tuple[str, float, int, int]:
        """Send one message; return (message_id, t_send, receiver MESSAGES_NEW index, sender MESSAGE_DELIVERED index).

        The indices are taken before the send so a fast signal cannot slip in ahead of the waiter.
        t_send is taken after the RPC returns, so the RPC's own duration is not in any interval.
        """
        recv_index = len(receiver.received_signals[SignalType.MESSAGES_NEW])
        ack_index = len(sender.received_signals[SignalType.MESSAGE_DELIVERED])
        response = sender.wakuext_service.send_one_to_one_message(receiver.public_key, f"latency_probe_{uuid4()}")
        t_send = time.monotonic()
        message_id = messenger.get_message_id(response)
        assert message_id, "Sender did not get a message id back"
        return message_id, t_send, recv_index, ack_index

    def _wait_receive(self, receiver: StatusBackend, message_id: str, recv_index: int, timeout: float) -> float | None:
        try:
            with receiver.expect_signal(
                SignalType.MESSAGES_NEW,
                predicate=lambda s: _carries_message(s, message_id),
                timeout=timeout,
                start=recv_index,
            ) as exp:
                pass
        except TimeoutError:
            logging.warning(f"messages.new for {message_id} did not arrive within {timeout}s")
            return None
        return exp.arrived_at

    def _ack_time(self, sender: StatusBackend, message_id: str, ack_index: int) -> float | None:
        acks = sender.received_with_arrival(SignalType.MESSAGE_DELIVERED)[ack_index:]
        return next((arrived_at for arrived_at, s in acks if _acks_message(s, message_id)), None)

    def _drain_acks(self, sender: StatusBackend, pending: dict[str, int]) -> dict[str, float]:
        """Give outstanding acks up to DELIVERED_DRAIN_S in total; whatever is still missing stays missing."""
        found: dict[str, float] = {}
        deadline = time.monotonic() + DELIVERED_DRAIN_S
        while pending and time.monotonic() < deadline:
            for message_id, ack_index in list(pending.items()):
                t_ack = self._ack_time(sender, message_id, ack_index)
                if t_ack is not None:
                    found[message_id] = t_ack
                    del pending[message_id]
            if pending:
                time.sleep(0.5)
        if pending:
            logging.warning(f"{len(pending)} message.delivered signal(s) never arrived: {sorted(pending)}")
        return found

    def _measure_online(self, sender: StatusBackend, receiver: StatusBackend) -> dict:
        """One message in flight at a time: send, wait for the receiver's messages.new, idle, repeat."""
        timings: list[tuple[str, float, float | None, int]] = []  # (message_id, t_send, t_recv, ack_index)
        consecutive_timeouts = 0
        aborted_after = None
        for i in range(MESSAGES_PER_ROW):
            message_id, t_send, recv_index, ack_index = self._send(sender, receiver)
            t_recv = self._wait_receive(receiver, message_id, recv_index, RECEIVE_TIMEOUT_S)
            timings.append((message_id, t_send, t_recv, ack_index))
            consecutive_timeouts = 0 if t_recv is not None else consecutive_timeouts + 1
            if consecutive_timeouts >= MAX_CONSECUTIVE_TIMEOUTS:
                aborted_after = i + 1
                logging.warning(f"{consecutive_timeouts} consecutive receive timeouts; abandoning the row after {aborted_after} sends")
                break
            if i < MESSAGES_PER_ROW - 1:
                time.sleep(IDLE_AFTER_RECEIVE_S)

        acks = self._drain_acks(sender, {message_id: ack_index for message_id, _, _, ack_index in timings})
        # The chat's first *delivered* message carries session setup, so it is reported on its own
        # even when message 0 was lost; the flag says which case this was.
        first_message_ms = None
        first_message_delivered_ms = None
        samples_ms: list[float] = []  # send order, timed messages only
        delivered_ms: list[float | None] = []  # send order, every message after the first delivered one; None when unacked
        receive_timeouts = 0
        for message_id, t_send, t_recv, _ in timings:
            t_ack = acks.get(message_id)
            delivered = _ms(t_ack, t_send) if t_ack is not None else None
            if t_recv is None:
                # The ack is the sender's own observation and stands even when the receiver's signal was missed.
                receive_timeouts += 1
                if first_message_ms is not None:
                    delivered_ms.append(delivered)
                continue
            receive = _ms(t_recv, t_send)
            if first_message_ms is None:
                first_message_ms, first_message_delivered_ms = receive, delivered
            else:
                samples_ms.append(receive)
                delivered_ms.append(delivered)

        acked = [d for d in delivered_ms if d is not None]
        return {
            "send_to_receive_p50_ms": statistics.median(samples_ms) if len(samples_ms) >= MIN_SAMPLES_FOR_STATS else None,
            "send_to_receive_max_ms": max(samples_ms) if len(samples_ms) >= MIN_SAMPLES_FOR_STATS else None,
            "send_to_delivered_p50_ms": statistics.median(acked) if len(acked) >= MIN_SAMPLES_FOR_STATS else None,
            "first_message_ms": first_message_ms,
            "first_message_delivered_ms": first_message_delivered_ms,
            "first_message_timed_out": timings[0][2] is None,
            "samples": len(samples_ms),
            "delivered_samples": len(acked),
            "messages_sent": len(timings),
            "messages_received": len(timings) - receive_timeouts,
            "receive_timeouts": receive_timeouts,
            "aborted_after": aborted_after,
            "samples_ms": samples_ms,
            "delivered_ms": delivered_ms,
        }

    def _measure_offline(self, sender: StatusBackend, receiver: StatusBackend) -> dict:
        """Send with the receiver paused; report how the queued messages come back after the unpause.

        What was queued during the pause is drained by the receiver in a few messages.new signals, so a
        per-message percentile would be one observation repeated. The row reports the batch shape
        instead and omits the per-message leaves; its ack figure is from the unpause too.
        """
        receiver.wait_for_online(timeout=30)
        t_pause = time.monotonic()
        sent: list[tuple[str, float, int, int]] = []
        with messenger.node_pause(receiver):
            for i in range(MESSAGES_PER_ROW):
                sent.append(self._send(sender, receiver))
                if i < MESSAGES_PER_ROW - 1:
                    time.sleep(IDLE_AFTER_RECEIVE_S)
            time.sleep(OFFLINE_PAUSE_S)
        t_unpause = time.monotonic()
        sender.events.append("Receiver unpaused")

        # One bounded wait for the first id, then one shared deadline for whatever is still missing,
        # scanning the arrival snapshot rather than waiting per id: a dead path costs at most
        # OFFLINE_FIRST_WAIT_S + RECEIVE_TIMEOUT_S, and a late batch is credited to every id it carries.
        arrivals: dict[str, float] = {}
        first_id, _, first_index, _ = sent[0]
        t_first = self._wait_receive(receiver, first_id, first_index, OFFLINE_FIRST_WAIT_S)
        if t_first is not None:
            arrivals[first_id] = t_first
        deadline = time.monotonic() + RECEIVE_TIMEOUT_S
        while True:
            for message_id, _, recv_index, _ in sent:
                if message_id not in arrivals:
                    seen = receiver.received_with_arrival(SignalType.MESSAGES_NEW)[recv_index:]
                    t_recv = next((arrived_at for arrived_at, s in seen if _carries_message(s, message_id)), None)
                    if t_recv is not None:
                        arrivals[message_id] = t_recv
            if len(arrivals) == len(sent) or time.monotonic() >= deadline:
                break
            time.sleep(0.5)
        if len(arrivals) < len(sent):
            logging.warning(f"{len(sent) - len(arrivals)} queued message(s) never arrived within {RECEIVE_TIMEOUT_S}s of the first batch window")
        acks = self._drain_acks(sender, {message_id: ack_index for message_id, _, _, ack_index in sent})

        stamps = sorted(arrivals.values())
        batches = sorted(set(stamps))  # one messages.new signal stamps every message it carries identically
        samples_ms = [_ms(arrivals[m], t_unpause) if m in arrivals else None for m, _, _, _ in sent]
        delivered_ms = [_ms(acks[m], t_unpause) if m in acks else None for m, _, _, _ in sent]
        acked = [d for d in delivered_ms if d is not None]
        return {
            "samples": 0,
            "delivered_samples": len(acked),
            "messages_sent": len(sent),
            "messages_received": len(arrivals),
            "receive_timeouts": len(sent) - len(arrivals),
            "aborted_after": None,
            "pause_seconds": t_unpause - t_pause,
            "unpause_to_first_batch_ms": _ms(batches[0], t_unpause) if batches else None,
            "unpause_to_last_ms": _ms(batches[-1], t_unpause) if batches else None,
            "unpause_to_delivered_p50_ms": statistics.median(acked) if len(acked) >= MIN_SAMPLES_FOR_STATS else None,
            "messages_in_first_batch": stamps.count(batches[0]) if batches else 0,
            "batches": len(batches),
            "samples_ms": samples_ms,  # send order, from the unpause, None when never received
            "delivered_ms": delivered_ms,
        }

    @pytest.mark.parametrize("sender_light,receiver_light,receiver_offline", ROWS)
    def test_delivery_latency(self, request, backend_factory, backend_new_profile, sender_light, receiver_light, receiver_offline):
        sender = self._new_aut(backend_factory, sender_light)
        latency: dict = {
            "fleet": Config.waku_fleet,
            "fleets_config": Config.waku_fleets_config or "builtin",
            "sender_light": sender_light,
            "receiver_light": receiver_light,
            "debounce_interval_ms": DEBOUNCE_INTERVAL_MS,
            "idle_after_receive_s": IDLE_AFTER_RECEIVE_S,
            "receive_timeout_s": RECEIVE_TIMEOUT_S,
        }
        request.addfinalizer(lambda: self._finalize(sender, request.node.name, latency))

        receiver = backend_new_profile("receiver", waku_light_client=receiver_light)
        messenger.make_contacts(sender, receiver)
        sender.events.append("Contacts made")

        measure = self._measure_offline if receiver_offline else self._measure_online
        latency.update(measure(sender, receiver))
        sender.events.append("Measurement done")
        logging.info(f"{request.node.name}: {latency}")
