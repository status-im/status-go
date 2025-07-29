from time import sleep
from uuid import uuid4
import pytest
from steps.messenger import MessengerSteps
from clients.signals import SignalType
from resources.enums import MessageContentType
from resources.constants import USE_IPV6


@pytest.mark.reliability
class TestContactRequests(MessengerSteps):

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        """Initialize two unprivileged backends (sender and receiver) for each test function"""
        self.sender = backend_new_profile("sender")
        self.receiver = backend_new_profile("receiver")

    def test_contact_request_baseline(self, execution_number=1, network_condition=None):
        self.add_contact(sender=self.sender, receiver=self.receiver, execution_number=execution_number, network_condition=network_condition)

    @pytest.mark.parametrize("execution_number", range(10))
    def test_multiple_contact_requests(self, execution_number):
        self.test_contact_request_baseline(execution_number=execution_number)

    @pytest.mark.parametrize("backend_factory", [{"privileged": True}], indirect=True)
    @pytest.mark.parametrize("execution_number", range(10))
    def test_contact_request_with_latency(self, execution_number):
        self.test_contact_request_baseline(execution_number=execution_number, network_condition=self.add_latency)

    @pytest.mark.parametrize("backend_factory", [{"privileged": True}], indirect=True)
    def test_contact_request_with_packet_loss(self):
        self.test_contact_request_baseline(execution_number=10, network_condition=self.add_packet_loss)

    @pytest.mark.parametrize("backend_factory", [{"privileged": True}], indirect=True)
    def test_contact_request_with_low_bandwidth(self):
        self.test_contact_request_baseline(execution_number=10, network_condition=self.add_low_bandwith)

    def test_contact_request_with_node_pause_30_seconds(self):
        with self.node_pause(self.receiver):
            message_text = f"test_contact_request_{uuid4()}"
            response = self.sender.wakuext_service.send_contact_request(self.receiver.public_key, message_text)
            expected_message = self.get_message_by_content_type(response, content_type=MessageContentType.CONTACT_REQUEST.value)[0]
            sleep(30)
        self.receiver.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=expected_message.get("id"))
        self.sender.wait_for_signal(SignalType.MESSAGE_DELIVERED.value)

    @pytest.mark.skipif(USE_IPV6 == "Yes", reason="Test works only with IPV4")
    def test_contact_request_with_ip_change(self):
        self.receiver.change_container_ip()

        message_text = f"test_contact_request_{uuid4()}"
        response = self.sender.wakuext_service.send_contact_request(self.receiver.public_key, message_text)
        expected_message = self.get_message_by_content_type(response, content_type=MessageContentType.CONTACT_REQUEST.value)[0]
        self.receiver.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=expected_message.get("id"))
