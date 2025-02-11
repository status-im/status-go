from time import sleep
from uuid import uuid4
import pytest
from tests.test_cases import MessengerTestCase
from clients.signals import SignalType


@pytest.mark.usefixtures("setup_two_nodes")
@pytest.mark.reliability
class TestPrivateGroupMessages(MessengerTestCase):

    def test_private_group_messages_baseline(self, message_count=1):
        self.make_contacts()
        self.private_group_id = self.join_private_group()
        self.private_group_message(message_count, self.private_group_id)

    def test_multiple_group_chat_messages(self):
        self.test_private_group_messages_baseline(message_count=50)

    def test_multiple_group_chat_messages_with_latency(self):
        with self.add_latency(self.receiver):
            self.test_private_group_messages_baseline(message_count=50)

    def test_multiple_group_chat_messages_with_packet_loss(self):
        with self.add_packet_loss(self.receiver):
            self.test_private_group_messages_baseline(message_count=50)

    def test_multiple_group_chat_messages_with_low_bandwidth(self):
        with self.add_low_bandwith(self.receiver):
            self.test_private_group_messages_baseline(message_count=50)

    def test_private_group_messages_with_node_pause_30_seconds(self):
        self.make_contacts()
        self.private_group_id = self.join_private_group()

        with self.node_pause(self.receiver):
            message_text = f"test_message_{uuid4()}"
            self.sender.wakuext_service.send_group_chat_message(self.private_group_id, message_text)
            sleep(30)
        self.receiver.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=message_text)
        self.sender.wait_for_signal(SignalType.MESSAGE_DELIVERED.value)
