from time import sleep
from uuid import uuid4
import pytest
from steps import messenger
from clients.signals import SignalType
from resources.constants import USE_IPV6


@pytest.mark.reliability
class TestCommunityMessages:

    @pytest.fixture()
    def community_admin(self, backend_new_profile):
        return backend_new_profile("community_admin", bridge_network=True)

    @pytest.fixture()
    def community_member(self, backend_new_profile):
        return backend_new_profile("community_member", bridge_network=True)

    def _run_community_messages_baseline(
        self,
        community_admin,
        community_member,
        message_count=1,
        network_condition=None,
    ):
        community_id = messenger.create_community(community_admin)
        message_chat_id = messenger.join_community(member=community_member, admin=community_admin, community_id=community_id)
        if network_condition:
            network_condition(community_member)
        messenger.community_messages(message_chat_id, message_count, sender=community_admin, receiver=community_member)

    def test_community_messages_baseline(
        self,
        community_admin,
        community_member,
        message_count=1,
        network_condition=None,
    ):
        self._run_community_messages_baseline(community_admin, community_member, message_count, network_condition)

    def test_multiple_community_messages(self, community_admin, community_member):
        self._run_community_messages_baseline(community_admin, community_member, message_count=50)

    @pytest.mark.parametrize("backend_factory", [{"privileged": True}], indirect=True)
    def test_community_messages_with_latency(self, community_admin, community_member):
        self._run_community_messages_baseline(community_admin, community_member, network_condition=messenger.add_latency)

    @pytest.mark.parametrize("backend_factory", [{"privileged": True}], indirect=True)
    def test_community_messages_with_packet_loss(self, community_admin, community_member):
        self._run_community_messages_baseline(community_admin, community_member, network_condition=messenger.add_packet_loss)

    @pytest.mark.parametrize("backend_factory", [{"privileged": True}], indirect=True)
    def test_community_messages_with_low_bandwidth(self, community_admin, community_member):
        self._run_community_messages_baseline(community_admin, community_member, network_condition=messenger.add_low_bandwith)

    def test_community_messages_with_node_pause_30_seconds(self, community_admin, community_member):
        community_id = messenger.create_community(community_admin)
        message_chat_id = messenger.join_community(member=community_member, admin=community_admin, community_id=community_id)

        with messenger.node_pause(community_member):
            message_text = f"test_message_{uuid4()}"
            community_admin.wakuext_service.send_chat_message(message_chat_id, message_text)
            sleep(30)
        with community_member.expect_signal(SignalType.MESSAGES_NEW, pattern=message_text):
            pass

    @pytest.mark.skipif(USE_IPV6 == "Yes", reason="Test works only with IPV4")
    def test_community_messages_with_ip_change(self, community_admin, community_member):
        community_id = messenger.create_community(community_admin)
        message_chat_id = messenger.join_community(member=community_member, admin=community_admin, community_id=community_id)

        messenger.community_messages(message_chat_id, 1, sender=community_admin, receiver=community_member)
        community_member.change_container_ip()
        messenger.community_messages(message_chat_id, 1, sender=community_admin, receiver=community_member)
