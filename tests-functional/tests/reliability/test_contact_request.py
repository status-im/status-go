from time import sleep
from uuid import uuid4
import pytest
from test_cases import MessengerTestCase
from clients.signals import SignalType
from resources.enums import MessageContentType


@pytest.mark.reliability
class TestContactRequests(MessengerTestCase):

    @pytest.mark.rpc  # until we have dedicated functional tests for this we can still run this test as part of the functional tests suite
    @pytest.mark.dependency(name="test_contact_request_baseline")
    def test_contact_request_baseline(self, execution_number=1):
        message_text = f"test_contact_request_{execution_number}_{uuid4()}"
        sender = self.initialize_backend(await_signals=self.await_signals)
        receiver = self.initialize_backend(await_signals=self.await_signals)

        existing_contacts = receiver.wakuext_service.get_contacts()

        if sender.public_key in str(existing_contacts):
            pytest.skip("Contact request was already sent for this sender<->receiver. Skipping test!!")

        response = sender.wakuext_service.send_contact_request(receiver.public_key, message_text)
        expected_message = self.get_message_by_content_type(response, content_type=MessageContentType.CONTACT_REQUEST.value)[0]

        messages_new_event = receiver.find_signal_containing_pattern(
            SignalType.MESSAGES_NEW.value,
            event_pattern=expected_message.get("id"),
            timeout=60,
        )

        signal_messages_texts = []
        if "messages" in messages_new_event.get("event", {}):
            signal_messages_texts.extend(message["text"] for message in messages_new_event["event"]["messages"] if "text" in message)

        assert (
            f"@{sender.public_key} sent you a contact request" in signal_messages_texts
        ), "Couldn't find the signal corresponding to the contact request"

        self.validate_signal_event_against_response(
            signal_event=messages_new_event,
            fields_to_validate={"text": "text"},
            expected_message=expected_message,
        )

    @pytest.mark.skip(
        reason=(
            "Skipping because of error 'Not enough status-backend containers, "
            "please add more'. Unkipping when we merge "
            "https://github.com/status-im/status-go/pull/6159"
        )
    )
    @pytest.mark.parametrize("execution_number", range(10))
    @pytest.mark.dependency(depends=["test_contact_request_baseline"])
    def test_multiple_contact_requests(self, execution_number):
        self.test_contact_request_baseline(execution_number=execution_number)

    @pytest.mark.dependency(depends=["test_contact_request_baseline"])
    @pytest.mark.skip(reason="Skipping until add_latency is implemented")
    def test_contact_request_with_latency(self):
        # with self.add_latency():
        #     self.test_contact_request_baseline()
        # to be done in the next PR
        pass

    @pytest.mark.dependency(depends=["test_contact_request_baseline"])
    @pytest.mark.skip(reason="Skipping until add_packet_loss is implemented")
    def test_contact_request_with_packet_loss(self):
        # with self.add_packet_loss():
        #     self.test_contact_request_baseline()
        # to be done in the next PR
        pass

    @pytest.mark.dependency(depends=["test_contact_request_baseline"])
    @pytest.mark.skip(reason="Skipping until add_low_bandwith is implemented")
    def test_contact_request_with_low_bandwidth(self):
        # with self.add_low_bandwith():
        #     self.test_contact_request_baseline()
        # to be done in the next PR
        pass

    @pytest.mark.dependency(depends=["test_contact_request_baseline"])
    def test_contact_request_with_node_pause_30_seconds(self):
        await_signals = [
            SignalType.MESSAGES_NEW.value,
            SignalType.MESSAGE_DELIVERED.value,
        ]
        sender = self.initialize_backend(await_signals=await_signals)
        receiver = self.initialize_backend(await_signals=await_signals)

        with self.node_pause(receiver):
            message_text = f"test_contact_request_{uuid4()}"
            response = sender.wakuext_service.send_contact_request(receiver.public_key, message_text)
            expected_message = self.get_message_by_content_type(response, content_type=MessageContentType.CONTACT_REQUEST.value)[0]
            sleep(30)
        receiver.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=expected_message.get("id"))
        sender.wait_for_signal(SignalType.MESSAGE_DELIVERED.value)
