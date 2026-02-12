import asyncio
import re

import pytest

from clients.api import ApiResponseError
from clients.async_status_backend import AsyncStatusBackend
from clients.services.wakuext import ActivityCenterNotificationType, ContactRequestState
from clients.signals import LocalPairingEventAction, LocalPairingEventType, SignalType
from resources.enums import MessageContentType
from steps.async_messenger import AsyncMessengerSteps


def check_server_sender_events(events):
    assert len(events) == 8

    assert events[0]["action"] == events[1]["action"] == LocalPairingEventAction.ACTION_PAIRING_ACCOUNT.value
    assert events[0]["type"] == LocalPairingEventType.EVENT_CONNECTION_SUCCESS.value
    assert events[1]["type"] == LocalPairingEventType.EVENT_TRANSFER_SUCCESS.value

    assert events[2]["action"] == events[3]["action"] == LocalPairingEventAction.ACTION_SYNC_DEVICE.value
    assert events[2]["type"] == LocalPairingEventType.EVENT_CONNECTION_SUCCESS.value
    assert events[3]["type"] == LocalPairingEventType.EVENT_TRANSFER_SUCCESS.value

    assert (
        events[4]["action"]
        == events[5]["action"]
        == events[6]["action"]
        == events[7]["action"]
        == LocalPairingEventAction.ACTION_PAIRING_INSTALLATION.value
    )
    assert events[4]["type"] == LocalPairingEventType.EVENT_CONNECTION_SUCCESS.value
    assert events[5]["type"] == LocalPairingEventType.EVENT_TRANSFER_SUCCESS.value
    assert events[6]["type"] == LocalPairingEventType.EVENT_RECEIVED_INSTALLATION.value
    assert events[7]["type"] == LocalPairingEventType.EVENT_PROCESS_SUCCESS.value

    for event in events:
        assert "error" not in event or not event["error"]


def check_client_sender_events(events):
    assert len(events) == 6

    assert events[0]["action"] == LocalPairingEventAction.ACTION_CONNECT.value
    assert events[0]["type"] == LocalPairingEventType.EVENT_CONNECTION_SUCCESS.value

    assert events[1]["action"] == LocalPairingEventAction.ACTION_PAIRING_ACCOUNT.value
    assert events[1]["type"] == LocalPairingEventType.EVENT_TRANSFER_SUCCESS.value

    assert events[2]["action"] == LocalPairingEventAction.ACTION_SYNC_DEVICE.value
    assert events[2]["type"] == LocalPairingEventType.EVENT_TRANSFER_SUCCESS.value

    assert events[3]["action"] == events[4]["action"] == events[5]["action"] == LocalPairingEventAction.ACTION_PAIRING_INSTALLATION.value
    assert events[3]["type"] == LocalPairingEventType.EVENT_TRANSFER_SUCCESS.value
    assert events[4]["type"] == LocalPairingEventType.EVENT_RECEIVED_INSTALLATION.value
    assert events[5]["type"] == LocalPairingEventType.EVENT_PROCESS_SUCCESS.value

    for event in events:
        assert "error" not in event or not event["error"]


def check_client_receiver_events(events):
    assert len(events) == 8

    assert events[0]["action"] == LocalPairingEventAction.ACTION_CONNECT.value
    assert events[0]["type"] == LocalPairingEventType.EVENT_CONNECTION_SUCCESS.value

    assert events[1]["action"] == events[2]["action"] == events[3]["action"] == LocalPairingEventAction.ACTION_PAIRING_ACCOUNT.value
    assert events[1]["type"] == LocalPairingEventType.EVENT_TRANSFER_SUCCESS.value
    assert events[2]["type"] == LocalPairingEventType.EVENT_RECEIVED_ACCOUNT.value
    assert events[3]["type"] == LocalPairingEventType.EVENT_PROCESS_SUCCESS.value

    # NOTE: We check events 4 and 6, for some reason they are out of order (but always the same)
    assert events[4]["action"] == events[6]["action"] == LocalPairingEventAction.ACTION_SYNC_DEVICE.value
    assert events[4]["type"] == LocalPairingEventType.EVENT_TRANSFER_SUCCESS.value
    assert events[6]["type"] == LocalPairingEventType.EVENT_PROCESS_SUCCESS.value

    assert events[5]["action"] == LocalPairingEventAction.ACTION_PAIRING_INSTALLATION.value
    assert events[5]["type"] == LocalPairingEventType.EVENT_RECEIVED_INSTALLATION.value

    assert events[7]["action"] == LocalPairingEventAction.ACTION_PAIRING_INSTALLATION.value
    assert events[7]["type"] == LocalPairingEventType.EVENT_TRANSFER_SUCCESS.value

    for event in events:
        assert "error" not in event or not event["error"]


def check_server_receiver_events(events):
    assert len(events) == 10

    assert (
        events[0]["action"]
        == events[1]["action"]
        == events[2]["action"]
        == events[3]["action"]
        == LocalPairingEventAction.ACTION_PAIRING_ACCOUNT.value
    )
    assert events[0]["type"] == LocalPairingEventType.EVENT_CONNECTION_SUCCESS.value
    assert events[1]["type"] == LocalPairingEventType.EVENT_TRANSFER_SUCCESS.value
    assert events[2]["type"] == LocalPairingEventType.EVENT_RECEIVED_ACCOUNT.value
    assert events[3]["type"] == LocalPairingEventType.EVENT_PROCESS_SUCCESS.value

    assert events[4]["action"] == events[5]["action"] == events[7]["action"] == LocalPairingEventAction.ACTION_SYNC_DEVICE.value
    assert events[4]["type"] == LocalPairingEventType.EVENT_CONNECTION_SUCCESS.value
    assert events[5]["type"] == LocalPairingEventType.EVENT_TRANSFER_SUCCESS.value
    assert events[7]["type"] == LocalPairingEventType.EVENT_PROCESS_SUCCESS.value

    assert events[6]["action"] == events[8]["action"] == events[9]["action"] == LocalPairingEventAction.ACTION_PAIRING_INSTALLATION.value
    assert events[6]["type"] == LocalPairingEventType.EVENT_RECEIVED_INSTALLATION.value
    assert events[8]["type"] == LocalPairingEventType.EVENT_CONNECTION_SUCCESS.value
    assert events[9]["type"] == LocalPairingEventType.EVENT_TRANSFER_SUCCESS.value

    for event in events:
        assert "error" not in event or not event["error"]


def get_all_local_pairing_events(backend: AsyncStatusBackend) -> list[dict]:
    """Get all LOCAL_PAIRING events from async backend's signal buffer."""
    signals = backend.router.get_buffer_signals(SignalType.LOCAL_PAIRING)
    return [s.event for s in signals]


async def wait_for_action_of_type(backend: AsyncStatusBackend, action, type, *, timeout=20):
    """Wait for a LOCAL_PAIRING signal with specific action and type."""
    await backend.wait_for_signal(
        SignalType.LOCAL_PAIRING,
        predicate=lambda s: s.event.get("action") == action and s.event.get("type") == type,
        timeout=timeout,
        check_buffer=True,
    )


async def pair_server_as_sender(sender: AsyncStatusBackend, receiver: AsyncStatusBackend, message_sync_enabled=False):
    connection_string = sender.backend.get_connection_string_for_bootstrapping_another_device(message_sync_enabled)
    response = receiver.backend.input_connection_string_for_bootstrapping(connection_string)
    assert response["error"] is None
    assert response["keyUID"] == sender.backend.key_uid

    # Wait for both signals concurrently (instead of sequential)
    await asyncio.gather(
        wait_for_action_of_type(
            sender,
            LocalPairingEventAction.ACTION_PAIRING_INSTALLATION.value,
            LocalPairingEventType.EVENT_PROCESS_SUCCESS.value,
            timeout=60,
        ),
        wait_for_action_of_type(
            receiver,
            LocalPairingEventAction.ACTION_PAIRING_INSTALLATION.value,
            LocalPairingEventType.EVENT_TRANSFER_SUCCESS.value,
            timeout=60,
        ),
    )


async def pair_server_as_receiver(sender: AsyncStatusBackend, receiver: AsyncStatusBackend):
    connection_string = receiver.backend.get_connection_string_for_being_bootstrapped()
    sender.backend.input_connection_string_for_bootstrapping_another_device(connection_string)

    # Wait for both signals concurrently (instead of sequential)
    await asyncio.gather(
        wait_for_action_of_type(
            sender,
            LocalPairingEventAction.ACTION_PAIRING_INSTALLATION.value,
            LocalPairingEventType.EVENT_PROCESS_SUCCESS.value,
            timeout=60,
        ),
        wait_for_action_of_type(
            receiver,
            LocalPairingEventAction.ACTION_PAIRING_INSTALLATION.value,
            LocalPairingEventType.EVENT_TRANSFER_SUCCESS.value,
            timeout=60,
        ),
    )


async def login_paired_device(backend: AsyncStatusBackend, key_uid, password):
    """Login on a paired device with async signal waiting."""
    backend.backend.init_status_backend()
    backend.backend.login(key_uid, password)
    await backend.wait_for_login(timeout=120.0)
    backend.backend.wakuext_service.start_messenger()


@pytest.mark.rpc
@pytest.mark.asyncio
class TestLocalPairing(AsyncMessengerSteps):

    async def test_pairing_server_as_sender(self, async_backend_new_profile, async_backend_factory):
        alice, bob, bob_second_device = await asyncio.gather(
            async_backend_new_profile("alice"),
            async_backend_new_profile("bob"),
            async_backend_factory("bob_second_device"),
        )
        bob_second_device.backend.init_status_backend()

        # Make contacts before local pairing
        await self.make_contacts(alice, bob)

        # Create community before local pairing
        self.create_community(bob)

        # Send a message to Alice before local pairing
        response = bob.wakuext_service.send_one_to_one_message(alice.public_key, "hello alice")
        message = self.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
        message_id1, sender_chat_id, clock_1 = message["id"], message["chatId"], message["clock"]

        # Send a message to Bob before local pairing
        response = alice.wakuext_service.send_one_to_one_message(bob.public_key, "hello bob")
        message = self.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
        message_id2, clock_2 = message["id"], message["clock"]

        response = bob.wakuext_service.send_emoji_reaction(alice.public_key, message_id2, "1f642")  # 🙂
        assert response["emojiReactions"][0]["emoji"] == "1f642", "Failed to add emoji reaction to message"

        # Wait for the message to be delivered
        await bob.wait_for_signal(SignalType.MESSAGES_NEW, pattern=message_id2, timeout=60)

        # Check that bob has the messages before pairing
        messages = bob.wakuext_service.chat_messages(alice.public_key, limit=10)["messages"]
        assert messages is not None, "No messages found on original device"
        messages_map = {message["id"]: message for message in messages}

        assert message_id1 in messages_map, "Message 1 not found on original device"
        assert messages_map[message_id1]["text"] == "hello alice"

        assert message_id2 in messages_map, "Message 2 not found on original device"
        assert messages_map[message_id2]["text"] == "hello bob"

        # Local pairing WITH message syncing
        await pair_server_as_sender(bob, bob_second_device, True)

        # Check sender signals
        events = get_all_local_pairing_events(bob)
        check_server_sender_events(events)

        # Check receiver signals
        events = get_all_local_pairing_events(bob_second_device)
        check_client_receiver_events(events)

        # Login on the second device
        await login_paired_device(bob_second_device, bob.backend.key_uid, bob.backend.password)

        # Check that contact is synced
        contacts = bob_second_device.wakuext_service.get_contacts()
        assert len(contacts) == 1
        assert contacts[0]["id"] == alice.public_key

        # Checking the paired devices using accounts_hasPairedDevices method
        alice_response = alice.backend.accounts_service.has_paired_devices()
        assert alice_response is False
        bob_response = bob.backend.accounts_service.has_paired_devices()
        assert bob_response is True

        # Check that the messages are synced
        messages = bob_second_device.wakuext_service.chat_messages(sender_chat_id, limit=10)["messages"]
        assert messages is not None, "No messages found on paired device"
        messages_map = {message["id"]: message for message in messages}

        assert message_id1 in messages_map, "Message 1 sent before pairing not found on paired device"
        assert messages_map[message_id1]["text"] == "hello alice"
        assert messages_map[message_id1]["clock"] == clock_1, "Message 1 clock is not right on paired device"

        assert message_id2 in messages_map, "Message 2 sent before pairing not found on paired device"
        assert messages_map[message_id2]["text"] == "hello bob"
        assert messages_map[message_id2]["clock"] == clock_2, "Message 2 clock is not right on paired device"

        # Validate emoji reactions restored on the new client
        result = bob_second_device.wakuext_service.emoji_reactions_by_chat_id_message_id(alice.public_key, message_id2)
        assert result is not None, "Emoji reactions for group chat not restored"
        assert len(result) == 1 and result[0].get("emoji") == "1f642", "Message emoji reaction was not restored correctly"

    async def test_pairing_server_as_receiver(self, async_backend_new_profile, async_backend_factory):
        alice, bob, bob_second_device = await asyncio.gather(
            async_backend_new_profile("alice"),
            async_backend_new_profile("bob"),
            async_backend_factory("bob_second_device"),
        )
        bob_second_device.backend.init_status_backend()

        # Make contacts before local pairing
        await self.make_contacts(alice, bob)

        # Create community before local pairing
        self.create_community(bob)

        # Send a message to Alice before local pairing
        response = bob.wakuext_service.send_one_to_one_message(alice.public_key, "test_message")
        message = self.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
        _, sender_chat_id = message["id"], message["chatId"]

        # Local pairing WITHOUT message syncing
        await pair_server_as_receiver(bob, bob_second_device)

        # Check sender signals
        events = get_all_local_pairing_events(bob)
        check_client_sender_events(events)

        # Check receiver signals
        events = get_all_local_pairing_events(bob_second_device)
        check_server_receiver_events(events)

        # Check that contact is synced
        contacts = bob_second_device.wakuext_service.get_contacts()
        assert len(contacts) == 1
        assert contacts[0]["id"] == alice.public_key

        # Checking the paired devices using accounts_hasPairedDevices method
        alice_response = alice.backend.accounts_service.has_paired_devices()
        assert alice_response is False
        bob_response = bob.backend.accounts_service.has_paired_devices()
        assert bob_response is True

        # Check that the messages are synced
        messages = bob_second_device.wakuext_service.chat_messages(sender_chat_id, limit=10)["messages"]
        assert messages is None, "Messages found on paired device wrongly"

    async def test_pairing_three_devices(self, async_backend_new_profile, async_backend_factory):
        bob1, bob2, bob3, user_accepted, user_pending, user_declined = await asyncio.gather(
            async_backend_new_profile("bob1"),
            async_backend_factory("bob2"),
            async_backend_factory("bob3"),
            async_backend_new_profile("user_accepted"),
            async_backend_new_profile("user_pending"),
            async_backend_new_profile("user_declined"),
        )
        bob2.backend.init_status_backend()
        bob3.backend.init_status_backend()

        # Setup contacts before local pairing
        await self.make_contacts(user_accepted, bob1)
        await self.send_contact_request_and_wait(user_pending, bob1)
        message_id_declined = await self.send_contact_request_and_wait(user_declined, bob1)
        bob1.wakuext_service.decline_contact_request(message_id_declined, user_declined.public_key)

        # Pair second device
        await pair_server_as_sender(bob1, bob2)

        # Login on the second device
        await login_paired_device(bob2, bob1.backend.key_uid, bob1.backend.password)

        # Pair third device from second device
        await pair_server_as_sender(bob2, bob3)

        # Login on the third device
        await login_paired_device(bob3, bob1.backend.key_uid, bob1.backend.password)

        # Check that contacts and notifications are synced on all devices
        for bob_another_device in [bob2, bob3]:
            contacts = bob_another_device.wakuext_service.get_contacts()
            assert len(contacts) == 3
            contacts_dict = {contact["id"]: contact for contact in contacts}
            assert contacts_dict[user_accepted.public_key]["mutual"] is True
            assert contacts_dict[user_accepted.public_key]["contactRequestState"] is ContactRequestState.MUTUAL.value
            assert contacts_dict[user_pending.public_key]["mutual"] is False
            assert contacts_dict[user_pending.public_key]["contactRequestState"] is ContactRequestState.RECEIVED.value
            assert contacts_dict[user_declined.public_key]["mutual"] is False
            assert contacts_dict[user_declined.public_key]["contactRequestState"] is ContactRequestState.DISMISSED.value

            # Paired device will get notifications of requests that are not fulfilled (not mutual)
            notifications = bob_another_device.wakuext_service.get_activity_center_notifications(
                activity_types=[ActivityCenterNotificationType.NOTIFICATION_TYPE_CONTACT_REQUEST]
            )["notifications"]
            assert len(notifications) == 2
            notifications_dict = {notification["chatId"]: notification for notification in notifications}
            user_pending_notification = notifications_dict[user_pending.public_key]
            assert user_pending_notification["read"] is False
            assert user_pending_notification["accepted"] is False
            assert user_pending_notification["dismissed"] is False
            user_declined_notification = notifications_dict[user_declined.public_key]
            assert user_declined_notification["read"] is True
            assert user_declined_notification["accepted"] is False
            assert user_declined_notification["dismissed"] is True

    async def test_pairing_receiver_must_be_logged_out(self, async_backend_new_profile):
        sender, receiver = await asyncio.gather(
            async_backend_new_profile("sender"),
            async_backend_new_profile("receiver"),
        )

        # Client receiver must be logged out
        connection_string = sender.backend.get_connection_string_for_bootstrapping_another_device()
        receiver.backend.input_connection_string_for_bootstrapping(connection_string)

        # Server receiver must be logged out
        connection_string = receiver.backend.get_connection_string_for_being_bootstrapped()
        with pytest.raises(ApiResponseError, match=re.escape("[client] status not ok when sending account data")):
            sender.backend.input_connection_string_for_bootstrapping_another_device(connection_string)
