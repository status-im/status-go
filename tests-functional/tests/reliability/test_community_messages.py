from time import sleep
from uuid import uuid4
import pytest
from steps.messenger import MessengerSteps
from clients.signals import SignalType
from resources.constants import USE_IPV6


@pytest.mark.reliability
class TestCommunityMessages(MessengerSteps):

    @pytest.fixture()
    def admin(self, backend_new_profile):
        return backend_new_profile("admin", bridge_network=True)

    @pytest.fixture()
    def member(self, backend_new_profile):
        return backend_new_profile("member", bridge_network=True)

    def _run_community_messages_baseline(self, admin, member, message_count=1, network_condition=None):
        self.create_community(admin)
        message_chat_id = self.join_community(member=member, admin=admin)
        if network_condition:
            network_condition(member)
        self.community_messages(message_chat_id, message_count, sender=admin, receiver=member)

    def test_community_messages_baseline(self, admin, member, message_count=1, network_condition=None):
        self._run_community_messages_baseline(admin, member, message_count, network_condition)

    def test_multiple_community_messages(self, admin, member):
        self._run_community_messages_baseline(admin, member, message_count=50)

    @pytest.mark.parametrize("backend_factory", [{"privileged": True}], indirect=True)
    def test_community_messages_with_latency(self, admin, member):
        self._run_community_messages_baseline(admin, member, network_condition=self.add_latency)

    @pytest.mark.parametrize("backend_factory", [{"privileged": True}], indirect=True)
    def test_community_messages_with_packet_loss(self, admin, member):
        self._run_community_messages_baseline(admin, member, network_condition=self.add_packet_loss)

    @pytest.mark.parametrize("backend_factory", [{"privileged": True}], indirect=True)
    def test_community_messages_with_low_bandwidth(self, admin, member):
        self._run_community_messages_baseline(admin, member, network_condition=self.add_low_bandwith)

    def test_community_messages_with_node_pause_30_seconds(self, admin, member):
        self.create_community(admin)
        message_chat_id = self.join_community(member=member, admin=admin)

        with self.node_pause(member):
            message_text = f"test_message_{uuid4()}"
            admin.wakuext_service.send_chat_message(message_chat_id, message_text)
            sleep(30)
        member.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=message_text)

    @pytest.mark.skipif(USE_IPV6 == "Yes", reason="Test works only with IPV4")
    def test_community_messages_with_ip_change(self, admin, member):
        self.create_community(admin)
        message_chat_id = self.join_community(member=member, admin=admin)

        self.community_messages(message_chat_id, 1, sender=admin, receiver=member)
        member.change_container_ip()
        self.community_messages(message_chat_id, 1, sender=admin, receiver=member)
