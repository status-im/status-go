from time import sleep
from uuid import uuid4
import pytest
from steps.messenger import MessengerSteps
from clients.signals import SignalType
from resources.constants import USE_IPV6


@pytest.mark.reliability
class TestCreatePrivateGroups(MessengerSteps):

    @pytest.fixture()
    def admin(self, backend_new_profile):
        return backend_new_profile("admin", bridge_network=True)

    @pytest.fixture()
    def member(self, backend_new_profile):
        return backend_new_profile("member", bridge_network=True)

    def _run_create_private_group_baseline(self, admin, member, private_groups_count=1):
        self.make_contacts(admin, member)
        self.create_private_group(private_groups_count, admin=admin, member=member)

    def test_create_private_group_baseline(self, admin, member, private_groups_count=1):
        self._run_create_private_group_baseline(admin, member, private_groups_count)

    def test_multiple_create_private_groups(self, admin, member):
        self._run_create_private_group_baseline(admin, member, private_groups_count=50)

    def test_create_private_groups_with_node_pause_30_seconds(self, admin, member):
        self.make_contacts(admin, member)

        with self.node_pause(member):
            private_group_name = f"private_group_{uuid4()}"
            admin.wakuext_service.create_group_chat_with_members([member.public_key], private_group_name)
            sleep(30)
        member.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=private_group_name)

    @pytest.mark.skipif(USE_IPV6 == "Yes", reason="Test works only with IPV4")
    def test_create_private_groups_with_ip_change(self, admin, member):
        self.make_contacts(admin, member)
        member.change_container_ip()

        private_group_name = f"private_group_{uuid4()}"
        admin.wakuext_service.create_group_chat_with_members([member.public_key], private_group_name)
        member.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=private_group_name)
