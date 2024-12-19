from time import sleep
from uuid import uuid4
import pytest
from test_cases import MessengerTestCase
from clients.signals import SignalType
from resources.enums import MessageContentType


@pytest.mark.usefixtures("setup_two_nodes")
@pytest.mark.reliability
class TestOneToOneMessages(MessengerTestCase):

    @pytest.mark.rpc  # until we have dedicated functional tests for this we can still run this test as part of the functional tests suite
    @pytest.mark.dependency(name="test_one_to_one_message_baseline")
    def test_one_to_one_message_baseline(self, message_count=1):
        sent_messages = []
        for i in range(message_count):
            message_text = f"test_message_{i+1}_{uuid4()}"
            response = self.sender.wakuext_service.send_message(self.receiver.public_key, message_text)
            expected_message = self.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
            sent_messages.append(expected_message)
            sleep(0.01)

        for i, expected_message in enumerate(sent_messages):
            messages_new_event = self.receiver.find_signal_containing_pattern(
                SignalType.MESSAGES_NEW.value,
                event_pattern=expected_message.get("id"),
                timeout=60,
            )
            self.validate_signal_event_against_response(
                signal_event=messages_new_event,
                fields_to_validate={"text": "text"},
                expected_message=expected_message,
            )

    @pytest.mark.dependency(depends=["test_one_to_one_message_baseline"])
    def test_multiple_one_to_one_messages(self):
        self.test_one_to_one_message_baseline(message_count=50)

    @pytest.mark.dependency(depends=["test_one_to_one_message_baseline"])
    @pytest.mark.skip(reason="Skipping until add_latency is implemented")
    def test_one_to_one_message_with_latency(self):
        # with self.add_latency():
        #     self.test_one_to_one_message_baseline()
        # to be done in the next PR
        pass

    @pytest.mark.dependency(depends=["test_one_to_one_message_baseline"])
    @pytest.mark.skip(reason="Skipping until add_packet_loss is implemented")
    def test_one_to_one_message_with_packet_loss(self):
        # with self.add_packet_loss():
        #     self.test_one_to_one_message_baseline()
        # to be done in the next PR
        pass

    @pytest.mark.dependency(depends=["test_one_to_one_message_baseline"])
    @pytest.mark.skip(reason="Skipping until add_low_bandwith is implemented")
    def test_one_to_one_message_with_low_bandwidth(self):
        # with self.add_low_bandwith():
        #     self.test_one_to_one_message_baseline()
        # to be done in the next PR
        pass

    @pytest.mark.dependency(depends=["test_one_to_one_message_baseline"])
    def test_one_to_one_message_with_node_pause_30_seconds(self):
        with self.node_pause(self.receiver):
            message_text = f"test_message_{uuid4()}"
            self.sender.wakuext_service.send_message(self.receiver.public_key, message_text)
            sleep(30)
        self.receiver.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=message_text)
        self.sender.wait_for_signal(SignalType.MESSAGE_DELIVERED.value)
