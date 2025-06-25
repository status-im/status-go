import pytest

from clients.signals import SignalType, LocalPairingEventType, LocalPairingEventAction
from clients.status_backend import StatusBackend
from resources.constants import Account
from steps.messenger import MessengerSteps


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


@pytest.mark.rpc
class TestLocalPairing(MessengerSteps):

    await_signals = [
        SignalType.MESSAGES_NEW.value,
        SignalType.MESSAGE_DELIVERED.value,
        SignalType.NODE_LOGIN.value,
        SignalType.NODE_LOGOUT.value,
        SignalType.LOCAL_PAIRING.value,
    ]

    def test_pairing_server_as_sender(self):
        # Create users
        alice = self.initialize_backend(self.await_signals, False)
        bob = self.initialize_backend(self.await_signals, False)

        # Make contacts before local pairing
        self.make_contacts(alice, bob)

        # Local pairing
        bob_second_device = StatusBackend(self.await_signals)
        bob_second_device.init_status_backend()

        connection_string = bob.get_connection_string_for_bootstrapping_another_device()
        response = bob_second_device.input_connection_string_for_bootstrapping(connection_string)
        assert response["error"] is None
        assert response["keyUID"] == bob.key_uid

        wait_for_action_of_type(bob, LocalPairingEventAction.ACTION_PAIRING_INSTALLATION.value, LocalPairingEventType.EVENT_PROCESS_SUCCESS.value)
        wait_for_action_of_type(
            bob_second_device, LocalPairingEventAction.ACTION_PAIRING_INSTALLATION.value, LocalPairingEventType.EVENT_TRANSFER_SUCCESS.value
        )

        # Check sender signals
        events = bob.get_all_events(signal_type=SignalType.LOCAL_PAIRING.value)
        check_server_sender_events(events)

        # Check receiver signals
        events = bob_second_device.get_all_events(signal_type=SignalType.LOCAL_PAIRING.value)
        check_client_receiver_events(events)

        # Login on the second device
        user = Account(
            password=bob.password,
            address="",
            private_key="",
            passphrase="",
        )
        bob_second_device.init_status_backend()
        bob_second_device.login(bob.key_uid, user)
        bob_second_device.wait_for_login()
        bob_second_device.wakuext_service.start_messenger()

        # Check that contact is synced
        response = bob_second_device.wakuext_service.get_contacts()
        assert "error" not in response

        contacts = response["result"]
        assert len(contacts) == 1
        assert contacts[0]["id"] == alice.public_key

    def test_pairing_server_as_receiver(self):
        # Create users
        alice = self.initialize_backend(self.await_signals, False)
        bob = self.initialize_backend(self.await_signals, False)

        # Make contacts before local pairing
        self.make_contacts(alice, bob)

        # Local pairing
        bob_second_device = StatusBackend(self.await_signals)
        bob_second_device.init_status_backend()

        connection_string = bob_second_device.get_connection_string_for_being_bootstrapped()
        response = bob.input_connection_string_for_bootstrapping_another_device(connection_string)
        assert response.get("error") in (None, "")

        wait_for_action_of_type(bob, LocalPairingEventAction.ACTION_PAIRING_INSTALLATION.value, LocalPairingEventType.EVENT_PROCESS_SUCCESS.value)
        wait_for_action_of_type(
            bob_second_device, LocalPairingEventAction.ACTION_PAIRING_INSTALLATION.value, LocalPairingEventType.EVENT_TRANSFER_SUCCESS.value
        )

        # Check sender signals
        events = bob.get_all_events(signal_type=SignalType.LOCAL_PAIRING.value)
        check_client_sender_events(events)

        # Check receiver signals
        events = bob_second_device.get_all_events(signal_type=SignalType.LOCAL_PAIRING.value)
        check_server_receiver_events(events)

        # Check that contact is synced
        response = bob_second_device.wakuext_service.get_contacts()
        assert "error" not in response

        contacts = response["result"]
        assert len(contacts) == 1
        assert contacts[0]["id"] == alice.public_key
