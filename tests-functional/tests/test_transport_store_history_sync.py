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

    A node that is offline while messages are published must backfill them
    from the store node once it reconnects. Community messages are used on
    purpose: 1:1 and private-group chats have protocol-level resend/MVDS
    mechanisms that would mask a broken store query, while community messages
    rely solely on relay + store.
    """

    @pytest.fixture()
    def sender(self, backend_new_profile):
        return backend_new_profile("sender")

    @pytest.fixture()
    def receiver(self, backend_new_profile):
        return backend_new_profile("receiver")

    def _community_message_texts(self, message_count):
        return [f"history_sync_{i}_{uuid4()}" for i in range(message_count)]

    def test_offline_node_backfills_community_messages_from_store(self, sender, receiver):
        community_id = messenger.create_community(sender)
        community_chat_id = messenger.join_community(member=receiver, admin=sender, community_id=community_id)

        # Baseline: confirm live delivery works before going offline.
        messenger.community_messages(community_chat_id, 1, sender=sender, receiver=receiver)

        message_texts = self._community_message_texts(5)
        sent_ids = []

        # Receiver is offline while the sender publishes. These messages can only
        # reach the receiver later via a store query on reconnect.
        with messenger.node_pause(receiver):
            for text in message_texts:
                response = sender.wakuext_service.send_chat_message(community_chat_id, text)
                expected = messenger.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
                sent_ids.append(expected.get("id"))
                time.sleep(0.05)
            # Give the store node time to confirm storage before the receiver returns.
            time.sleep(10)

        receiver.wait_for_online(timeout=60)

        # On reconnect the receiver must backfill every missed message from the store.
        for message_id in sent_ids:
            with receiver.expect_signal(SignalType.MESSAGES_NEW, pattern=message_id, timeout=120, start="beginning"):
                pass

        received_texts = self._collect_chat_texts(receiver, community_chat_id, expected_count=len(message_texts) + 1)
        for text in message_texts:
            assert text in received_texts, f"Missed message '{text}' was not backfilled from the store node"

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
