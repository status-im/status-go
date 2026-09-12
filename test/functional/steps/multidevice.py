"""Bootstrapping a second device for the same profile."""

import asyncio

from clients.async_status_backend import AsyncStatusBackend
from clients.signals import LocalPairingEventAction, LocalPairingEventType, SignalType


async def wait_for_pairing_action(backend: AsyncStatusBackend, action, event_type, *, timeout=60):
    await backend.wait_for_signal(
        SignalType.LOCAL_PAIRING,
        predicate=lambda s: s.event.get("action") == action and s.event.get("type") == event_type,
        timeout=timeout,
        check_buffer=True,
    )


async def pair_second_device(primary: AsyncStatusBackend, secondary: AsyncStatusBackend):
    """Bootstrap *secondary* as another device of *primary*, with message syncing enabled."""
    connection_string = primary.backend.get_connection_string_for_bootstrapping_another_device(message_sync_enabled=True)
    response = secondary.backend.input_connection_string_for_bootstrapping(connection_string)
    assert response["error"] is None
    assert response["keyUID"] == primary.backend.key_uid

    await asyncio.gather(
        wait_for_pairing_action(
            primary,
            LocalPairingEventAction.ACTION_PAIRING_INSTALLATION.value,
            LocalPairingEventType.EVENT_PROCESS_SUCCESS.value,
        ),
        wait_for_pairing_action(
            secondary,
            LocalPairingEventAction.ACTION_PAIRING_INSTALLATION.value,
            LocalPairingEventType.EVENT_TRANSFER_SUCCESS.value,
        ),
    )


async def login_paired_device(backend: AsyncStatusBackend, key_uid, password):
    backend.backend.init_status_backend()
    backend.backend.login(key_uid, password)
    await backend.wait_for_login(timeout=120.0)
    backend.backend.wakuext_service.start_messenger()
