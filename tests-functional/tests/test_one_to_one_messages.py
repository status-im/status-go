from uuid import uuid4
import pytest
from constants import *
from src.libs.common import delay
from src.libs.custom_logger import get_custom_logger
from src.steps.common import StepsCommon
from validators.message_validator import MessageValidator

logger = get_custom_logger(__name__)


@pytest.mark.usefixtures("start_2_nodes")
class TestOneToOneMessages(StepsCommon):
    def test_one_to_one_message_baseline(self):
        timeout_secs = EVENT_SIGNAL_TIMEOUT_SEC
        num_messages = NUM_MESSAGES
        nodes = [
            (self.first_node, self.second_node, "second_node_user"),
            (self.second_node, self.first_node, "first_node_user")
        ]
        messages = []
        self.accept_contact_request()

        missing_messages = []

        for i in range(num_messages):
            sending_node, receiving_node, receiving_display_name = nodes[i % 2]
            result = self.send_and_wait_for_message(
                sending_node, receiving_node, receiving_display_name, i, timeout_secs)
            timestamp, message_text, message_id, response = result

            if not response:
                missing_messages.append((timestamp, message_text, message_id, sending_node.name))
            else:
                messages.append((timestamp, message_text, message_id, sending_node.name))

        self.first_node.stop()
        self.second_node.stop()

        if missing_messages:
            formatted_missing_messages = [
                f"Timestamp: {ts}, Message: {msg}, ID: {mid}, Sender: {snd}"
                for ts, msg, mid, snd in missing_messages
            ]
            raise AssertionError(
                f"{len(missing_messages)} messages out of {num_messages} were not received: "
                + "\n".join(formatted_missing_messages)
            )

    def send_and_wait_for_message(self, sending_node, receiving_node, display_name, index, timeout=10):
        receiving_node_pubkey = receiving_node.get_pubkey(display_name)
        message_text = f"message_from_{sending_node.name}_{index}"

        timestamp, message_id, response = self.send_with_timestamp(
            sending_node.send_message, receiving_node_pubkey, message_text
        )

        validator = MessageValidator(response)
        validator.run_all_validations(
            expected_chat_id=receiving_node_pubkey,
            expected_display_name=display_name,
            expected_text=message_text
        )

        try:
            messages_new_events = receiving_node.wait_for_complete_signal("messages.new", timeout)
            receiving_node.wait_for_signal("message.delivered", timeout)

            messages_new_event = None
            for event in messages_new_events:
                if "chats" in event.get("event", {}):
                    messages_new_event = event
                    try:
                        validator.validate_event_against_response(
                            messages_new_event,
                            fields_to_validate={
                                "text": "text",
                                "displayName": "displayName",
                                "id": "id"
                            }
                        )
                        break
                    except AssertionError as validation_error:
                        logger.error(f"Validation failed for event: {messages_new_event}, Error: {validation_error}")
                        continue

            if messages_new_event is None:
                raise ValueError("No 'messages.new' event with 'chats' data found within the timeout period.")

        except (TimeoutError, ValueError) as e:
            logger.error(f"Signal validation failed: {str(e)}")
            return timestamp, message_text, message_id, None

        return timestamp, message_text, message_id, response

    def test_one_to_one_message_with_latency(self):
        with self.add_latency():
            self.test_one_to_one_message_baseline()

    def test_one_to_one_message_with_packet_loss(self):
        with self.add_packet_loss():
            self.test_one_to_one_message_baseline()

    def test_one_to_one_message_with_low_bandwidth(self):
        with self.add_low_bandwidth():
            self.test_one_to_one_message_baseline()

    def test_one_to_one_message_with_node_pause_30_seconds(self):
        self.accept_contact_request()
        with self.node_pause(self.first_node):
            message = str(uuid4())
            self.second_node.send_message(self.first_node_pubkey, message)
            delay(30)
        assert self.second_node.wait_for_signal("messages.new")
