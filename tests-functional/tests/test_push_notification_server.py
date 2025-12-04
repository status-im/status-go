import time
import uuid

import pytest

from clients.gorush_stub import GorushStub
from clients.push_notification_server import PushNotificationServer
from clients.services.wakuext import PushNotificationRegistrationTokenType
from steps.messenger import MessengerSteps
from utils import keys

APN_TOPIC = "im.status.ethereum"
DEFAULT_PUSH_MESSAGE = "You have a new message"


@pytest.fixture
def gorush_stub():
    """
    Fixture that provides a GorushStub instance

    The instance is automatically closed after the test
    """
    gorush = GorushStub(address="0.0.0.0", port=0)  # Use port 0 to get a random available port
    yield gorush
    gorush.close()


@pytest.fixture
def push_notification_server(gorush_stub):
    """
    Fixture that provides a PushNotificationServer connected to a GorushStub

    The PushNotificationServer is automatically stopped after the test
    """
    server = PushNotificationServer(gorush_port=gorush_stub.server.server_port)
    yield server, gorush_stub
    if server.container:
        server.container.stop()


def expect_push_notification(gorush, sender, receiver):
    requests = gorush.wait_for_requests()
    assert len(requests) == 1, "Expected 1 push notification to be received"

    push = requests[0]
    assert len(push["tokens"]) == 1 and push["tokens"][0] == receiver.device_id, "Expected only Bob device_id in push notification tokens"
    assert push["platform"] == receiver.device_platform.value
    assert push["message"] == DEFAULT_PUSH_MESSAGE
    assert push["data"]["publicKey"] == keys.shake256(bytes.fromhex(receiver.compressed_public_key()[2:]))
    assert push["data"]["chatId"] == keys.shake256(sender.public_key.encode("utf-8"))


@pytest.mark.rpc
class TestPushNotificationServer(MessengerSteps):

    @pytest.fixture()
    def sender(self, backend_new_profile):
        """Backend for Alice (push notification sender/receiver)."""
        return backend_new_profile("alice")

    @pytest.fixture()
    def receiver(self, backend_new_profile):
        """Backend for Bob (push notification sender/receiver)."""
        return backend_new_profile("bob")

    def test_push_notification_delivery(self, push_notification_server, sender, receiver):
        server, gorush = push_notification_server

        # Register devices
        sender.device_platform = PushNotificationRegistrationTokenType.APN_TOKEN
        sender.wakuext_service.register_for_push_notifications(sender.device_id, APN_TOPIC, sender.device_platform)

        receiver.device_platform = PushNotificationRegistrationTokenType.FIREBASE_TOKEN
        receiver.wakuext_service.register_for_push_notifications(receiver.device_id, APN_TOPIC, receiver.device_platform)

        # There is currently no way to reliably check if the devices have been registered, so we just wait a few seconds
        time.sleep(10)

        # Make contacts, this should force delivery of a push notification
        self.make_contacts(sender, receiver)
        expect_push_notification(gorush, sender, receiver)

        # Send a message from Alice to Bob
        sender.wakuext_service.send_one_to_one_message(receiver.public_key, f"Message {uuid.uuid4()}")
        expect_push_notification(gorush, sender, receiver)

        # Send a message from Bob to Alice
        receiver.wakuext_service.send_one_to_one_message(sender.public_key, f"Message {uuid.uuid4()}")
        expect_push_notification(gorush, receiver, sender)
