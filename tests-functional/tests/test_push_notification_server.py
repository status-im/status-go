import json
import logging
import sys
import time
import uuid

import pytest

from clients.gorush_stub import GorushStub
from clients.push_notification_server import PushNotificationServer
from clients.services.wakuext import PushNotificationRegistrationTokenType
from clients.statusgo_container import StatusGoContainer
from resources.enums import MessageContentType
from steps.messenger import MessengerSteps

GORUSH_PORT = 8088
GORUSH_URL = f"http://localhost:{GORUSH_PORT}"

from hashlib import shake_256

def Shake256(msg):
    h = shake_256()
    h.update(msg)
    return "0x" + h.hexdigest(64)

@pytest.mark.usefixtures("setup_two_unprivileged_nodes")
@pytest.mark.rpc
class TestPushNotificationServer(MessengerSteps):
    def test_push_notification_delivery(self):

        # Initialize gorush stub
        # NOTE: Host's `localhost` is not accessible from containers on Linux, so we use the bridge IP instead.
        address = StatusGoContainer.get_bridge_ip() if sys.platform.startswith('linux') else '127.0.0.1'
        gorush = GorushStub(address=address, port=0)

        # Initialize push notification server
        server = PushNotificationServer(gorush_port=gorush.server.server_port)

        # Setup Alice and Bob with push notification server public key
        alice = self.sender
        bob = self.receiver

        # Register devices (simulate or use actual registration logic)
        apn_topic = "im.status.ethereum"
        alice_device_id, alice_platform = str(uuid.uuid4()), PushNotificationRegistrationTokenType.APN_TOKEN
        bob_device_id, bob_platform = str(uuid.uuid4()), PushNotificationRegistrationTokenType.FIREBASE_TOKEN

        response = alice.wakuext_service.register_for_push_notifications(alice_device_id, apn_topic, alice_platform)
        assert "error" not in response

        alice_public_key_hash = Shake256(alice.public_key.encode("utf-8"))
        alice_compressed_public_key = alice.compressed_public_key()
        alice_compressed_public_key_hash = Shake256(bytes.fromhex(alice_compressed_public_key[2:]))

        logging.info(f"Alice device_id {alice_device_id}")
        logging.info(f"Alice registration response: {response}")
        logging.info(f"Alice public key: {alice.public_key}, hash: {alice_public_key_hash}")
        logging.info(f"Alice compressed public key: {alice_compressed_public_key}, hash: {alice_compressed_public_key_hash}")

        response = bob.wakuext_service.register_for_push_notifications(bob_device_id, apn_topic, bob_platform)
        assert "error" not in response

        bob_public_key_hash = Shake256(bob.public_key.encode("utf-8"))
        bob_compressed_public_key = bob.compressed_public_key()
        bob_compressed_public_key_hash = Shake256(bytes.fromhex(bob_compressed_public_key[2:]))

        logging.info(f"Bob device_id {bob_device_id}")
        logging.info(f"Bob registration response: {response}")
        logging.info(f"Bob public key: {bob.public_key}, hash: {bob_public_key_hash}")
        logging.info(f"Bob compressed public key: {bob_compressed_public_key}, hash: {bob_compressed_public_key_hash}")

        time.sleep(5) # WARNING: Wait a few seconds for the devices to be registered

        # Make contacts, this should force delivery of a push notification
        self.make_contacts()

        # Send a message from Alice to Bob
        message_text = f"Message {uuid.uuid4()}"
        response = alice.wakuext_service.send_one_to_one_message(bob.public_key, message_text)
        message_id = self.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0].get("id")
        logging.info(f"Sent message from Alice to Bob: id='{message_id}', text='{message_text}'")

        # Send a message from Bob to Alice
        message_text = f"Message {uuid.uuid4()}"
        response = bob.wakuext_service.send_one_to_one_message(alice.public_key, message_text)
        message_id = self.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0].get("id")
        logging.info(f"Sent message from Bob to Alice: id='{message_id}', text='{message_text}'")

        # Get gorush requests
        alice_device_requests = []
        bob_device_requests = []
        start_time = time.time()

        while len(alice_device_requests) < 1 or len(bob_device_requests) < 2:
            if time.time() - start_time > 30:
                assert False, "Timeout waiting for push notifications"

            time.sleep(1)

            requests = gorush.get_debug_requests()

            for req in requests:
                logging.info(f"gorush request: {req}")
                notifications = json.loads(req)["notifications"]

                for notification in notifications:
                    for token in notification["tokens"]:
                        if token == alice_device_id:
                            alice_device_requests.append(notification)
                        elif token == bob_device_id:
                            bob_device_requests.append(notification)
                        else:
                            assert False, f"Unexpected token in push notification: {token}"

        DEFAULT_PUSH_MESSAGE = "You have a new message"

        logging.info(f"Alice device requests: {alice_device_requests}")
        logging.info(f"Bob device requests: {bob_device_requests}")

        assert len(alice_device_requests) == 1, "Expected 1 push notifications to be received by Alice device"
        assert len(bob_device_requests) == 2, "Expected 2 push notifications to be received by Bob device"

        alice_push = alice_device_requests[0]
        logging.info(f"Alice push notification: {alice_push}")
        assert len(alice_push["tokens"]) == 1 and alice_push["tokens"][0] == alice_device_id, "Expected only Bob device_id in push notification tokens"
        assert alice_push["platform"] == alice_platform.value
        assert alice_push["message"] == DEFAULT_PUSH_MESSAGE
        assert alice_push["data"]["publicKey"] == alice_compressed_public_key_hash
        assert alice_push["data"]["chatId"] == bob_public_key_hash

        for bob_push in bob_device_requests:
            # Bob is expected to receive 2 pushes:
            # - contact request from Alice
            # - the message from Alice
            # Both pushes should have the same content.
            logging.info(f"Bob push notification: {bob_push}")
            assert len(bob_push["tokens"]) == 1 and bob_push["tokens"][0] == bob_device_id, "Expected only Alice device_id in push notification tokens"
            assert bob_push["platform"] == bob_platform.value
            assert bob_push["message"] == DEFAULT_PUSH_MESSAGE
            assert bob_push["data"]["publicKey"] == bob_compressed_public_key_hash
            assert bob_push["data"]["chatId"] == alice_public_key_hash

        # Clean up when done
        gorush.close()
