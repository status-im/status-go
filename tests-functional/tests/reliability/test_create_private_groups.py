from time import sleep
from uuid import uuid4
import pytest
from tests.test_cases import MessengerTestCase
from clients.signals import SignalType
from resources.enums import MessageContentType


@pytest.mark.usefixtures("setup_two_nodes")
@pytest.mark.reliability
class TestCreatePrivateGroups(MessengerTestCase):

    def test_create_private_group_baseline(self, private_groups_count=1):
        self.make_contacts()

        private_groups = []
        for i in range(private_groups_count):
            private_group_name = f"private_group_{i+1}_{uuid4()}"
            response = self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], private_group_name)

            expected_group_creation_msg = f"@{self.sender.public_key} created the group {private_group_name}"
            expected_message = self.get_message_by_content_type(
                response, content_type=MessageContentType.SYSTEM_MESSAGE_CONTENT_PRIVATE_GROUP.value, message_pattern=expected_group_creation_msg
            )[0]

            private_groups.append(expected_message)
            sleep(0.01)

        for i, expected_message in enumerate(private_groups):
            messages_new_event = self.receiver.find_signal_containing_pattern(
                SignalType.MESSAGES_NEW.value, event_pattern=expected_message.get("id"), timeout=60
            )
            self.validate_signal_event_against_response(
                signal_event=messages_new_event,
                expected_message=expected_message,
                fields_to_validate={"text": "text"},
            )

    def test_multiple_one_create_private_groups(self):
        self.test_create_private_group_baseline(private_groups_count=50)

    def test_create_private_groups_with_node_pause_30_seconds(self):
        self.make_contacts()

        with self.node_pause(self.receiver):
            private_group_name = f"private_group_{uuid4()}"
            self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], private_group_name)
            sleep(30)
        self.receiver.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=private_group_name)
