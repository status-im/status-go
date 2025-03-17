from time import sleep
from uuid import uuid4
import pytest
from steps.messenger import MessengerSteps
from clients.signals import SignalType
from resources.constants import USE_IPV6


@pytest.mark.usefixtures("setup_two_privileged_nodes")
@pytest.mark.reliability
class TestOneToOneMessages(MessengerSteps):

    def test_one_to_one_message_baseline(self, message_count=1):
        self.one_to_one_message(message_count)

    def test_multiple_one_to_one_messages(self):
        self.test_one_to_one_message_baseline(message_count=50)

    def test_one_to_one_message_with_latency(self):
        with self.add_latency(self.receiver):
            self.test_one_to_one_message_baseline(message_count=50)

    def test_one_to_one_message_with_packet_loss(self):
        with self.add_packet_loss(self.receiver):
            self.test_one_to_one_message_baseline(message_count=50)

    def test_one_to_one_message_with_low_bandwidth(self):
        with self.add_low_bandwith(self.receiver):
            self.test_one_to_one_message_baseline(message_count=50)

    def test_one_to_one_message_with_node_pause_30_seconds(self):
        with self.node_pause(self.receiver):
            message_text = f"test_message_{uuid4()}"
            self.sender.wakuext_service.send_one_to_one_message(self.receiver.public_key, message_text)
            sleep(30)
        self.receiver.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=message_text)
        self.sender.wait_for_signal(SignalType.MESSAGE_DELIVERED.value)

    @pytest.mark.skipif(USE_IPV6 == "Yes", reason="Test works only with IPV4")
    def test_one_to_one_messages_with_ip_change(self):
        self.test_one_to_one_message_baseline()
        self.receiver.change_container_ip()
        self.test_one_to_one_message_baseline()
