from __future__ import annotations

import asyncio
import logging
import time
from typing import TYPE_CHECKING, Callable, Optional

from clients.async_signals import AsyncSignalClient, Signal, SignalRouter
from clients.signals import SignalType
from resources.enums import ContactVerificationState

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

        # Disconnect sync signal client to avoid duplicate WebSocket connections
        # StatusBackend inherits from SignalClient and starts its own WS in __init__
        # Note: disconnect() may fail if sync client was never connected (skip_signal_client=True)
        try:
            self._backend.disconnect()
        except Exception:
            pass  # Sync client may not have been connected

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
        check_buffer: bool = False,  # Default False to avoid O(N) buffer scan
        after_seq: Optional[int] = None,  # Only match signals with seq > after_seq
    ) -> Signal:
        """
        Wait for a signal.

        Args:
            signal_type: Type of signal to wait for
            pattern: Optional string pattern to match in signal
            predicate: Optional predicate function
            timeout: Timeout in seconds
            check_buffer: If True, check buffered signals first (O(N) scan).
                Default is False for performance - use True only when you need
                to find signals that may have arrived before calling wait_for_signal().
            after_seq: If provided, only match signals with seq > after_seq.
                Used to skip signals that have already been processed.

        Returns:
            Signal that matched criteria
        """
        return await self._router.wait_for(
            signal_type,
            pattern=pattern,
            predicate=predicate,
            timeout=timeout,
            check_buffer=check_buffer,
            after_seq=after_seq,
        )

    async def wait_for_signals_sequence(
        self,
        signal_types: list[SignalType],
        timeout: float = 60.0,
    ) -> list[Signal]:
        """Wait for a sequence of signals in order."""
        return await self._router.wait_for_sequence(signal_types, timeout=timeout)

    async def wait_for_verification_status(
        self,
        challenge: str,
        status: ContactVerificationState,
        timeout: float = 10.0,
    ) -> Signal:
        """Wait for a verification request with specific challenge and status.

        Args:
            challenge: The verification challenge string to match
            status: Expected verification status
            timeout: Timeout in seconds

        Returns:
            Signal containing the matching verification request
        """

        def predicate(s: Signal) -> bool:
            reqs = s.raw.get("event", {}).get("verificationRequests") or []
            return any(r.get("challenge") == challenge and r.get("verification_status") == status.value for r in reqs)

        return await self.wait_for_signal(
            SignalType.MESSAGES_NEW,
            predicate=predicate,
            timeout=timeout,
            check_buffer=True,
        )

    async def wait_for_login(self, timeout: float = 60.0) -> dict:
        """Wait until the backend has completed login.

        Async version of StatusBackend.wait_for_login().
        """

        def _apply_login_signal(signal_data: dict) -> None:
            if "error" in signal_data.get("event", {}):
                error_details = signal_data["event"]["error"]
                assert not error_details, f"Unexpected error during login: {error_details}"
            self._backend.node_login_event = signal_data
            logging.debug(f"Node login event: {self._backend.node_login_event}")
            self._backend.public_key = signal_data.get("event", {}).get("settings", {}).get("public-key", "")
            self._backend.mnemonic = signal_data.get("event", {}).get("settings", {}).get("mnemonic", "")
            self._backend.key_uid = signal_data.get("event", {}).get("account", {}).get("key-uid", "")

        # 1) Preferred path: wait for the `node.login` signal
        try:
            signal = await self.wait_for_signal(SignalType.NODE_LOGIN, timeout=timeout, check_buffer=True)
            signal_data = signal.raw
            assert isinstance(signal_data, dict), f"Unexpected NODE_LOGIN signal payload type: {type(signal_data)}"
            _apply_login_signal(signal_data)
            return signal_data
        except asyncio.TimeoutError:
            logging.warning("NODE_LOGIN signal was not received in time; falling back to RPC polling")

        # 2) Fallback path: poll RPC state until it reflects a logged-in account
        deadline = time.monotonic() + timeout
        last_error = None

        while time.monotonic() < deadline:
            try:
                # Check if signal arrived in buffer while we were polling
                buffered = self._router.get_buffer_signals(SignalType.NODE_LOGIN)
                if buffered:
                    signal_data = buffered[-1].raw
                    _apply_login_signal(signal_data)
                    return signal_data

                # Poll RPC state - use to_thread to avoid blocking event loop
                last_settings = await asyncio.to_thread(self._backend.settings_service.get_settings)
                public_key = (last_settings or {}).get("public-key", "")
                mnemonic = (last_settings or {}).get("mnemonic", "")

                last_keypairs = await asyncio.to_thread(self._backend.accounts_service.get_account_keypairs) or []
                key_uid = ""
                if isinstance(last_keypairs, list) and last_keypairs:
                    key_uid = (last_keypairs[0] or {}).get("key-uid", "")

                if public_key and key_uid:
                    signal_data = {
                        "type": SignalType.NODE_LOGIN.value,
                        "event": {
                            "settings": {"public-key": public_key, "mnemonic": mnemonic},
                            "account": {"key-uid": key_uid},
                        },
                    }
                    _apply_login_signal(signal_data)
                    return signal_data
            except Exception as e:
                last_error = str(e)

            await asyncio.sleep(0.5)

        raise TimeoutError(f"Login did not complete within timeout. last_error={last_error}")

    # === Cleanup ===

    async def shutdown(self, log_sufix: str = "") -> None:
        """Shutdown backend and signal client."""
        await self.stop_signal_client()
        # Run sync shutdown in thread to not block
        await asyncio.to_thread(self._backend.shutdown, log_sufix)
