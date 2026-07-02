import asyncio
import logging
import time
from uuid import uuid4

import pytest

from clients.async_status_backend import AsyncStatusBackend
from clients.signals import LocalPairingEventAction, LocalPairingEventType, SignalType
from resources.enums import MessageContentType
from steps import async_messenger

logger = logging.getLogger(__name__)


async def _wait_for_action_of_type(backend: AsyncStatusBackend, action, event_type, *, timeout=60):
    await backend.wait_for_signal(
        SignalType.LOCAL_PAIRING,
        predicate=lambda s: s.event.get("action") == action and s.event.get("type") == event_type,
        timeout=timeout,
        check_buffer=True,
    )


async def _pair_second_device(primary: AsyncStatusBackend, secondary: AsyncStatusBackend):
    """Bootstrap *secondary* as another device of *primary* with message syncing."""
    connection_string = primary.backend.get_connection_string_for_bootstrapping_another_device(message_sync_enabled=True)
    response = secondary.backend.input_connection_string_for_bootstrapping(connection_string)
    assert response["error"] is None
    assert response["keyUID"] == primary.backend.key_uid

    await asyncio.gather(
        _wait_for_action_of_type(
            primary,
            LocalPairingEventAction.ACTION_PAIRING_INSTALLATION.value,
            LocalPairingEventType.EVENT_PROCESS_SUCCESS.value,
        ),
        _wait_for_action_of_type(
            secondary,
            LocalPairingEventAction.ACTION_PAIRING_INSTALLATION.value,
            LocalPairingEventType.EVENT_TRANSFER_SUCCESS.value,
        ),
    )


async def _login_paired_device(backend: AsyncStatusBackend, key_uid, password):
    backend.backend.init_status_backend()
    backend.backend.login(key_uid, password)
    await backend.wait_for_login(timeout=120.0)
    backend.backend.wakuext_service.start_messenger()


@pytest.mark.rpc
@pytest.mark.asyncio
class TestMultiDeviceSync:
    """Regression coverage for multi-device sync (issue #7472).

    Once a second device is paired, a message sent from one device must
    propagate to the other over the network, and a subsequent edit must sync
    across devices too.
    """

    async def test_message_and_edit_sync_across_devices(self, async_backend_new_profile, async_backend_factory):
        alice, bob, bob_second_device = await asyncio.gather(
            async_backend_new_profile("alice"),
            async_backend_new_profile("bob"),
            async_backend_factory("bob_second_device"),
        )
        bob_second_device.backend.init_status_backend()

        await async_messenger.make_contacts(alice, bob)

        # Pair and bring up Bob's second device with message syncing enabled.
        await _pair_second_device(bob, bob_second_device)
        await _login_paired_device(bob_second_device, bob.backend.key_uid, bob.backend.password)

        # Wait for the second device to connect to the fleet before relying on it
        # receiving anything, so we don't race its Waku subscriptions.
        await asyncio.to_thread(bob_second_device.backend.wait_for_online, timeout=60)

        # A message Bob sends from device 1 must propagate to device 2 via the
        # multi-device sync protocol, and a subsequent edit must sync too.
        outgoing_text = f"from_device_1_{uuid4()}"
        response = bob.wakuext_service.send_one_to_one_message(alice.public_key, outgoing_text)
        message = async_messenger.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
        message_id = message["id"]

        await self._wait_for_text_on_device(bob_second_device, alice.public_key, message_id, outgoing_text, timeout=120)

        edited_text = f"edited_{uuid4()}"
        bob.wakuext_service.edit_message(message_id, edited_text)

        await self._wait_for_text_on_device(bob_second_device, alice.public_key, message_id, edited_text, timeout=120)

    async def _wait_for_text_on_device(self, device, chat_id, message_id, expected_text, timeout=90):
        deadline = time.monotonic() + timeout
        last_text = None
        while time.monotonic() < deadline:
            response = device.wakuext_service.chat_messages(chat_id, limit=20)
            messages = response.get("messages", []) or []
            for message in messages:
                if message.get("id") == message_id:
                    last_text = message.get("text", "")
                    if last_text == expected_text:
                        return
            await asyncio.sleep(2)
        raise AssertionError(f"Device never saw message {message_id} with text '{expected_text}' (last seen: '{last_text}')")
