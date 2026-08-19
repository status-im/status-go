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


class TestSignalRouterExpect:
    """Tests for SignalRouter.expect() context manager - race-condition-free waiting."""

    @pytest.mark.asyncio
    async def test_expect_basic_flow(self):
        """Test basic expect() flow - waiter registered before action."""
        router = SignalRouter()
        await router.start()

        try:
            async with router.expect(SignalType.NODE_READY, timeout=5.0) as exp:
                # Waiter is registered here - publish signal
                signal = _make_signal(SignalType.NODE_READY, {"ok": True})
                await router.publish(signal)

            # Signal is automatically awaited when context exits
            assert exp.result is not None
            assert exp.result.signal_type == SignalType.NODE_READY
            assert exp.result.event["ok"] is True

        finally:
            await router.stop()

    @pytest.mark.asyncio
    async def test_expect_signal_already_in_buffer(self):
        """Test expect() finds signal that's already in buffer."""
        router = SignalRouter()
        await router.start()

        try:
            # Publish signal BEFORE expect()
            await router.publish(_make_signal(SignalType.NODE_READY, {"ok": True}))

            async with router.expect(SignalType.NODE_READY, timeout=1.0) as exp:
                # Signal already in buffer - result should be set immediately
                pass

            assert exp.result is not None
            assert exp.result.signal_type == SignalType.NODE_READY
            assert exp.result.event["ok"] is True

        finally:
            await router.stop()

    @pytest.mark.asyncio
    async def test_expect_with_pattern(self):
        """Test expect() with pattern matching."""
        router = SignalRouter()
        await router.start()

        try:
            async with router.expect(SignalType.MESSAGES_NEW, pattern="msg-123", timeout=5.0) as exp:
                # Publish non-matching signal first
                await router.publish(
                    _make_signal(
                        SignalType.MESSAGES_NEW,
                        {"id": "msg-456"},
                        {"type": "messages.new", "event": {"id": "msg-456"}},
                    )
                )
                # Publish matching signal
                await router.publish(
                    _make_signal(
                        SignalType.MESSAGES_NEW,
                        {"id": "msg-123"},
                        {"type": "messages.new", "event": {"id": "msg-123"}},
                    )
                )

            assert exp.result is not None
            assert exp.result.event["id"] == "msg-123"

        finally:
            await router.stop()

    @pytest.mark.asyncio
    async def test_expect_with_predicate(self):
        """Test expect() with predicate filtering."""
        router = SignalRouter()
        await router.start()

        try:
            async with router.expect(
                SignalType.NODE_LOGIN,
                predicate=lambda s: s.event.get("success") is True,
                timeout=5.0,
            ) as exp:
                # Publish non-matching
                await router.publish(_make_signal(SignalType.NODE_LOGIN, {"success": False}))
                # Publish matching
                await router.publish(_make_signal(SignalType.NODE_LOGIN, {"success": True}))

            assert exp.result is not None
            assert exp.result.event["success"] is True

        finally:
            await router.stop()

    @pytest.mark.asyncio
    async def test_expect_timeout_on_await(self):
        """Test that timeout is applied when context exits."""
        router = SignalRouter()
        await router.start()

        try:
            # Timeout should be raised when context exits (auto-await in finally)
            with pytest.raises(asyncio.TimeoutError):
                async with router.expect(SignalType.NODE_READY, timeout=0.1):
                    # Don't publish anything - will timeout on exit
                    pass

        finally:
            await router.stop()

    @pytest.mark.asyncio
    async def test_expect_cleanup_on_normal_exit(self):
        """Test that waiter is cleaned up after context exit."""
        router = SignalRouter()
        await router.start()

        try:
            async with router.expect(SignalType.NODE_READY, timeout=1.0) as exp:
                await router.publish(_make_signal(SignalType.NODE_READY, {"ok": True}))

            # Verify result is populated
            assert exp.result is not None

            # Verify waiter was cleaned up
            assert SignalType.NODE_READY not in router._waiters or len(router._waiters[SignalType.NODE_READY]) == 0

        finally:
            await router.stop()

    @pytest.mark.asyncio
    async def test_expect_cleanup_on_exception(self):
        """Test that waiter is cleaned up even if exception occurs inside context."""
        router = SignalRouter()
        await router.start()

        try:
            with pytest.raises(RuntimeError, match="Test error"):
                async with router.expect(SignalType.NODE_READY, timeout=1.0):
                    raise RuntimeError("Test error")

            # Verify waiter was cleaned up despite exception
            assert SignalType.NODE_READY not in router._waiters or len(router._waiters[SignalType.NODE_READY]) == 0

        finally:
            await router.stop()

    @pytest.mark.asyncio
    async def test_expect_no_race_condition(self):
        """
        Test that expect() eliminates race condition.

        This is the key test - verifies that signal published immediately
        after context entry is NOT missed.
        """
        router = SignalRouter()
        await router.start()

        try:
            # Run multiple iterations to increase confidence
            for i in range(10):
                async with router.expect(SignalType.MESSAGES_NEW, pattern=f"msg-{i}", timeout=1.0) as exp:
                    # Publish immediately - no sleep!
                    await router.publish(
                        _make_signal(
                            SignalType.MESSAGES_NEW,
                            {"id": f"msg-{i}"},
                            {"type": "messages.new", "event": {"id": f"msg-{i}"}},
                        )
                    )

                assert exp.result is not None
                assert exp.result.event["id"] == f"msg-{i}"

        finally:
            await router.stop()

    @pytest.mark.asyncio
    async def test_expect_multiple_concurrent(self):
        """Test multiple concurrent expect() contexts."""
        router = SignalRouter()
        await router.start()

        try:
            # Use nested expects (both waiters registered before any publish)
            async with router.expect(SignalType.NODE_READY, timeout=1.0) as exp1:
                async with router.expect(SignalType.NODE_LOGIN, timeout=1.0) as exp2:
                    # Both waiters registered - publish signals
                    await router.publish(_make_signal(SignalType.NODE_LOGIN, {"uid": "123"}))
                    await router.publish(_make_signal(SignalType.NODE_READY, {"ok": True}))

                # exp2 result is available after inner context exits
                assert exp2.result is not None
                assert exp2.result.signal_type == SignalType.NODE_LOGIN

            # exp1 result is available after outer context exits
            assert exp1.result is not None
            assert exp1.result.signal_type == SignalType.NODE_READY

        finally:
            await router.stop()

    @pytest.mark.asyncio
    async def test_expect_same_pattern_all_receive(self):
        """Test that when multiple expects match same signal, ALL receive it.

        This documents the current behavior: all matching waiters get the same signal.
        This is useful for scenarios where multiple parts of code need the same notification.
        """
        router = SignalRouter()
        await router.start()

        try:
            async with router.expect(SignalType.MESSAGES_NEW, pattern="msg-X", timeout=1.0) as exp1:
                async with router.expect(SignalType.MESSAGES_NEW, pattern="msg-X", timeout=1.0) as exp2:
                    # Publish one signal matching both waiters
                    await router.publish(
                        _make_signal(
                            SignalType.MESSAGES_NEW,
                            {"id": "msg-X"},
                            {"type": "messages.new", "event": {"id": "msg-X"}},
                        )
                    )

                # exp2 result available after inner context exits
                assert exp2.result is not None
                assert exp2.result.event["id"] == "msg-X"

            # exp1 result available after outer context exits
            assert exp1.result is not None
            assert exp1.result.event["id"] == "msg-X"

            # Both should have the same seq number (same signal)
            assert exp1.result.seq == exp2.result.seq

        finally:
            await router.stop()

    @pytest.mark.asyncio
    async def test_expect_cleanup_on_cancellation(self):
        """Test that waiter is cleaned up when task is cancelled."""
        router = SignalRouter()
        await router.start()

        async def expect_and_wait():
            async with router.expect(SignalType.NODE_READY, timeout=10.0) as exp:
                await asyncio.sleep(10)  # Will be cancelled here
            return exp.result

        try:
            task = asyncio.create_task(expect_and_wait())
            await asyncio.sleep(0.05)  # Let the expect register

            # Verify waiter is registered
            assert SignalType.NODE_READY in router._waiters
            assert len(router._waiters[SignalType.NODE_READY]) == 1

            # Cancel the task
            task.cancel()

            with pytest.raises(asyncio.CancelledError):
                await task

            # Give cleanup time to run
            await asyncio.sleep(0.05)

            # Verify waiter was cleaned up
            assert SignalType.NODE_READY not in router._waiters or len(router._waiters[SignalType.NODE_READY]) == 0

        finally:
            await router.stop()

    @pytest.mark.asyncio
    async def test_expect_predicate_exception_in_buffer(self):
        """Test that exception in predicate during buffer check is handled gracefully."""
        router = SignalRouter()
        await router.start()

        def bad_predicate(s: Signal) -> bool:
            if s.event.get("trigger_error"):
                raise ValueError("Predicate failed!")
            return s.event.get("match", False)

        try:
            # Publish signals: one will trigger error, one will match
            await router.publish(_make_signal(SignalType.NODE_READY, {"trigger_error": True}))
            await router.publish(_make_signal(SignalType.NODE_READY, {"match": True}))

            # Should skip the error-causing signal and find the matching one
            async with router.expect(SignalType.NODE_READY, predicate=bad_predicate, timeout=1.0) as exp:
                pass  # Should find in buffer

            assert exp.result is not None
            assert exp.result.event.get("match") is True

        finally:
            await router.stop()
