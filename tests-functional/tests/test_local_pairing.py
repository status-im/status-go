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
        # user 1
        self.sender = self.initialize_backend(self.await_signals, False)

        # User 2
        self.receiver = self.initialize_backend(self.await_signals, False)

        # Make contacts
        self.make_contacts()

        # Local pairing
        receiver_second_device = StatusBackend(self.await_signals)
        receiver_second_device.init_status_backend()

        connection_string = self.receiver.get_connection_string_for_bootstrapping_another_device()
        response = receiver_second_device.input_connection_string_for_bootstrapping(connection_string)
        response = json.loads(response)
        print(f"response = {response}")
        
        assert "keyUID" in response, "keyUID not found in response"
        assert response["keyUID"] == self.receiver.key_uid

        # Login on second device
        user = Account(
            password=self.receiver.password,
            address="",
            private_key="",
            passphrase="",
        )
        receiver_second_device.init_status_backend()
        receiver_second_device.login(self.receiver.key_uid, user)
        input("Press Enter to continue...")
        receiver_second_device.wait_for_login()
        receiver_second_device.find_public_key()
        receiver_second_device.find_key_uid()
        receiver_second_device.wakuext_service.start_messenger()

        # Check contacts
        contacts = receiver_second_device.wakuext_service.get_contacts()
        print(f"contacts = {contacts}")
