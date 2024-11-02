from uuid import uuid4
from constants import *
from src.libs.common import delay
from src.libs.custom_logger import get_custom_logger
from src.node.status_node import StatusNode
from src.steps.common import StepsCommon
from src.libs.common import create_unique_data_dir, get_project_root
from validators.contact_request_validator import ContactRequestValidator

logger = get_custom_logger(__name__)


class TestContactRequest(StepsCommon):
    def test_contact_request_baseline(self):
        timeout_secs = 3
        num_contact_requests = NUM_CONTACT_REQUESTS
        project_root = get_project_root()
        nodes = []
        LOCAL_DATA = "tests-functional/local"

        for index in range(num_contact_requests):
            first_node = StatusNode(name=f"first_node_{index}")
            second_node = StatusNode(name=f"second_node_{index}")

            data_dir_first = create_unique_data_dir(os.path.join(project_root, LOCAL_DATA), index)
            data_dir_second = create_unique_data_dir(os.path.join(project_root, LOCAL_DATA), index)

            delay(2)
            first_node.start(data_dir=data_dir_first)
            second_node.start(data_dir=data_dir_second)

            account_data_first = {
                "rootDataDir": data_dir_first,
                "displayName": f"test_user_first_{index}",
                "password": f"test_password_first_{index}",
                "customizationColor": "primary"
            }
            account_data_second = {
                "rootDataDir": data_dir_second,
                "displayName": f"test_user_second_{index}",
                "password": f"test_password_second_{index}",
                "customizationColor": "primary"
            }
            first_node.create_account_and_login(account_data_first)
            second_node.create_account_and_login(account_data_second)

            delay(5)
            first_node.start_messenger()
            second_node.start_messenger()

            first_node.pubkey = first_node.get_pubkey(account_data_first["displayName"])
            second_node.pubkey = second_node.get_pubkey(account_data_second["displayName"])

            first_node.wait_fully_started()
            second_node.wait_fully_started()

            nodes.append((first_node, second_node, account_data_first["displayName"], index))

        missing_contact_requests = []
        for first_node, second_node, display_name, index in nodes:
            result = self.send_and_wait_for_message((first_node, second_node), display_name, index, timeout_secs)
            timestamp, message_id, contact_request_message, response = result

            if not response:
                missing_contact_requests.append((timestamp, contact_request_message, message_id))
            else:
                validator = ContactRequestValidator(response)
                validator.run_all_validations(
                    expected_chat_id=first_node.pubkey,
                    expected_display_name=display_name,
                    expected_text=f"contact_request_{index}"
                )

        if missing_contact_requests:
            formatted_missing_requests = [
                f"Timestamp: {ts}, Message: {msg}, ID: {mid}" for ts, msg, mid in missing_contact_requests
            ]
            raise AssertionError(
                f"{len(missing_contact_requests)} contact requests out of {num_contact_requests} didn't reach the peer node: "
                + "\n".join(formatted_missing_requests)
            )

    def send_and_wait_for_message(self, nodes, display_name, index, timeout=10):
        first_node, second_node = nodes
        first_node_pubkey = first_node.get_pubkey(display_name)
        contact_request_message = f"contact_request_{index}"

        timestamp, message_id, response = self.send_with_timestamp(
            second_node.send_contact_request, first_node_pubkey, contact_request_message
        )

        validator = ContactRequestValidator(response)
        validator.run_all_validations(first_node_pubkey, display_name, contact_request_message)

        try:
            first_node.wait_for_signal("history.request.started", timeout)
            messages_new_events = first_node.wait_for_complete_signal("messages.new", timeout)

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
            first_node.wait_for_signal("history.request.completed", timeout)

        except (TimeoutError, ValueError) as e:
            logger.error(f"Signal validation failed: {str(e)}")
            return timestamp, message_id, contact_request_message, None

        first_node.stop()
        second_node.stop()

        return timestamp, message_id, contact_request_message, response

    def test_contact_request_with_latency(self):
        with self.add_latency():
            self.test_contact_request_baseline()

    def test_contact_request_with_packet_loss(self):
        with self.add_packet_loss():
            self.test_contact_request_baseline()

    def test_contact_request_with_low_bandwidth(self):
        with self.add_low_bandwidth():
            self.test_contact_request_baseline()

    def test_contact_request_with_node_pause(self, start_2_nodes):
        with self.node_pause(self.second_node):
            message = str(uuid4())
            self.first_node.send_contact_request(self.second_node_pubkey, message)
            delay(10)
        assert self.second_node.wait_for_complete_signal("history.request.completed")
