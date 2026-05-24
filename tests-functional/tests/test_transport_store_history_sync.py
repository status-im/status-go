import logging
import time
from uuid import uuid4

import pytest

from clients.signals import SignalType
from resources.enums import MessageContentType
from steps import messenger

logger = logging.getLogger(__name__)


@pytest.mark.rpc
@pytest.mark.wakuext
class TestStoreHistorySync:
    """Regression coverage for store-backed history sync (issue #7472).

    A node that is offline while messages are published must recover them once
    it reconnects. The receiver is kept offline long enough for its libp2p
    connection to drop, so the messages are genuinely missed over relay (not
    just buffered at the socket and replayed on resume) and have to be
    backfilled from the store node on reconnect.

    A private group is used as the message vehicle: it reliably establishes a
    multi-party chat without depending on community discovery. The store-only
    delivery path (no resend) is additionally exercised by the persisted
    control message in ``test_transport_ephemeral_messages``.
    """

    @pytest.fixture()
    def sender(self, backend_new_profile):
        return backend_new_profile("sender")

    @pytest.fixture()
    def receiver(self, backend_new_profile):
        return backend_new_profile("receiver")

    def test_offline_node_backfills_messages_after_reconnect(self, sender, receiver):
        messenger.make_contacts(sender, receiver)
        group_chat_id = messenger.join_private_group(admin=sender, member=receiver)

        # Baseline: confirm live delivery works before going offline.
        messenger.private_group_message(1, group_chat_id, sender=sender, receiver=receiver)

        message_texts = [f"history_sync_{i}_{uuid4()}" for i in range(5)]
        sent_ids = []

        # Receiver is offline while the sender publishes; these messages can only
        # reach the receiver later, on reconnect.
        with messenger.node_pause(receiver):
            # Wait for the libp2p connection to time out and drop, so the messages
            # below are genuinely missed over relay.
            time.sleep(60)
            for text in message_texts:
                response = sender.wakuext_service.send_group_chat_message(group_chat_id, text)
                expected = messenger.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
                sent_ids.append(expected.get("id"))
                time.sleep(0.05)
            # Give the store node time to confirm storage before the receiver returns.
            time.sleep(10)

        receiver.wait_for_online(timeout=60)

        # On reconnect the receiver must recover every missed message.
        for message_id in sent_ids:
            with receiver.expect_signal(SignalType.MESSAGES_NEW, pattern=message_id, timeout=120, start="beginning"):
                pass

        received_texts = self._collect_chat_texts(receiver, group_chat_id, expected_count=len(message_texts) + 1)
        for text in message_texts:
            assert text in received_texts, f"Missed message '{text}' was not recovered after reconnect"

    def _collect_chat_texts(self, node, chat_id, expected_count, timeout=120):
        deadline = time.time() + timeout
        texts = []
        while time.time() < deadline:
            response = node.wakuext_service.chat_messages(chat_id, limit=50)
            messages = response.get("messages", []) or []
            texts = [m.get("text", "") for m in messages]
            if len(texts) >= expected_count:
                return texts
            time.sleep(2)
        return texts
