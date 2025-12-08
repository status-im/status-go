from time import sleep
from uuid import uuid4
import pytest
from steps.messenger import MessengerSteps
from clients.signals import SignalType
from resources.constants import USE_IPV6


@pytest.mark.reliability
class TestCreatePrivateGroups(MessengerSteps):

    @pytest.fixture()
    def community_admin(self, backend_new_profile):
        return backend_new_profile("community_admin", bridge_network=True)

    @pytest.fixture()
    def community_member(self, backend_new_profile):
        return backend_new_profile("member", bridge_network=True)

    def _run_create_private_group_baseline(self, community_admin, community_member, private_groups_count=1):
        self.make_contacts(community_admin, community_member)
        self.create_private_group(private_groups_count, admin=community_admin, member=community_member)

    def test_create_private_group_baseline(self, community_admin, community_member, private_groups_count=1):
        self._run_create_private_group_baseline(community_admin, community_member, private_groups_count)

    def test_multiple_create_private_groups(self, community_admin, community_member):
        self._run_create_private_group_baseline(community_admin, community_member, private_groups_count=50)

    def test_create_private_groups_with_node_pause_30_seconds(self, community_admin, community_member):
        self.make_contacts(community_admin, community_member)

        with self.node_pause(community_member):
            private_group_name = f"private_group_{uuid4()}"
            community_admin.wakuext_service.create_group_chat_with_members(
                [community_member.public_key],
                private_group_name,
            )
            sleep(30)
        community_member.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=private_group_name)

    @pytest.mark.skipif(USE_IPV6 == "Yes", reason="Test works only with IPV4")
    def test_create_private_groups_with_ip_change(self, community_admin, community_member):
        self.make_contacts(community_admin, community_member)
        community_member.change_container_ip()

        private_group_name = f"private_group_{uuid4()}"
        community_admin.wakuext_service.create_group_chat_with_members([community_member.public_key], private_group_name)
        community_member.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=private_group_name)
