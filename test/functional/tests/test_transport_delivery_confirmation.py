import logging
import time
from uuid import uuid4

import pytest

from clients.logos_delivery import LogosDeliveryClient
from clients.signals import SignalType
from steps import messenger

logger = logging.getLogger(__name__)


@pytest.mark.rpc
@pytest.mark.wakuext
class TestDeliveryConfirmation:
    """Regression coverage for outgoing message status (issue #7472).

    Two distinct outgoing states are exercised:

    * ``delivered`` — driven by an MVDS delivery ACK from the recipient. Verified
      with the recipient online (``test_delivery_confirmation_online``).
    * ``sent`` — driven by ``envelope.sent`` (the envelope reaching at least one
      peer, i.e. the store node), independent of MVDS. Verified with the
      recipient offline (``test_sent_status_while_recipient_offline``), so MVDS
      cannot ACK and the only way the status can advance is the envelope-sent
      path. This is the path that ``EnvelopesMonitor`` currently drives and that
      the logos-delivery Messaging API will own once integrated.
    """

    @pytest.fixture()
    def logos_delivery(self):
        return LogosDeliveryClient()

    @pytest.fixture()
    def sender(self, backend_new_profile):
        return backend_new_profile("sender")

    @pytest.fixture()
    def receiver(self, backend_new_profile):
        return backend_new_profile("receiver")

    def _assert_outgoing_status(self, node, message_id, expected_status):
        # The status field is written just before the corresponding signal fires
        # (same loop in markDeliveredMessages / processSentMessage), so once we've
        # seen the signal a single read is enough — no polling needed.
        message = node.wakuext_service.message_by_message_id(message_id)
        actual_status = message.get("outgoingStatus", "")
        assert actual_status == expected_status, f"Message {message_id} outgoing status was '{actual_status}', expected '{expected_status}'"

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

        self._assert_outgoing_status(sender, message_id, "delivered")

    def test_sent_status_while_recipient_offline(self, sender, receiver, logos_delivery):
        messenger.make_contacts(sender, receiver)

        message_text = f"sent_offline_{uuid4()}"

        # Mark the start of the offline window for the store query (Unix ns, with
        # a small buffer for host/container clock skew).
        offline_start_ns = time.time_ns() - 5_000_000_000

        # The recipient stays offline for the whole exchange, so MVDS datasync can
        # never ACK the message. The message can therefore only ever reach "sent"
        # (envelope published to a peer), never "delivered" — which isolates the
        # envelope.sent path from MVDS delivery. All assertions stay inside the
        # pause block so the recipient can't come back and ACK mid-test.
        with messenger.node_pause(receiver):
            response = sender.wakuext_service.send_one_to_one_message(receiver.public_key, message_text)
            message_id = response.get("messages", [])[0].get("id", "")
            assert message_id, "Sender did not get a message id back"

            # `message_id` is only known after the send, so wait post-hoc and scan
            # from the beginning to avoid matching envelope.sent signals from
            # make_contacts.
            with sender.expect_signal(SignalType.ENVELOPE_SENT, pattern=message_id, timeout=120, start="beginning"):
                pass

            # Exact match: "sent" (not "delivered") proves the status came from the
            # envelope-sent path and not from an MVDS ACK.
            self._assert_outgoing_status(sender, message_id, "sent")

            # envelope.sent means the envelope already reached a peer, so it is
            # already persisted at the store node: a single REST read confirms it
            # was published to the network (not just marked locally), with MVDS
            # provably out of the picture (recipient offline). No polling needed.
            stored = logos_delivery.get_messages(start_time=offline_start_ns)
            assert stored, "Store node holds no messages for the offline window; envelope was not published"
