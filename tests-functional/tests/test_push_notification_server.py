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


@pytest.mark.usefixtures("setup_two_unprivileged_nodes")
@pytest.mark.rpc
class TestPushNotificationServer(MessengerSteps):
    def test_push_notification_delivery(self, push_notification_server):
        server, gorush = push_notification_server
        alice = self.sender
        bob = self.receiver

        # Register devices
        alice.device_platform = PushNotificationRegistrationTokenType.APN_TOKEN
        response = alice.wakuext_service.register_for_push_notifications(alice.device_id, APN_TOPIC, alice.device_platform)
        assert "error" not in response

        bob.device_platform = PushNotificationRegistrationTokenType.FIREBASE_TOKEN
        response = bob.wakuext_service.register_for_push_notifications(bob.device_id, APN_TOPIC, bob.device_platform)
        assert "error" not in response

        # There is currently no way to reliably check if the devices have been registered, so we just wait a few seconds
        time.sleep(5)

        # Make contacts, this should force delivery of a push notification
        self.make_contacts()
        expect_push_notification(gorush, alice, bob)

        # Send a message from Alice to Bob
        alice.wakuext_service.send_one_to_one_message(bob.public_key, f"Message {uuid.uuid4()}")
        expect_push_notification(gorush, alice, bob)

        # Send a message from Bob to Alice
        bob.wakuext_service.send_one_to_one_message(alice.public_key, f"Message {uuid.uuid4()}")
        expect_push_notification(gorush, bob, alice)
