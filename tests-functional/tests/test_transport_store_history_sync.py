import logging
import time
from collections import Counter
from uuid import uuid4

import pytest

from clients.logos_delivery import LogosDeliveryClient
from resources.enums import MessageContentType
from steps import messenger

logger = logging.getLogger(__name__)


@pytest.mark.rpc
@pytest.mark.wakuext
class TestStoreHistorySync:
    """Regression coverage for store-backed history sync (issue #7472).

    status-go fetches missed history from the store node on login. This test
    verifies that a user who was fully offline (logged out) while messages were
    published recovers all of them on the next login, for a private group chat.

    The sender is also logged out before the receiver comes back, so MVDS
    datasync cannot retransmit the messages peer-to-peer: the store is then the
    only possible source, which is what we want to exercise. The logos-delivery
    store node is queried directly (REST) to confirm the messages were actually
    persisted before relying on the receiver fetching them.
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

    def test_offline_node_backfills_history_on_login(self, sender, receiver, logos_delivery):
        messenger.make_contacts(sender, receiver)
        group_chat_id = messenger.join_private_group(admin=sender, member=receiver)

        # Baseline: confirm live delivery works before the receiver goes offline.
        messenger.private_group_message(1, group_chat_id, sender=sender, receiver=receiver)

        receiver_key_uid = receiver.key_uid
        receiver_password = receiver.password
        sender_key_uid = sender.key_uid
        sender_password = sender.password

        # Take the receiver fully offline by logging out, so it misses everything
        # the sender publishes next.
        receiver.logout()

        # Mark the start of the offline window for the store query (Unix ns, with
        # a small buffer for host/container clock skew).
        offline_start_ns = time.time_ns() - 5_000_000_000

        message_count = 5
        sent_ids = []
        for i in range(message_count):
            response = sender.wakuext_service.send_group_chat_message(group_chat_id, f"history_sync_{i}_{uuid4()}")
            message = messenger.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
            sent_ids.append(message["id"])
            time.sleep(1)

        logger.info(f"Sent {message_count} messages: {sent_ids}")

        # Confirm the messages actually reached the store node before continuing.
        stored = logos_delivery.wait_for_message_count(message_count, start_time=offline_start_ns, timeout=60)
        topic_counts = Counter(m.get("message", {}).get("contentTopic", "?") for m in stored)
        logger.info(f"Store holds {len(stored)} messages since offline start; content topics: {dict(topic_counts)}")
        assert len(stored) >= message_count, f"Store node only holds {len(stored)} messages, expected at least {message_count}"

        # Log the sender out so it cannot MVDS-retransmit once the receiver
        # returns; the store is now the receiver's only source.
        sender.logout()

        try:
            # Logging back in triggers status-go's historic-message request,
            # which fetches everything missed during the offline period.
            receiver.login(receiver_key_uid, receiver_password)
            receiver.wait_for_login()
            receiver.wait_for_wakuext_ready(timeout=30)

            recovered = self._wait_for_message_ids(receiver, group_chat_id, sent_ids, timeout=180)
            missing = [mid for mid in sent_ids if mid not in recovered]
            assert not missing, f"Messages not backfilled from the store on login: {missing}"
        finally:
            # Re-login the sender so the fixture teardown can log it out cleanly.
            try:
                sender.login(sender_key_uid, sender_password)
                sender.wait_for_login()
            except Exception as exc:
                logger.warning("Failed to re-login sender during cleanup: %s", exc)

    def _wait_for_message_ids(self, node, chat_id, expected_ids, timeout=180):
        deadline = time.time() + timeout
        ids: set[str] = set()
        while time.time() < deadline:
            response = node.wakuext_service.chat_messages(chat_id, limit=50)
            messages = response.get("messages", []) or []
            ids = {m.get("id", "") for m in messages}
            if all(mid in ids for mid in expected_ids):
                return ids
            time.sleep(3)
        return ids
