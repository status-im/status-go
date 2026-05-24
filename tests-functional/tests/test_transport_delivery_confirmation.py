import logging
from time import sleep, time
from uuid import uuid4

import pytest

from clients.signals import SignalType
from steps import messenger

logger = logging.getLogger(__name__)


@pytest.mark.rpc
@pytest.mark.wakuext
class TestDeliveryConfirmation:
    """Regression coverage for delivery confirmation (issue #7472).

    A 1:1 message must progress to outgoing status ``delivered`` and the
    sender must receive a ``message.delivered`` signal, both when the
    recipient is online and when it comes online after a period offline.
    """

    @pytest.fixture()
    def sender(self, backend_new_profile):
        return backend_new_profile("sender")

    @pytest.fixture()
    def receiver(self, backend_new_profile):
        return backend_new_profile("receiver")

    def _wait_for_outgoing_status(self, node, message_id, expected_status="delivered", timeout=120):
        start = time()
        last_status = None
        while time() - start < timeout:
            message = node.wakuext_service.message_by_message_id(message_id)
            last_status = message.get("outgoingStatus", "")
            if last_status == expected_status:
                return
            sleep(2)
        raise AssertionError(f"Message {message_id} outgoing status was '{last_status}', expected '{expected_status}'")

    def test_delivery_confirmation_online(self, sender, receiver):
        messenger.make_contacts(sender, receiver)

        message_text = f"delivery_{uuid4()}"
        response = sender.wakuext_service.send_one_to_one_message(receiver.public_key, message_text)
        message_id = response.get("messages", [])[0].get("id", "")
        assert message_id, "Sender did not get a message id back"

        # `message_id` is only known after the send, so wait post-hoc and scan
        # from the beginning to avoid matching delivery signals from make_contacts.
        with sender.expect_signal(SignalType.MESSAGE_DELIVERED, pattern=message_id, timeout=120, start="beginning"):
            pass

        self._wait_for_outgoing_status(sender, message_id, "delivered")

    def test_delivery_confirmation_after_recipient_offline(self, sender, receiver):
        messenger.make_contacts(sender, receiver)

        message_text = f"delivery_offline_{uuid4()}"

        # The recipient is offline when the message is sent; confirmation must
        # still arrive once it reconnects and the message is delivered.
        with messenger.node_pause(receiver):
            response = sender.wakuext_service.send_one_to_one_message(receiver.public_key, message_text)
            message_id = response.get("messages", [])[0].get("id", "")
            assert message_id, "Sender did not get a message id back"
            sleep(30)

        receiver.wait_for_online(timeout=60)

        with receiver.expect_signal(SignalType.MESSAGES_NEW, pattern=message_id, timeout=120, start="beginning"):
            pass

        with sender.expect_signal(SignalType.MESSAGE_DELIVERED, pattern=message_id, timeout=120, start="beginning"):
            pass

        self._wait_for_outgoing_status(sender, message_id, "delivered")
