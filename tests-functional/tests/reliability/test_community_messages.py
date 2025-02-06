from time import sleep
from uuid import uuid4
import pytest
from tests.test_cases import MessengerTestCase
from clients.signals import SignalType


@pytest.mark.usefixtures("setup_two_nodes")
@pytest.mark.reliability
class TestCommunityMessages(MessengerTestCase):

    def test_community_messages_baseline(self, message_count=1, network_condition=None):
        message_chat_id = self.create_and_join_community()
        if network_condition:
            network_condition(self.receiver)
        self.community_messages(message_chat_id, message_count)

    def test_multiple_community_messages(self):
        self.test_community_messages_baseline(message_count=50)

    def test_community_messages_with_latency(self):
        self.test_community_messages_baseline(network_condition=self.add_latency)

    def test_community_messages_with_packet_loss(self):
        self.test_community_messages_baseline(network_condition=self.add_packet_loss)

    def test_community_messages_with_low_bandwidth(self):
        self.test_community_messages_baseline(network_condition=self.add_low_bandwith)

    def test_community_messages_with_node_pause_30_seconds(self):
        message_chat_id = self.create_and_join_community()

        with self.node_pause(self.receiver):
            message_text = f"test_message_{uuid4()}"
            self.sender.wakuext_service.send_community_chat_message(message_chat_id, message_text)
            sleep(30)
        self.receiver.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=message_text)
