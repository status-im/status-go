import pytest
from uuid import uuid4
from time import sleep

from steps import messenger
from clients.signals import SignalType
from resources.enums import MessageContentType
from resources.constants import USE_IPV6


@pytest.mark.reliability
class TestContactRequests:
    @pytest.fixture()
    def sender(self, backend_new_profile):
        return backend_new_profile("sender", bridge_network=True)

    @pytest.fixture()
    def receiver(self, backend_new_profile):
        return backend_new_profile("receiver", bridge_network=True)

    def _run_contact_request_baseline(self, sender, receiver, execution_number=None, network_condition=None):
        messenger.add_contact(
            sender=sender,
            receiver=receiver,
            execution_number=execution_number,
            network_condition=network_condition,
        )

    def test_contact_request_baseline(self, sender, receiver, execution_number: int = 1, network_condition=None):
        self._run_contact_request_baseline(sender, receiver, execution_number, network_condition)

    @pytest.mark.parametrize("execution_number", range(10))
    def test_multiple_contact_requests(self, sender, receiver, execution_number: int):
        self._run_contact_request_baseline(sender, receiver, execution_number)

    @pytest.mark.parametrize("backend_factory", [{"privileged": True}], indirect=True)
    @pytest.mark.parametrize("execution_number", range(10))
    def test_contact_request_with_latency(self, sender, receiver, execution_number: int, backend_factory):
        self._run_contact_request_baseline(sender, receiver, execution_number, network_condition=messenger.add_latency)

    @pytest.mark.parametrize("backend_factory", [{"privileged": True}], indirect=True)
    def test_contact_request_with_packet_loss(self, sender, receiver):
        self._run_contact_request_baseline(sender, receiver, execution_number=10, network_condition=messenger.add_packet_loss)

    @pytest.mark.parametrize("backend_factory", [{"privileged": True}], indirect=True)
    def test_contact_request_with_low_bandwidth(self, sender, receiver, backend_factory):
        self._run_contact_request_baseline(sender, receiver, execution_number=10, network_condition=messenger.add_low_bandwith)

    def test_contact_request_with_node_pause_30_seconds(self, sender, receiver):
        with messenger.node_pause(receiver):
            message_text = f"test_contact_request_{uuid4()}"
            response = sender.wakuext_service.send_contact_request(receiver.public_key, message_text)
            expected_message = messenger.get_message_by_content_type(response, content_type=MessageContentType.CONTACT_REQUEST.value)[0]
            sleep(30)
        with receiver.expect_signal(
            SignalType.MESSAGES_NEW,
            pattern=expected_message.get("id"),
            start="beginning",
            timeout=60,
        ):
            pass
        with sender.expect_signal(SignalType.MESSAGE_DELIVERED, start="beginning", timeout=60):
            pass

    @pytest.mark.skipif(USE_IPV6 == "Yes", reason="Test works only with IPV4")
    def test_contact_request_with_ip_change(self, sender, receiver):
        receiver.change_container_ip()
        message_text = f"test_contact_request_{uuid4()}"
        response = sender.wakuext_service.send_contact_request(receiver.public_key, message_text)
        expected_message = messenger.get_message_by_content_type(response, content_type=MessageContentType.CONTACT_REQUEST.value)[0]
        with receiver.expect_signal(
            SignalType.MESSAGES_NEW,
            pattern=expected_message.get("id"),
            start="beginning",
            timeout=60,
        ):
            pass
