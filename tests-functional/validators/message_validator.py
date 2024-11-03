from src.libs.custom_logger import get_custom_logger

logger = get_custom_logger(__name__)


class MessageValidator:
    def __init__(self, response):
        self.response = response

    def validate_response_structure(self):
        assert self.response.get("jsonrpc") == "2.0", "Invalid JSON-RPC version"
        assert "result" in self.response, "Missing 'result' in response"

    def validate_chat_data(self, expected_chat_id, expected_display_name, expected_text):
        chats = self.response["result"].get("chats", [])
        assert len(chats) > 0, "No chats found in the response"

        chat = chats[0]
        actual_chat_id = chat.get("id")
        assert actual_chat_id == expected_chat_id, (
            f"Chat ID mismatch: Expected '{expected_chat_id}', found '{actual_chat_id}'"
        )

        actual_chat_name = chat.get("name")
        assert actual_chat_name.startswith("0x"), (
            f"Invalid chat name format: Expected name to start with '0x', found '{actual_chat_name}'"
        )

        last_message = chat.get("lastMessage", {})
        actual_text = last_message.get("text")
        assert actual_text == expected_text, (
            f"Message text mismatch: Expected '{expected_text}', found '{actual_text}'"
        )

        assert "compressedKey" in last_message, "Missing 'compressedKey' in last message"

    def validate_event_against_response(self, event, fields_to_validate):
        chats_in_event = event.get("event", {}).get("chats", [])
        assert len(chats_in_event) > 0, "No chats found in the event"

        response_chat = self.response["result"]["chats"][0]
        event_chat = chats_in_event[0]

        for response_field, event_field in fields_to_validate.items():
            response_value = response_chat.get("lastMessage", {}).get(response_field)
            event_value = event_chat.get("lastMessage", {}).get(event_field)
            assert response_value == event_value, (
                f"Mismatch for '{response_field}': Expected '{response_value}', found '{event_value}'"
            )

    def run_all_validations(self, expected_chat_id, expected_display_name, expected_text):
        self.validate_response_structure()
        self.validate_chat_data(expected_chat_id, expected_display_name, expected_text)
        logger.info("All validations passed for the one-to-one message response.")
