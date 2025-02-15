from time import sleep
from uuid import uuid4
import pytest
from tests.test_cases import MessengerTestCase
from clients.signals import SignalType
from resources.enums import MessageContentType


@pytest.mark.reliability
class TestContactRequests(MessengerTestCase):

    def test_contact_request_baseline(self, execution_number=1, network_condition=None):
        self.add_contact(execution_number, network_condition)

    @pytest.mark.parametrize("execution_number", range(10))
    def test_multiple_contact_requests(self, execution_number):
        self.test_contact_request_baseline(execution_number=execution_number)

    @pytest.mark.parametrize("execution_number", range(10))
    def test_contact_request_with_latency(self, execution_number):
        self.test_contact_request_baseline(execution_number=execution_number, network_condition=self.add_latency)

    def test_contact_request_with_packet_loss(self):
        self.test_contact_request_baseline(execution_number=10, network_condition=self.add_packet_loss)

    def test_contact_request_with_low_bandwidth(self):
        self.test_contact_request_baseline(execution_number=10, network_condition=self.add_low_bandwith)

    def test_contact_request_with_node_pause_30_seconds(self):
        sender = self.initialize_backend(await_signals=self.await_signals)
        receiver = self.initialize_backend(await_signals=self.await_signals)

        with self.node_pause(receiver):
            message_text = f"test_contact_request_{uuid4()}"
            response = sender.wakuext_service.send_contact_request(receiver.public_key, message_text)
            expected_message = self.get_message_by_content_type(response, content_type=MessageContentType.CONTACT_REQUEST.value)[0]
            sleep(30)
        receiver.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=expected_message.get("id"))
        sender.wait_for_signal(SignalType.MESSAGE_DELIVERED.value)

    def test_contact_request_with_ip_change(self):
        sender = self.initialize_backend(await_signals=self.await_signals, privileged=True)
        receiver = self.initialize_backend(await_signals=self.await_signals, privileged=True)
        receiver.change_container_ip()

        message_text = f"test_contact_request_{uuid4()}"
        response = sender.wakuext_service.send_contact_request(receiver.public_key, message_text)
        expected_message = self.get_message_by_content_type(response, content_type=MessageContentType.CONTACT_REQUEST.value)[0]
        receiver.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=expected_message.get("id"))
