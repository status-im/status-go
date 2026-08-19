from uuid import uuid4

import pytest

from steps import messenger
from clients.signals import SignalType


@pytest.mark.reliability
class TestPostReconnectDelivery:
    """Check delivery of a message sent AFTER the node goes offline and comes back.

    The other node_pause tests only check a message sent while the node is offline,
    and confirm it arrives once the node is back. node_pause freezes the whole
    process, so status-go never sees a connection change and the offline/online
    event never runs. This test sends a real offline->online event through the
    ConnectionChange API, so the reconnect code actually runs.
    """

    @pytest.fixture()
    def community_admin(self, backend_new_profile):
        return backend_new_profile("community_admin", bridge_network=True)

    @pytest.fixture()
    def community_member(self, backend_new_profile):
        return backend_new_profile("community_member", bridge_network=True)

    @pytest.fixture()
    def sender(self, backend_new_profile):
        return backend_new_profile("sender", bridge_network=True)

    @pytest.fixture()
    def receiver(self, backend_new_profile):
        return backend_new_profile("receiver", bridge_network=True)

    @staticmethod
    def _set_connection_state(node, conn_type):
        # "none" means offline; any other value means online.
        node.api_request_json("ConnectionChange", {"type": conn_type, "expensive": False})

    @classmethod
    def _offline_online_cycle(cls, node):
        cls._set_connection_state(node, "none")
        cls._set_connection_state(node, "wifi")
        node.wait_for_online(timeout=30)

    @staticmethod
    def _send_until_received(receiver, send_fn, text, attempts=3, per_attempt_timeout=15):
        # After a reconnect the filters re-subscribe in the background, so retry until it arrives.
        last_exc = None
        for _ in range(attempts):
            try:
                with receiver.expect_signal(SignalType.MESSAGES_NEW, pattern=text, timeout=per_attempt_timeout):
                    send_fn()
                return
            except TimeoutError as exc:
                last_exc = exc
        raise last_exc

    def test_community_message_after_reconnect(self, community_admin, community_member):
        community_id = messenger.create_community(community_admin)
        chat_id = messenger.join_community(member=community_member, admin=community_admin, community_id=community_id)

        baseline = f"baseline_{uuid4()}"
        self._send_until_received(
            community_member,
            lambda: community_admin.wakuext_service.send_chat_message(chat_id, baseline),
            baseline,
        )

        self._offline_online_cycle(community_member)

        after = f"after_reconnect_{uuid4()}"
        self._send_until_received(
            community_member,
            lambda: community_admin.wakuext_service.send_chat_message(chat_id, after),
            after,
        )

    def test_one_to_one_message_after_reconnect(self, sender, receiver):
        """Control: 1:1 messages don't use SDS, so this must still work after a reconnect."""
        messenger.make_contacts(sender, receiver)

        baseline = f"baseline_{uuid4()}"
        self._send_until_received(
            receiver,
            lambda: sender.wakuext_service.send_one_to_one_message(receiver.public_key, baseline),
            baseline,
        )

        self._offline_online_cycle(receiver)

        after = f"after_reconnect_{uuid4()}"
        self._send_until_received(
            receiver,
            lambda: sender.wakuext_service.send_one_to_one_message(receiver.public_key, after),
            after,
        )
