import logging
import time
from uuid import uuid4

import pytest

from steps import messenger

logger = logging.getLogger(__name__)


@pytest.mark.rpc
@pytest.mark.wakuext
class TestStoreHistorySync:
    """Regression coverage for store-backed history sync (issue #7472).

    status-go fetches missed history from the store node on login. This test
    verifies that a user who was fully offline (logged out) while messages were
    published recovers all of them on the next login, for a private group chat.
    """

    @pytest.fixture()
    def sender(self, backend_new_profile):
        return backend_new_profile("sender")

    @pytest.fixture()
    def receiver(self, backend_new_profile):
        return backend_new_profile("receiver")

    def test_offline_node_backfills_history_on_login(self, sender, receiver):
        messenger.make_contacts(sender, receiver)
        group_chat_id = messenger.join_private_group(admin=sender, member=receiver)

        # Baseline: confirm live delivery works before the receiver goes offline.
        messenger.private_group_message(1, group_chat_id, sender=sender, receiver=receiver)

        receiver_key_uid = receiver.key_uid
        receiver_password = receiver.password

        message_texts = [f"history_sync_{i}_{uuid4()}" for i in range(5)]

        # Take the receiver fully offline by logging out, so it misses everything
        # the sender publishes next.
        receiver.logout()

        for text in message_texts:
            sender.wakuext_service.send_group_chat_message(group_chat_id, text)
            time.sleep(0.05)

        # Give the store node time to persist the messages.
        time.sleep(15)

        # Logging back in triggers status-go's historic-message request, which
        # fetches everything missed during the offline period from the store.
        receiver.login(receiver_key_uid, receiver_password)
        receiver.wait_for_login()
        receiver.wait_for_wakuext_ready(timeout=30)
        receiver.wait_for_online(timeout=60)

        received_texts = self._wait_for_texts(
            receiver,
            group_chat_id,
            expected_texts=message_texts,
            timeout=180,
        )
        missing = [text for text in message_texts if text not in received_texts]
        assert not missing, f"Messages not backfilled from the store on login: {missing}"

    def _wait_for_texts(self, node, chat_id, expected_texts, timeout=180):
        deadline = time.time() + timeout
        texts = set()
        while time.time() < deadline:
            response = node.wakuext_service.chat_messages(chat_id, limit=50)
            messages = response.get("messages", []) or []
            texts = {m.get("text", "") for m in messages}
            if all(text in texts for text in expected_texts):
                return texts
            time.sleep(3)
        return texts
