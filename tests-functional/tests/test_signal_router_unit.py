"""
Unit tests for SignalRouter and async signal infrastructure.

These tests validate the futures-based signal waiting mechanism
without requiring a real status-go backend.
"""

from __future__ import annotations

import asyncio
from typing import Any, Optional

import pytest

from clients.async_signals import Signal, SignalRouter
from clients.signals import SignalType


def _make_signal(
    signal_type: SignalType,
    event: Optional[dict[str, Any]] = None,
    raw: Optional[dict[str, Any]] = None,
) -> Signal:
    """Helper to create test signals."""
    _event: dict[str, Any] = event if event is not None else {}
    _raw: dict[str, Any] = raw if raw is not None else {"type": signal_type.value, "event": _event}
    return Signal(signal_type=signal_type, event=_event, raw=_raw)


class TestSignalRouter:
    """Tests for SignalRouter class."""

    @pytest.mark.asyncio
    async def test_publish_and_wait_for_signal(self):
        """Test basic publish/wait_for flow."""
        router = SignalRouter()
        await router.start()

        try:
            # Start waiting for signal
            wait_task = asyncio.create_task(router.wait_for(SignalType.NODE_READY, timeout=5.0))

            # Give the task time to register
            await asyncio.sleep(0.01)

            # Publish matching signal
            signal = _make_signal(SignalType.NODE_READY, {"ok": True})
            await router.publish(signal)

            # Verify wait completes
            result = await wait_task
            assert result.signal_type == SignalType.NODE_READY
            assert result.event["ok"] is True

        finally:
            await router.stop()

    @pytest.mark.asyncio
    async def test_wait_for_timeout(self):
        """Test that wait_for raises TimeoutError when no signal arrives."""
        router = SignalRouter()
        await router.start()

        try:
            with pytest.raises(asyncio.TimeoutError):
                await router.wait_for(SignalType.NODE_READY, timeout=0.1)
        finally:
            await router.stop()

    @pytest.mark.asyncio
    async def test_buffer_for_late_waiters(self):
        """Test that buffer allows waiting for signals that already arrived."""
        router = SignalRouter()
        await router.start()

        try:
            # Publish signal BEFORE waiting
            signal = _make_signal(SignalType.NODE_READY, {"ok": True})
            await router.publish(signal)

            # Wait for it - should find in buffer
            result = await router.wait_for(
                SignalType.NODE_READY,
                timeout=1.0,
                check_buffer=True,
            )
            assert result.signal_type == SignalType.NODE_READY
            assert result.event["ok"] is True

        finally:
            await router.stop()

    @pytest.mark.asyncio
    async def test_wait_for_without_buffer_does_not_match_existing(self):
        """Test that check_buffer=False does not match already-arrived signals."""
        router = SignalRouter()
        await router.start()

        try:
            # Publish signal BEFORE waiting
            await router.publish(_make_signal(SignalType.NODE_READY, {"ok": True}))

            # Wait with check_buffer=False - should timeout
            with pytest.raises(asyncio.TimeoutError):
                await router.wait_for(
                    SignalType.NODE_READY,
                    timeout=0.1,
                    check_buffer=False,
                )
        finally:
            await router.stop()

    @pytest.mark.asyncio
    async def test_pattern_matching(self):
        """Test pattern-based signal filtering."""
        router = SignalRouter()
        await router.start()

        try:
            # Start waiting with pattern
            wait_task = asyncio.create_task(
                router.wait_for(
                    SignalType.MESSAGES_NEW,
                    pattern="message-123",
                    timeout=5.0,
                )
            )

            await asyncio.sleep(0.01)

            # Publish non-matching signal
            signal1 = _make_signal(
                SignalType.MESSAGES_NEW,
                {"id": "message-456"},
                {"type": "messages.new", "event": {"id": "message-456"}},
            )
            await router.publish(signal1)

            # Verify still waiting
            assert not wait_task.done()

            # Publish matching signal
            signal2 = _make_signal(
                SignalType.MESSAGES_NEW,
                {"id": "message-123"},
                {"type": "messages.new", "event": {"id": "message-123"}},
            )
            await router.publish(signal2)

            # Verify wait completes with matching signal
            result = await wait_task
            assert result.event["id"] == "message-123"

        finally:
            await router.stop()

    @pytest.mark.asyncio
    async def test_predicate_filtering(self):
        """Test predicate-based signal filtering."""
        router = SignalRouter()
        await router.start()

        try:
            # Wait with predicate
            wait_task = asyncio.create_task(
                router.wait_for(
                    SignalType.NODE_LOGIN,
                    predicate=lambda s: s.event.get("success") is True,
                    timeout=5.0,
                )
            )

            await asyncio.sleep(0.01)

            # Publish non-matching signal
            await router.publish(_make_signal(SignalType.NODE_LOGIN, {"success": False}))

            # Still waiting
            assert not wait_task.done()

            # Publish matching signal
            await router.publish(_make_signal(SignalType.NODE_LOGIN, {"success": True}))

            result = await wait_task
            assert result.event["success"] is True

        finally:
            await router.stop()

    @pytest.mark.asyncio
    async def test_multiple_concurrent_waiters(self):
        """Test multiple concurrent waiters for different signals."""
        router = SignalRouter()
        await router.start()

        try:
            # Start multiple waiters
            wait1 = asyncio.create_task(router.wait_for(SignalType.NODE_READY, timeout=5.0))
            wait2 = asyncio.create_task(router.wait_for(SignalType.NODE_LOGIN, timeout=5.0))
            wait3 = asyncio.create_task(
                router.wait_for(
                    SignalType.MESSAGES_NEW,
                    pattern="msg-1",
                    timeout=5.0,
                )
            )

            await asyncio.sleep(0.01)

            # Publish signals
            await router.publish(_make_signal(SignalType.NODE_LOGIN, {"uid": "123"}))
            await router.publish(
                _make_signal(
                    SignalType.MESSAGES_NEW,
                    {"id": "msg-1"},
                    {"type": "messages.new", "event": {"id": "msg-1"}},
                )
            )
            await router.publish(_make_signal(SignalType.NODE_READY, {"ok": True}))

            # Verify all complete
            result1 = await wait1
            result2 = await wait2
            result3 = await wait3

            assert result1.signal_type == SignalType.NODE_READY
            assert result2.signal_type == SignalType.NODE_LOGIN
            assert result3.event["id"] == "msg-1"

        finally:
            await router.stop()

    @pytest.mark.asyncio
    async def test_parallel_pattern_waiters_same_type(self):
        """Test two waiters for same signal type with different patterns."""
        router = SignalRouter()
        await router.start()

        try:
            wait1 = asyncio.create_task(router.wait_for(SignalType.MESSAGES_NEW, pattern="id1", timeout=5.0))
            wait2 = asyncio.create_task(router.wait_for(SignalType.MESSAGES_NEW, pattern="id2", timeout=5.0))

            await asyncio.sleep(0.01)

            # Publish both signals
            await router.publish(
                _make_signal(
                    SignalType.MESSAGES_NEW,
                    {"id": "id1"},
                    {"type": "messages.new", "event": {"id": "id1"}},
                )
            )
            await router.publish(
                _make_signal(
                    SignalType.MESSAGES_NEW,
                    {"id": "id2"},
                    {"type": "messages.new", "event": {"id": "id2"}},
                )
            )

            result1 = await wait1
            result2 = await wait2

            assert result1.event["id"] == "id1"
            assert result2.event["id"] == "id2"
        finally:
            await router.stop()

    @pytest.mark.asyncio
    async def test_wait_for_sequence(self):
        """Test waiting for a sequence of signals."""
        router = SignalRouter()
        await router.start()

        try:
            # Start sequence wait
            wait_task = asyncio.create_task(
                router.wait_for_sequence(
                    [
                        SignalType.NODE_STARTED,
                        SignalType.NODE_LOGIN,
                        SignalType.NODE_READY,
                    ],
                    timeout=5.0,
                )
            )

            # Give enough time for the coroutine to register the first waiter
            await asyncio.sleep(0.05)

            # Publish signals in order with small delays
            await router.publish(_make_signal(SignalType.NODE_STARTED))
            await asyncio.sleep(0.01)
            await router.publish(_make_signal(SignalType.NODE_LOGIN))
            await asyncio.sleep(0.01)
            await router.publish(_make_signal(SignalType.NODE_READY))

            # Verify sequence received
            results = await wait_task
            assert len(results) == 3
            assert results[0].signal_type == SignalType.NODE_STARTED
            assert results[1].signal_type == SignalType.NODE_LOGIN
            assert results[2].signal_type == SignalType.NODE_READY

        finally:
            await router.stop()

    @pytest.mark.asyncio
    async def test_sequence_ignores_other_signals(self):
        """Test that sequence wait ignores unrelated signals."""
        router = SignalRouter()
        await router.start()

        try:
            wait_task = asyncio.create_task(
                router.wait_for_sequence(
                    [SignalType.NODE_STARTED, SignalType.NODE_READY],
                    timeout=5.0,
                )
            )

            # Give enough time for the coroutine to register the first waiter
            await asyncio.sleep(0.05)

            # Publish with noise
            await router.publish(_make_signal(SignalType.NODE_STARTED))
            await asyncio.sleep(0.01)
            await router.publish(_make_signal(SignalType.MESSAGES_NEW, {"id": "noise"}))
            await router.publish(_make_signal(SignalType.NODE_LOGIN))  # Also noise
            await asyncio.sleep(0.01)
            await router.publish(_make_signal(SignalType.NODE_READY))

            results = await wait_task
            assert len(results) == 2
            assert results[0].signal_type == SignalType.NODE_STARTED
            assert results[1].signal_type == SignalType.NODE_READY

        finally:
            await router.stop()

    @pytest.mark.asyncio
    async def test_buffer_has_bounded_size(self):
        """Test that buffer respects max size."""
        router = SignalRouter(buffer_size=5, buffer_ttl=300.0)
        await router.start()

        try:
            # Publish more signals than buffer size
            for i in range(10):
                await router.publish(_make_signal(SignalType.MESSAGES_NEW, {"i": i}))

            # Buffer should only have last 5
            signals = router.get_buffer_signals()
            assert len(signals) == 5
            # Check that we have signals with i=5,6,7,8,9 (last 5)
            indices = [s.event["i"] for s in signals]
            assert indices == [5, 6, 7, 8, 9]

        finally:
            await router.stop()

    @pytest.mark.asyncio
    async def test_get_buffer_signals_by_type(self):
        """Test filtering buffer by signal type."""
        router = SignalRouter()
        await router.start()

        try:
            await router.publish(_make_signal(SignalType.NODE_READY))
            await router.publish(_make_signal(SignalType.MESSAGES_NEW, {"id": "1"}))
            await router.publish(_make_signal(SignalType.MESSAGES_NEW, {"id": "2"}))
            await router.publish(_make_signal(SignalType.NODE_LOGIN))

            all_signals = router.get_buffer_signals()
            assert len(all_signals) == 4

            messages = router.get_buffer_signals(SignalType.MESSAGES_NEW)
            assert len(messages) == 2
            assert all(s.signal_type == SignalType.MESSAGES_NEW for s in messages)

        finally:
            await router.stop()

    @pytest.mark.asyncio
    async def test_stop_cancels_pending_waiters(self):
        """Test that stop() cancels all pending waiters."""
        router = SignalRouter()
        await router.start()

        # Start waiter
        wait_task = asyncio.create_task(router.wait_for(SignalType.NODE_READY, timeout=60.0))

        await asyncio.sleep(0.05)

        # Stop router
        await router.stop()

        # Give time for cancellation to propagate
        await asyncio.sleep(0.05)

        # Waiter should be cancelled or done
        assert wait_task.cancelled() or wait_task.done()


class TestSignal:
    """Tests for Signal dataclass."""

    def test_signal_creation(self):
        """Test creating a Signal."""
        signal = Signal(
            signal_type=SignalType.NODE_READY,
            event={"ok": True},
            raw={"type": "node.ready", "event": {"ok": True}},
        )

        assert signal.signal_type == SignalType.NODE_READY
        assert signal.event["ok"] is True
        assert signal.raw["type"] == "node.ready"
        assert signal.seq == 0  # Default

    def test_signal_immutability(self):
        """Test that Signal is immutable (frozen dataclass)."""
        from dataclasses import FrozenInstanceError

        signal = Signal(
            signal_type=SignalType.NODE_READY,
            event={},
            raw={},
        )

        with pytest.raises(FrozenInstanceError):
            signal.signal_type = SignalType.NODE_LOGIN  # type: ignore[misc]
