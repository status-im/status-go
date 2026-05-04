from time import sleep
from uuid import uuid4
import pytest
from steps import messenger
from clients.signals import SignalType
from resources.constants import USE_IPV6, FULL_NODE, LIGHT_CLIENT


@pytest.mark.reliability
@pytest.mark.parametrize(
    "waku_light_client",
    [
        pytest.param(False, id=FULL_NODE),
        pytest.param(True, id=LIGHT_CLIENT, marks=pytest.mark.light_client_7393),
    ],
    indirect=True,
)
class TestPrivateGroupMessages:

    @pytest.fixture()
    def community_admin(self, backend_new_profile, waku_light_client):
        return backend_new_profile("community_admin", waku_light_client=waku_light_client, bridge_network=True)

    @pytest.fixture()
    def community_member(self, backend_new_profile, waku_light_client):
        return backend_new_profile("community_member", waku_light_client=waku_light_client, bridge_network=True)

    def _run_private_group_messages_baseline(self, community_admin, community_member, message_count=1):
        messenger.make_contacts(community_admin, community_member)
        private_group_id = messenger.join_private_group(admin=community_admin, member=community_member)
        messenger.private_group_message(message_count, private_group_id, sender=community_admin, receiver=community_member)

    def test_private_group_messages_baseline(self, community_admin, community_member, message_count=1):
        self._run_private_group_messages_baseline(community_admin, community_member, message_count)

    def test_multiple_group_chat_messages(self, community_admin, community_member):
        self._run_private_group_messages_baseline(community_admin, community_member, message_count=50)

    @pytest.mark.parametrize("backend_factory", [{"privileged": True}], indirect=True)
    def test_private_group_chat_messages_with_latency(self, community_admin, community_member):
        with messenger.add_latency(community_member):
            self._run_private_group_messages_baseline(community_admin, community_member, message_count=50)

    @pytest.mark.parametrize("backend_factory", [{"privileged": True}], indirect=True)
    def test_private_group_chat_messages_with_packet_loss(self, community_admin, community_member):
        with messenger.add_packet_loss(community_member):
            self._run_private_group_messages_baseline(community_admin, community_member, message_count=50)

    @pytest.mark.parametrize("backend_factory", [{"privileged": True}], indirect=True)
    def test_private_group_chat_messages_with_low_bandwidth(self, community_admin, community_member):
        with messenger.add_low_bandwith(community_member):
            self._run_private_group_messages_baseline(community_admin, community_member, message_count=50)

    def test_private_group_messages_with_node_pause_30_seconds(self, community_admin, community_member):
        messenger.make_contacts(community_admin, community_member)
        private_group_id = messenger.join_private_group(admin=community_admin, member=community_member)

        with messenger.node_pause(community_member):
            message_text = f"test_message_{uuid4()}"
            community_admin.wakuext_service.send_group_chat_message(private_group_id, message_text)
            sleep(30)
        with community_member.expect_signal(SignalType.MESSAGES_NEW, pattern=message_text):
            pass
        with community_admin.expect_signal(SignalType.MESSAGE_DELIVERED):
            pass

    @pytest.mark.skipif(USE_IPV6 == "Yes", reason="Test works only with IPV4")
    def test_private_group_messages_with_ip_change(self, community_admin, community_member):
        messenger.make_contacts(community_admin, community_member)
        private_group_id = messenger.join_private_group(admin=community_admin, member=community_member)
        messenger.private_group_message(1, private_group_id, sender=community_admin, receiver=community_member)
        community_member.change_container_ip()
        messenger.private_group_message(1, private_group_id, sender=community_admin, receiver=community_member)
