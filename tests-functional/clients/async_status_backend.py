from __future__ import annotations

import asyncio
import logging
from typing import TYPE_CHECKING, Callable, Optional

from clients.async_signals import AsyncSignalClient, Signal, SignalRouter
from clients.signals import SignalType

if TYPE_CHECKING:
    from clients.status_backend import StatusBackend


class AsyncStatusBackend:
    """
    Async wrapper around sync StatusBackend.

    Provides:
    - Direct access to sync RPC services (wakuext_service, wallet_service)
    - Async signal waiting via SignalRouter

    Usage:
        backend = StatusBackend(...)
        async_backend = AsyncStatusBackend(backend)
        await async_backend.start_signal_client()

        # Sync RPC (fast, doesn't block event loop significantly)
        response = async_backend.wakuext_service.send_contact_request(...)

        # Async signal waiting
        signal = await async_backend.wait_for_signal(SignalType.MESSAGES_NEW, pattern="...")
    """

    def __init__(self, backend: "StatusBackend"):
        self._backend = backend
        self._router = SignalRouter()
        self._signal_client: Optional[AsyncSignalClient] = None

    # === Proxy properties to sync backend ===

    @property
    def backend(self) -> "StatusBackend":
        """Access underlying sync backend."""
        return self._backend

    @property
    def wakuext_service(self):
        """Access sync wakuext service."""
        return self._backend.wakuext_service

    @property
    def wallet_service(self):
        """Access sync wallet service."""
        return self._backend.wallet_service

    @property
    def public_key(self) -> str:
        return self._backend.public_key

    @property
    def display_name(self) -> str:
        return self._backend.display_name

    @property
    def data_dir(self) -> str:
        return self._backend.data_dir

    @property
    def router(self) -> SignalRouter:
        """Access signal router."""
        return self._router

    # === Signal client lifecycle ===

    async def start_signal_client(self) -> None:
        """Start WebSocket signal client and router."""
        if self._signal_client is not None:
            return

        await self._router.start()

        ws_url = self._backend.ws_url
        self._signal_client = AsyncSignalClient(ws_url, self._router)
        await self._signal_client.connect()
        logging.debug(f"[AsyncStatusBackend] Signal client started for {ws_url}")

    async def stop_signal_client(self) -> None:
        """Stop WebSocket signal client and router."""
        if self._signal_client:
            await self._signal_client.disconnect()
            self._signal_client = None
        await self._router.stop()
        logging.debug("[AsyncStatusBackend] Signal client stopped")

    # === Async signal waiting ===

    async def wait_for_signal(
        self,
        signal_type: SignalType,
        *,
        pattern: Optional[str] = None,
        predicate: Optional[Callable[[Signal], bool]] = None,
        timeout: float = 30.0,
        check_buffer: bool = True,
    ) -> Signal:
        """
        Wait for a signal.

        Args:
            signal_type: Type of signal to wait for
            pattern: Optional string pattern to match in signal
            predicate: Optional predicate function
            timeout: Timeout in seconds
            check_buffer: If True, check buffered signals first

        Returns:
            Signal that matched criteria
        """
        return await self._router.wait_for(
            signal_type,
            pattern=pattern,
            predicate=predicate,
            timeout=timeout,
            check_buffer=check_buffer,
        )

    async def wait_for_signals_sequence(
        self,
        signal_types: list[SignalType],
        timeout: float = 60.0,
    ) -> list[Signal]:
        """Wait for a sequence of signals in order."""
        return await self._router.wait_for_sequence(signal_types, timeout=timeout)

    # === Cleanup ===

    async def shutdown(self, log_sufix: str = "") -> None:
        """Shutdown backend and signal client."""
        await self.stop_signal_client()
        # Run sync shutdown in thread to not block
        await asyncio.to_thread(self._backend.shutdown, log_sufix)
