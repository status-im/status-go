import logging
from uuid import uuid4
import pytest
from constants import *
from src.libs.common import delay
from src.steps.common import StepsCommon
from validators.message_validator import MessageValidator


@pytest.mark.usefixtures("start_2_nodes")
class TestOneToOneMessages(StepsCommon):
    def test_one_to_one_message_baseline(self):
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
            receiving_node_pubkey = receiving_node.get_pubkey(receiving_display_name)

            message_text = f"message_from_{sending_node.name}_{i}"
            timestamp, message_id, response = self.send_with_timestamp(
                sending_node.send_message, receiving_node_pubkey, message_text
            )
            messages.append((timestamp, message_text, message_id, sending_node.name))

            if response:
                validator = MessageValidator(response)
                validator.validate_response(
                    expected_chat_id=receiving_node_pubkey,
                    expected_message=message_text,
                    expected_sender=f"{sending_node.name}_user"
                )
            else:
                logging.error(f"Response for message {message_text} was empty or invalid")

            try:
                receiving_node.wait_for_signal("messages.new", None)
            except TimeoutError:
                error_message = f"Signal for message {message_text} from {sending_node.name} was not received by {receiving_node.name}"
                logging.error(error_message)
                missing_messages.append((timestamp, message_text, message_id, sending_node.name))

            delay(DELAY_BETWEEN_MESSAGES)

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
