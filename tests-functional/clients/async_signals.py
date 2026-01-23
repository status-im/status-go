from __future__ import annotations

import asyncio
import json
import logging
import time
from collections import deque
from dataclasses import dataclass, field
from typing import Any, Callable, Optional

import websockets
from websockets.exceptions import ConnectionClosed

from clients.signals import SignalType


@dataclass(frozen=True)
class Signal:
    """Immutable signal representation."""

    signal_type: SignalType
    event: dict
    raw: dict
    raw_json: str = ""  # Cached JSON string for O(1) pattern matching
    timestamp: float = field(default_factory=time.time)
    seq: int = 0

    def __post_init__(self):
        # Allow setting seq after creation since frozen=True
        pass


@dataclass
class SignalMatcher:
    """Criteria for matching signals."""

    signal_type: SignalType
    predicate: Optional[Callable[[Signal], bool]] = None

    def matches(self, signal: Signal) -> bool:
        if signal.signal_type != self.signal_type:
            return False
        if self.predicate and not self.predicate(signal):
            return False
        return True


@dataclass
class PendingWaiter:
    """A registered future waiting for a signal."""

    matcher: SignalMatcher
    future: asyncio.Future[Signal]
    created_at: float = field(default_factory=time.time)
    timeout: float = 30.0


class SignalRouter:
    """
    Central signal distribution hub.

    Responsibilities:
    1. Receive signals from WebSocket reader
    2. Match against registered waiters (futures)
    3. Maintain bounded buffer for late waiters
    4. Handle timeout cleanup

    Example:
        router = SignalRouter(buffer_size=1000, buffer_ttl=300.0)
        await router.start()

        # In WebSocket reader
        await router.publish(signal)

        # In test code
        signal = await router.wait_for(SignalType.MESSAGES_NEW, timeout=30)
    """

    def __init__(
        self,
        buffer_size: int = 1000,
        buffer_ttl: float = 300.0,
    ):
        self._waiters: dict[SignalType, list[PendingWaiter]] = {}
        self._buffer: deque[Signal] = deque(maxlen=buffer_size)
        self._buffer_ttl = buffer_ttl
        self._seq = 0
        self._lock = asyncio.Lock()
        self._cleanup_task: Optional[asyncio.Task[None]] = None
        self._started = False

    async def start(self) -> None:
        """Start background cleanup task."""
        if self._started:
            return
        self._started = True
        self._cleanup_task = asyncio.create_task(self._cleanup_loop())

    async def stop(self) -> None:
        """Stop router and cancel pending waiters."""
        self._started = False
        if self._cleanup_task:
            self._cleanup_task.cancel()
            try:
                await self._cleanup_task
            except asyncio.CancelledError:
                pass
            self._cleanup_task = None

        # Cancel all pending waiters
        async with self._lock:
            for waiters in self._waiters.values():
                for waiter in waiters:
                    if not waiter.future.done():
                        waiter.future.cancel()
            self._waiters.clear()

    async def publish(self, signal: Signal) -> None:
        """
        Called by WebSocket reader when a signal arrives.

        1. Assign sequence number
        2. Add to buffer
        3. Match against registered waiters
        4. Complete matching futures
        """
        logging.debug(f"[SignalRouter] Publishing signal: type={signal.signal_type}, seq={self._seq + 1}")
        async with self._lock:
            self._seq += 1
            # Create new Signal with updated seq (since Signal is frozen)
            signal = Signal(
                signal_type=signal.signal_type,
                event=signal.event,
                raw=signal.raw,
                raw_json=signal.raw_json,  # Preserve cached JSON
                timestamp=signal.timestamp,
                seq=self._seq,
            )

            # Add to buffer for late waiters
            self._buffer.append(signal)

            # Find and complete matching waiters
            signal_waiters = self._waiters.get(signal.signal_type, [])
            completed: list[PendingWaiter] = []

            for waiter in signal_waiters:
                if waiter.matcher.matches(signal):
                    if not waiter.future.done():
                        waiter.future.set_result(signal)
                    completed.append(waiter)

            # Remove completed waiters
            for waiter in completed:
                signal_waiters.remove(waiter)

    async def wait_for(
        self,
        signal_type: SignalType,
        *,
        predicate: Optional[Callable[[Signal], bool]] = None,
        pattern: Optional[str] = None,
        timeout: float = 30.0,
        check_buffer: bool = False,  # Default False to avoid O(N) buffer scan
    ) -> Signal:
        """
        Wait for a signal matching the criteria.

        1. Optionally check buffer first (for signals that arrived before wait)
        2. Register future for incoming signals
        3. Await with timeout

        Args:
            signal_type: Type of signal to wait for
            predicate: Optional function to filter signals
            pattern: Optional string pattern to search in signal JSON
            timeout: Maximum wait time in seconds
            check_buffer: If True, check buffer before waiting (O(N) scan).
                Default is False for performance - use True only when you need
                to find signals that may have arrived before calling wait_for()
                (e.g., startup signals like NODE_LOGIN).

        Returns:
            Matching Signal

        Raises:
            asyncio.TimeoutError: If no matching signal within timeout
        """
        logging.debug(f"[SignalRouter] Waiting for signal: type={signal_type}, pattern={pattern}, timeout={timeout}")

        # Build predicate from pattern if provided
        if pattern and not predicate:

            def pattern_predicate(s: Signal) -> bool:
                # Use cached raw_json for O(1) lookup instead of O(n) json.dumps
                if s.raw_json:
                    return pattern in s.raw_json
                return pattern in json.dumps(s.raw)

            predicate = pattern_predicate

        matcher = SignalMatcher(
            signal_type=signal_type,
            predicate=predicate,
        )

        async with self._lock:
            # Check buffer first (for late waiters)
            if check_buffer:
                for signal in reversed(self._buffer):
                    if matcher.matches(signal):
                        return signal

            # Register waiter
            loop = asyncio.get_running_loop()
            future: asyncio.Future[Signal] = loop.create_future()
            waiter = PendingWaiter(
                matcher=matcher,
                future=future,
                timeout=timeout,
            )

            if signal_type not in self._waiters:
                self._waiters[signal_type] = []
            self._waiters[signal_type].append(waiter)

        try:
            return await asyncio.wait_for(future, timeout=timeout)
        except asyncio.TimeoutError:
            # Clean up waiter on timeout
            async with self._lock:
                if signal_type in self._waiters and waiter in self._waiters[signal_type]:
                    self._waiters[signal_type].remove(waiter)
            raise

    async def wait_for_sequence(
        self,
        signal_types: list[SignalType],
        timeout: float = 60.0,
    ) -> list[Signal]:
        """
        Wait for a sequence of signals in order.

        Useful for multi-step operations producing multiple signals.

        Args:
            signal_types: List of signal types to expect in sequence
            timeout: Maximum time to wait for ALL signals

        Returns:
            List of matched signals in order

        Raises:
            asyncio.TimeoutError: If sequence not completed within timeout
        """
        results: list[Signal] = []
        deadline = time.time() + timeout

        for signal_type in signal_types:
            remaining = deadline - time.time()
            if remaining <= 0:
                raise asyncio.TimeoutError(f"Timeout waiting for signal sequence. " f"Got {len(results)}/{len(signal_types)} signals.")

            signal = await self.wait_for(
                signal_type,
                timeout=remaining,
                check_buffer=False,  # Only match signals after previous
            )
            results.append(signal)

        return results

    async def _cleanup_loop(self) -> None:
        """Periodically clean up expired buffer entries and timed-out waiters."""
        while self._started:
            await asyncio.sleep(10.0)

            async with self._lock:
                # Clean expired buffer entries
                now = time.time()
                cutoff = now - self._buffer_ttl
                while self._buffer and self._buffer[0].timestamp < cutoff:
                    self._buffer.popleft()

                # Clean timed-out waiters
                for signal_type, waiters in list(self._waiters.items()):
                    expired = [w for w in waiters if now - w.created_at > w.timeout and not w.future.done()]
                    for waiter in expired:
                        waiter.future.set_exception(asyncio.TimeoutError(f"Signal wait timed out: {signal_type}"))
                        waiters.remove(waiter)

    def get_buffer_signals(
        self,
        signal_type: Optional[SignalType] = None,
    ) -> list[Signal]:
        """
        Get signals from buffer (for debugging/inspection).

        Args:
            signal_type: If provided, filter by signal type

        Returns:
            List of signals (copy)
        """
        signals = list(self._buffer)
        if signal_type:
            signals = [s for s in signals if s.signal_type == signal_type]
        return signals


class AsyncSignalClient:
    """
    Async WebSocket client for receiving signals.

    Connects to status-backend WebSocket and forwards
    parsed signals to the SignalRouter.

    Example:
        router = SignalRouter()
        client = AsyncSignalClient("ws://localhost:8080", router)
        await client.connect()
        # ... use router.wait_for() ...
        await client.disconnect()
    """

    def __init__(
        self,
        ws_url: str,
        router: SignalRouter,
    ):
        # Normalize URL
        if ws_url.endswith("/signals"):
            self.url = ws_url
        else:
            self.url = f"{ws_url}/signals"

        self.router = router
        self._ws: Optional[Any] = None
        self._reader_task: Optional[asyncio.Task[None]] = None
        self._connected = asyncio.Event()
        self._should_stop = False

    async def connect(self) -> None:
        """Establish WebSocket connection and start reader."""
        self._should_stop = False
        self._reader_task = asyncio.create_task(self._reader_loop())
        # Wait for connection to be established
        await asyncio.wait_for(self._connected.wait(), timeout=30.0)

    async def disconnect(self) -> None:
        """Close connection and stop reader."""
        self._should_stop = True
        if self._ws:
            await self._ws.close()
        if self._reader_task:
            self._reader_task.cancel()
            try:
                await self._reader_task
            except asyncio.CancelledError:
                pass
            self._reader_task = None
        self._connected.clear()

    @property
    def is_connected(self) -> bool:
        """Check if connected."""
        return self._connected.is_set()

    async def _reader_loop(self) -> None:
        """Main WebSocket reader loop with reconnection."""
        reconnect_delay = 1.0

        while not self._should_stop:
            try:
                async with websockets.connect(
                    self.url,
                    ping_interval=None,  # Disable client-side pings (server manages keepalive)
                    ping_timeout=None,
                    close_timeout=1,
                ) as ws:
                    self._ws = ws
                    self._connected.set()
                    reconnect_delay = 1.0  # Reset on successful connection
                    logging.debug(f"AsyncSignalClient connected to {self.url}")

                    async for message in ws:
                        if self._should_stop:
                            break
                        await self._handle_message(message)

            except ConnectionClosed as e:
                if not self._should_stop:
                    logging.warning(f"WebSocket connection closed (code={e.code}), reconnecting...")
                    self._connected.clear()
                    await asyncio.sleep(reconnect_delay)
                    reconnect_delay = min(reconnect_delay * 2, 30.0)
            except asyncio.CancelledError:
                break
            except Exception as e:
                if not self._should_stop:
                    logging.error(f"WebSocket error: {e}")
                    self._connected.clear()
                    await asyncio.sleep(reconnect_delay)
                    reconnect_delay = min(reconnect_delay * 2, 30.0)

    async def _handle_message(self, raw_message: str | bytes) -> None:
        """Parse and publish signal to router."""
        try:
            if isinstance(raw_message, bytes):
                raw_message = raw_message.decode("utf-8")
            data = json.loads(raw_message)
            signal_type_str = data.get("type")

            try:
                signal_type = SignalType(signal_type_str)
            except ValueError:
                # Ignore unregistered signal types
                return

            signal = Signal(
                signal_type=signal_type,
                event=data.get("event", {}),
                raw=data,
                raw_json=raw_message,  # Cache original JSON for O(1) pattern matching
            )

            await self.router.publish(signal)

        except json.JSONDecodeError:
            logging.warning(f"Invalid JSON in signal: {raw_message[:100]}")
