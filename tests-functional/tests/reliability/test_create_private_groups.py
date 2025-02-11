from time import sleep
from uuid import uuid4
import pytest
from tests.test_cases import MessengerTestCase
from clients.signals import SignalType


@pytest.mark.usefixtures("setup_two_privileged_nodes")
@pytest.mark.reliability
class TestCreatePrivateGroups(MessengerTestCase):

    def test_create_private_group_baseline(self, private_groups_count=1):
        self.make_contacts()
        self.create_private_group(private_groups_count)

    def test_multiple_one_create_private_groups(self):
        self.test_create_private_group_baseline(private_groups_count=50)

    def test_create_private_groups_with_node_pause_30_seconds(self):
        self.make_contacts()

        with self.node_pause(self.receiver):
            private_group_name = f"private_group_{uuid4()}"
            self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], private_group_name)
            sleep(30)
        self.receiver.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=private_group_name)
