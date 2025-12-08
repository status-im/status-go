from time import sleep
from uuid import uuid4
import pytest
from steps.messenger import MessengerSteps
from clients.signals import SignalType
from resources.constants import USE_IPV6


@pytest.mark.reliability
class TestPrivateGroupMessages(MessengerSteps):

    @pytest.fixture()
    def community_admin(self, backend_new_profile):
        return backend_new_profile("community_admin", bridge_network=True)

    @pytest.fixture()
    def community_member(self, backend_new_profile):
        return backend_new_profile("community_member", bridge_network=True)

    def _run_private_group_messages_baseline(self, community_admin, community_member, message_count=1):
        self.make_contacts(community_admin, community_member)
        private_group_id = self.join_private_group(admin=community_admin, member=community_member)
        self.private_group_message(message_count, private_group_id, sender=community_admin, receiver=community_member)

    def test_private_group_messages_baseline(self, community_admin, community_member, message_count=1):
        self._run_private_group_messages_baseline(community_admin, community_member, message_count)

    def test_multiple_group_chat_messages(self, community_admin, community_member):
        self._run_private_group_messages_baseline(community_admin, community_member, message_count=50)

    @pytest.mark.parametrize("backend_factory", [{"privileged": True}], indirect=True)
    def test_private_group_chat_messages_with_latency(self, community_admin, community_member):
        with self.add_latency(community_member):
            self._run_private_group_messages_baseline(community_admin, community_member, message_count=50)

    @pytest.mark.parametrize("backend_factory", [{"privileged": True}], indirect=True)
    def test_private_group_chat_messages_with_packet_loss(self, community_admin, community_member):
        with self.add_packet_loss(community_member):
            self._run_private_group_messages_baseline(community_admin, community_member, message_count=50)

    @pytest.mark.parametrize("backend_factory", [{"privileged": True}], indirect=True)
    def test_private_group_chat_messages_with_low_bandwidth(self, community_admin, community_member):
        with self.add_low_bandwith(community_member):
            self._run_private_group_messages_baseline(community_admin, community_member, message_count=50)

    def test_private_group_messages_with_node_pause_30_seconds(self, community_admin, community_member):
        self.make_contacts(community_admin, community_member)
        private_group_id = self.join_private_group(admin=community_admin, member=community_member)

        with self.node_pause(community_member):
            message_text = f"test_message_{uuid4()}"
            community_admin.wakuext_service.send_group_chat_message(private_group_id, message_text)
            sleep(30)
        community_member.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=message_text)
        community_admin.wait_for_signal(SignalType.MESSAGE_DELIVERED.value)

    @pytest.mark.skipif(USE_IPV6 == "Yes", reason="Test works only with IPV4")
    def test_private_group_messages_with_ip_change(self, community_admin, community_member):
        self.make_contacts(community_admin, community_member)
        private_group_id = self.join_private_group(admin=community_admin, member=community_member)
        self.private_group_message(1, private_group_id, sender=community_admin, receiver=community_member)
        community_member.change_container_ip()
        self.private_group_message(1, private_group_id, sender=community_admin, receiver=community_member)
