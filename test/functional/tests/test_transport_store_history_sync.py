import logging
import time
from collections import Counter
from uuid import uuid4

import pytest

from clients.logos_delivery import LogosDeliveryClient
from resources.enums import MessageContentType
from steps import messenger

logger = logging.getLogger(__name__)

HISTORY_WINDOW_TOLERANCE_S = 60


@pytest.mark.rpc
@pytest.mark.wakuext
class TestStoreHistorySync:
    @pytest.fixture()
    def logos_delivery(self):
        return LogosDeliveryClient()

    @pytest.fixture()
    def sender(self, backend_new_profile):
        return backend_new_profile("sender")

    @pytest.fixture()
    def receiver(self, backend_new_profile):
        return backend_new_profile("receiver")

    @pytest.fixture()
    def admin(self, backend_new_profile):
        return backend_new_profile("admin")

    @pytest.fixture()
    def member(self, backend_new_profile):
        return backend_new_profile("member")

    def test_offline_node_backfills_history_on_login(self, sender, receiver, logos_delivery):
        messenger.make_contacts(sender, receiver)
        group_chat_id = messenger.join_private_group(admin=sender, member=receiver)

        messenger.private_group_message(1, group_chat_id, sender=sender, receiver=receiver)

        receiver_key_uid = receiver.key_uid
        receiver_password = receiver.password
        sender_key_uid = sender.key_uid
        sender_password = sender.password

        receiver.logout()
        offline_start_ns = time.time_ns() - 5_000_000_000

        message_count = 5
        sent_ids = []
        for i in range(message_count):
            response = sender.wakuext_service.send_group_chat_message(group_chat_id, f"history_sync_{i}_{uuid4()}")
            message = messenger.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
            sent_ids.append(message["id"])
            time.sleep(1)

        logger.info(f"Sent {message_count} messages: {sent_ids}")

        stored = logos_delivery.wait_for_message_count(message_count, start_time=offline_start_ns, timeout=60)
        topic_counts = Counter(m.get("message", {}).get("contentTopic", "?") for m in stored)
        logger.info(f"Store holds {len(stored)} messages since offline start; content topics: {dict(topic_counts)}")
        assert len(stored) >= message_count, f"Store node only holds {len(stored)} messages, expected at least {message_count}"

        sender.logout()

        try:
            receiver.login(receiver_key_uid, receiver_password)
            receiver.wait_for_login()
            receiver.wait_for_wakuext_ready(timeout=30)

            recovered = self._wait_for_message_ids(receiver, group_chat_id, sent_ids, timeout=180)
            missing = [mid for mid in sent_ids if mid not in recovered]
            assert not missing, f"Messages not backfilled from the store on login: {missing}"
        finally:
            try:
                sender.login(sender_key_uid, sender_password)
                sender.wait_for_login()
            except Exception as exc:
                logger.warning("Failed to re-login sender during cleanup: %s", exc)

    def test_community_offline_backfills_history_on_login(self, admin, member, logos_delivery):
        community_id = messenger.create_community(admin)
        chat_id = messenger.join_community(member=member, admin=admin, community_id=community_id)
        messenger.community_messages(chat_id, 1, sender=admin, receiver=member)

        member_key_uid = member.key_uid
        member_password = member.password
        admin_key_uid = admin.key_uid
        admin_password = admin.password

        member.logout()
        offline_start_ns = time.time_ns() - 5_000_000_000

        old_ids = self._send_community_texts(admin, chat_id, 3, "history_cursor_old")
        time.sleep(HISTORY_WINDOW_TOLERANCE_S + 10)
        recent_ids = self._send_community_texts(admin, chat_id, 2, "history_cursor_recent")

        expected_ids = old_ids + recent_ids
        stored = logos_delivery.wait_for_message_count(len(expected_ids), start_time=offline_start_ns, timeout=60)
        assert len(stored) >= len(expected_ids), f"Store node only holds {len(stored)} messages, expected at least {len(expected_ids)}"

        admin.logout()
        try:
            member.login(member_key_uid, member_password)
            member.wait_for_login()
            member.wait_for_wakuext_ready(timeout=30)

            recovered = self._wait_for_message_ids(member, chat_id, expected_ids, timeout=180)
            missing_old = [message_id for message_id in old_ids if message_id not in recovered]
            missing_recent = [message_id for message_id in recent_ids if message_id not in recovered]
            assert not missing_old, f"Community messages older than {HISTORY_WINDOW_TOLERANCE_S}s were not backfilled: {missing_old}"
            assert not missing_recent, f"Recent community messages were not backfilled: {missing_recent}"
        finally:
            try:
                admin.login(admin_key_uid, admin_password)
                admin.wait_for_login()
            except Exception as exc:
                logger.warning("Failed to re-login admin during cleanup: %s", exc)

    def _send_community_texts(self, sender, chat_id, count, prefix):
        sent_ids = []
        for i in range(count):
            response = sender.wakuext_service.send_chat_message(chat_id, f"{prefix}_{i}_{uuid4()}")
            message = messenger.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
            sent_ids.append(message["id"])
            time.sleep(1)
        return sent_ids

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
