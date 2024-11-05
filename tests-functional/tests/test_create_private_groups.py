import pytest
from src.libs.common import delay
from src.libs.custom_logger import get_custom_logger
from src.steps.common import StepsCommon
from constants import *
from validators.contact_request_validator import ContactRequestValidator
from validators.group_chat_validator import GroupChatValidator

logger = get_custom_logger(__name__)


@pytest.mark.usefixtures("start_2_nodes")
class TestCreatePrivateGroups(StepsCommon):
    def test_create_group_chat_baseline(self):
        num_private_groups = PRIVATE_GROUPS
        private_groups = []
        contact_request_sent = False

        for i in range(num_private_groups):
            if i % 2 == 0:
                sender_node = self.first_node
                receiver_node = self.second_node
                receiver_pubkey = self.second_node_pubkey
            else:
                sender_node = self.second_node
                receiver_node = self.first_node
                receiver_pubkey = self.first_node_pubkey

            if not contact_request_sent:
                display_name = f"{receiver_node.name}_user"
                contact_request_message = f"contact_request_{i}"
                timestamp, message_id, contact_request_message, response = self.send_and_wait_for_message(
                    (sender_node, receiver_node), display_name, i
                )

                if not response:
                    raise AssertionError(f"Contact request failed between {sender_node.name} and {receiver_node.name}")

                chat_id = response["result"]["chats"][0]["lastMessage"]["id"]
                accept_response = receiver_node.accept_contact_request(chat_id)

                if not accept_response:
                    raise AssertionError(
                        f"Failed to accept contact request on {receiver_node.name} for chatId {chat_id}")

                contact_request_sent = True
                delay(10)

            group_name = f"private_group_from_{sender_node.name}_{i}"
            try:
                logger.info(f"Creating group '{group_name}' from {sender_node.name}")
                timestamp, message_id, response = self.create_and_validate_private_group(
                    sender_node, [receiver_pubkey], group_name, timeout=30
                )

                if not response:
                    raise AssertionError(f"Failed to create private group '{group_name}' from {sender_node.name}")
                else:
                    logger.info(f"Private group '{group_name}' created successfully with message ID: {message_id}")
                    private_groups.append((timestamp, group_name, message_id, sender_node.name))

            except AssertionError as e:
                logger.info(f"Group creation validation failed: {e}")

            delay(5)

        missing_private_groups = [
            (ts, name, mid, node) for ts, name, mid, node in private_groups if mid is None
        ]

        if missing_private_groups:
            formatted_missing_groups = [
                f"Timestamp: {ts}, GroupName: {msg}, ID: {mid}, Node: {node}" for ts, msg, mid, node in
                missing_private_groups
            ]
            raise AssertionError(
                f"{len(missing_private_groups)} private groups out of {num_private_groups} were not created: " +
                "\n".join(formatted_missing_groups)
            )
        self.first_node.stop()
        self.second_node.stop()

    def send_and_wait_for_message(self, nodes, display_name, index, timeout=10):
        sender_node, receiver_node = nodes

        receiver_pubkey = receiver_node.get_pubkey(display_name)
        contact_request_message = f"contact_request_{index}"

        timestamp, message_id, response = self.send_with_timestamp(
            sender_node.send_contact_request, receiver_pubkey, contact_request_message
        )

        validator = ContactRequestValidator(response)
        validator.run_all_validations(receiver_pubkey, display_name, contact_request_message)

        try:
            receiver_node.wait_for_signal("history.request.started", timeout)

            messages_new_events = receiver_node.wait_for_complete_signal("messages.new", timeout)
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

            receiver_node.wait_for_signal("history.request.completed", timeout)

        except (TimeoutError, ValueError) as e:
            logger.error(f"Signal validation failed: {str(e)}")
            return timestamp, message_id, contact_request_message, None

        return timestamp, message_id, contact_request_message, response

    def create_and_validate_private_group(self, node, members_pubkeys, group_name, timeout=10):
        timestamp, message_id, response = self.send_with_timestamp_for_group(
            node.create_group_chat_with_members, members_pubkeys, group_name
        )

        if not response or "result" not in response or "chats" not in response["result"]:
            raise AssertionError("Invalid response structure. Expected 'result' with 'chats' list.")

        chat_data = response["result"]["chats"][0]
        validator = GroupChatValidator(chat_data)

        try:
            validator.validate_fields(
                {
                    "name": group_name,
                    "active": True,
                    "chatType": 3,
                    "members": [
                        {"id": pubkey} for pubkey in members_pubkeys
                    ]
                }
            )
        except AssertionError as validation_error:
            raise AssertionError(f"Validation failed for group chat creation: {validation_error}")

        try:
            node.wait_for_signal("history.request.completed", timeout)
        except TimeoutError:
            logger.error("Timeout waiting for group chat creation events.")
            return timestamp, message_id, None

        return timestamp, message_id, response

    def test_create_group_chat_with_latency(self):
        with self.add_latency():
            self.test_create_group_chat_baseline()

    def test_create_group_chat_with_packet_loss(self):
        with self.add_packet_loss():
            self.test_create_group_chat_baseline()

    def test_create_group_chat_with_low_bandwidth(self):
        with self.add_low_bandwidth():
            self.test_create_group_chat_baseline()

    def test_create_group_with_node_pause(self):
        with self.node_pause(self.first_node):
            delay(10)
            try:
                self.test_create_group_chat_baseline()
            except Exception as e:
                logger.info(f"Expected exception occurred while node was paused: {e}")
                assert "Read timed out" in str(e) or "ConnectionError" in str(
                    e), "Unexpected error type when node is paused"
