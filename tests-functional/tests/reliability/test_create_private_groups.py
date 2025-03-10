from time import sleep
from uuid import uuid4
import pytest
from steps.messenger import MessengerSteps
from clients.signals import SignalType
from resources.constants import USE_IPV6


@pytest.mark.usefixtures("setup_two_privileged_nodes")
@pytest.mark.reliability
class TestCreatePrivateGroups(MessengerSteps):

    def test_create_private_group_baseline(self, private_groups_count=1):
        self.make_contacts()
        self.create_private_group(private_groups_count)

    def test_multiple_create_private_groups(self):
        self.test_create_private_group_baseline(private_groups_count=50)

    def test_create_private_groups_with_node_pause_30_seconds(self):
        self.make_contacts()

        with self.node_pause(self.receiver):
            private_group_name = f"private_group_{uuid4()}"
            self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], private_group_name)
            sleep(30)
        self.receiver.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=private_group_name)

    @pytest.mark.skipif(USE_IPV6 == "Yes", reason="Test works only with IPV4")
    def test_create_private_groups_with_ip_change(self):
        self.make_contacts()
        self.receiver.change_container_ip()

        private_group_name = f"private_group_{uuid4()}"
        self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], private_group_name)
        self.receiver.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=private_group_name)
