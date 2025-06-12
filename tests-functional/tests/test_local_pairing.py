import json
import pytest

from clients.signals import SignalType
from clients.status_backend import StatusBackend
from resources.constants import Account
from steps.messenger import MessengerSteps


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
        self.sender = self.initialize_backend(self.await_signals, False)
        self.receiver = self.initialize_backend(self.await_signals, False)

        # Make contacts before local pairing
        self.make_contacts()

        # Local pairing
        receiver_second_device = StatusBackend(self.await_signals)
        receiver_second_device.init_status_backend()

        connection_string = self.receiver.get_connection_string_for_bootstrapping_another_device()
        response = receiver_second_device.input_connection_string_for_bootstrapping(connection_string)
        response = json.loads(response)

        events = receiver_second_device.get_all_signals(signal_type=SignalType.LOCAL_PAIRING.value)
        for event in events:
            assert "error" not in event["event"] or not event["event"]["error"]

        assert response["error"] is None
        assert response["keyUID"] == self.receiver.key_uid

        # Login on the second device
        user = Account(
            password=self.receiver.password,
            address="",
            private_key="",
            passphrase="",
        )
        receiver_second_device.init_status_backend()
        receiver_second_device.login(self.receiver.key_uid, user)
        receiver_second_device.wait_for_login()
        receiver_second_device.find_public_key()
        receiver_second_device.find_key_uid()
        receiver_second_device.wakuext_service.start_messenger()

        # Check that contact is synced
        response = receiver_second_device.wakuext_service.get_contacts()
        assert "error" not in response

        contacts = response["result"]
        print(f"contacts = {contacts}")

        assert len(contacts) == 1
        assert contacts[0]["id"] == self.sender.public_key
