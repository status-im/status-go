import re
import pytest
from clients.api import ApiResponseError
from clients.services.wakuext import ActivityCenterNotificationType, ContactRequestState
from clients.signals import SignalType, LocalPairingEventType, LocalPairingEventAction
from clients.status_backend import StatusBackend
from resources.constants import Account
from steps.messenger import MessengerSteps
from resources.enums import MessageContentType


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


def wait_for_action_of_type(backend: StatusBackend, action, type):
    backend.wait_for_signal_predicate(
        SignalType.LOCAL_PAIRING.value,
        lambda signal: signal.get("event", {}).get("action") == action and signal.get("event", {}).get("type") == type,
    )


def pair_server_as_sender(sender, receiver, message_sync_enabled=False):
    connection_string = sender.get_connection_string_for_bootstrapping_another_device(message_sync_enabled)
    response = receiver.input_connection_string_for_bootstrapping(connection_string)
    assert response["error"] is None
    assert response["keyUID"] == sender.key_uid

    wait_for_action_of_type(sender, LocalPairingEventAction.ACTION_PAIRING_INSTALLATION.value, LocalPairingEventType.EVENT_PROCESS_SUCCESS.value)
    wait_for_action_of_type(receiver, LocalPairingEventAction.ACTION_PAIRING_INSTALLATION.value, LocalPairingEventType.EVENT_TRANSFER_SUCCESS.value)


def pair_server_as_receiver(sender, receiver):
    connection_string = receiver.get_connection_string_for_being_bootstrapped()
    sender.input_connection_string_for_bootstrapping_another_device(connection_string)

    wait_for_action_of_type(sender, LocalPairingEventAction.ACTION_PAIRING_INSTALLATION.value, LocalPairingEventType.EVENT_PROCESS_SUCCESS.value)
    wait_for_action_of_type(receiver, LocalPairingEventAction.ACTION_PAIRING_INSTALLATION.value, LocalPairingEventType.EVENT_TRANSFER_SUCCESS.value)


def login_paired_device(backend: StatusBackend, key_uid, password):
    user = Account(
        password=password,
        address="",
        private_key="",
        passphrase="",
    )
    backend.init_status_backend()
    backend.login(key_uid, user)
    backend.wait_for_login()
    backend.wakuext_service.start_messenger()


@pytest.mark.rpc
class TestLocalPairing(MessengerSteps):

    await_signals = [
        SignalType.MESSAGES_NEW.value,
        SignalType.MESSAGE_DELIVERED.value,
        SignalType.NODE_LOGIN.value,
        SignalType.NODE_STOPPED.value,
        SignalType.LOCAL_PAIRING.value,
    ]

    @pytest.fixture(autouse=True)
    def setup_cleanup(self, close_status_backend_containers):
        """Automatically cleanup containers after each test"""
        yield

    def initialize_backend(self, await_signals, privileged=True, **kwargs):
        backend = StatusBackend(await_signals, privileged=privileged)
        backend.init_status_backend()
        backend.create_account_and_login(**kwargs)
        backend.wait_for_login()
        backend.wakuext_service.start_messenger()
        return backend

    def test_pairing_server_as_sender(self):
        # Create users
        alice = self.initialize_backend(self.await_signals, False)
        bob = self.initialize_backend(self.await_signals, False)

        bob_second_device = StatusBackend(self.await_signals)
        bob_second_device.init_status_backend()

        # Make contacts before local pairing
        self.make_contacts(alice, bob)

        # Create community before local pairing
        self.create_community(bob)

        # Send a message to Alice before local pairing
        response = bob.wakuext_service.send_one_to_one_message(alice.public_key, "hello alice")
        message = self.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
        message_id1, sender_chat_id = message["id"], message["chatId"]

        # Send a message to Bob before local pairing
        response = alice.wakuext_service.send_one_to_one_message(bob.public_key, "hello bob")
        message = self.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
        self.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
        message_id2 = message["id"]

        # Wait for the message to be delivered
        bob.find_signal_containing_pattern(
            SignalType.MESSAGES_NEW.value,
            event_pattern=message_id2,
            timeout=60,
        )

        # Check that bob has the messages before pairing
        messages = bob.wakuext_service.chat_messages(alice.public_key, limit=10)["messages"]
        assert messages is not None, "No messages found on paired device"
        message_found1 = False
        message_found2 = False
        for message in messages:
            if message["id"] == message_id1 and message["text"] == "hello alice":
                message_found1 = True
                continue
            if message["id"] == message_id2 and message["text"] == "hello bob":
                message_found2 = True
                continue
        assert message_found1, "Message 1 not found on original device"
        assert message_found2, "Message 2 not found on original device"

        # Local pairing WITH message syncing
        pair_server_as_sender(bob, bob_second_device, True)

        # Check sender signals
        events = bob.get_all_events(signal_type=SignalType.LOCAL_PAIRING.value)
        check_server_sender_events(events)

        # Check receiver signals
        events = bob_second_device.get_all_events(signal_type=SignalType.LOCAL_PAIRING.value)
        check_client_receiver_events(events)

        # Login on the second device
        login_paired_device(bob_second_device, bob.key_uid, bob.password)

        # Check that contact is synced
        contacts = bob_second_device.wakuext_service.get_contacts()
        assert len(contacts) == 1
        assert contacts[0]["id"] == alice.public_key

        # Checking the paired devices using accounts_hasPairedDevices method
        alice_response = alice.accounts_service.has_paired_devices()
        assert alice_response is False
        bob_response = bob.accounts_service.has_paired_devices()
        assert bob_response is True

        # Check that the messages are synced
        messages = bob_second_device.wakuext_service.chat_messages(sender_chat_id, limit=10)["messages"]
        assert messages is not None, "No messages found on paired device"
        message_found1 = False
        message_found2 = False
        for message in messages:
            if message["id"] == message_id1 and message["text"] == "hello alice":
                message_found1 = True
                continue
            if message["id"] == message_id2 and message["text"] == "hello bob":
                message_found2 = True
                continue
        assert message_found1, "Message 1 sent before pairing not found on paired device"
        assert message_found2, "Message 2 sent before pairing not found on paired device"

    def test_pairing_server_as_receiver(self):
        # Create users
        alice = self.initialize_backend(self.await_signals, False)
        bob = self.initialize_backend(self.await_signals, False)
        bob_second_device = StatusBackend(self.await_signals)
        bob_second_device.init_status_backend()

        # Make contacts before local pairing
        self.make_contacts(alice, bob)

        # Create community before local pairing
        self.create_community(bob)

        # Send a message to Alice before local pairing
        response = bob.wakuext_service.send_one_to_one_message(alice.public_key, "test_message")
        message = self.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
        _, sender_chat_id = message["id"], message["chatId"]

        # Local pairing WITHOUT message syncing
        pair_server_as_receiver(bob, bob_second_device)

        # Check sender signals
        events = bob.get_all_events(signal_type=SignalType.LOCAL_PAIRING.value)
        check_client_sender_events(events)

        # Check receiver signals
        events = bob_second_device.get_all_events(signal_type=SignalType.LOCAL_PAIRING.value)
        check_server_receiver_events(events)

        # Check that contact is synced
        contacts = bob_second_device.wakuext_service.get_contacts()
        assert len(contacts) == 1
        assert contacts[0]["id"] == alice.public_key

        # Checking the paired devices using accounts_hasPairedDevices method
        alice_response = alice.accounts_service.has_paired_devices()
        assert alice_response is False
        bob_response = bob.accounts_service.has_paired_devices()
        assert bob_response is True

        # Check that the messages are synced
        messages = bob_second_device.wakuext_service.chat_messages(sender_chat_id, limit=10)["messages"]
        assert messages is None, "Messages found on paired device wrongly"

    def test_pairing_three_devices(self):
        # Create users
        bob1 = self.initialize_backend(self.await_signals, False)
        bob2 = StatusBackend(self.await_signals)
        bob2.init_status_backend()
        bob3 = StatusBackend(self.await_signals)
        bob3.init_status_backend()
        user_accepted = self.initialize_backend(self.await_signals, False)
        user_pending = self.initialize_backend(self.await_signals, False)
        user_declined = self.initialize_backend(self.await_signals, False)

        # Setup contacts before local pairing
        self.make_contacts(user_accepted, bob1)
        self.send_contact_request_and_wait_for_signal_to_be_received(user_pending, bob1)
        message_id = self.send_contact_request_and_wait_for_signal_to_be_received(user_declined, bob1)
        bob1.wakuext_service.decline_contact_request(message_id)

        # Pair second device
        pair_server_as_sender(bob1, bob2)

        # Login on the second device
        login_paired_device(bob2, bob1.key_uid, bob1.password)

        # Pair third device from second device
        pair_server_as_sender(bob2, bob3)

        # Login on the third device
        login_paired_device(bob3, bob1.key_uid, bob1.password)

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

    def test_pairing_receiver_must_be_logged_out(self):
        sender = self.initialize_backend(self.await_signals, False)
        receiver = self.initialize_backend(self.await_signals, False)

        # Client receiver must be logged out
        connection_string = sender.get_connection_string_for_bootstrapping_another_device()
        receiver.input_connection_string_for_bootstrapping(connection_string)

        # Server receiver must be logged out
        connection_string = receiver.get_connection_string_for_being_bootstrapped()
        with pytest.raises(ApiResponseError, match=re.escape("[client] status not ok when sending account data")):
            sender.input_connection_string_for_bootstrapping_another_device(connection_string)
