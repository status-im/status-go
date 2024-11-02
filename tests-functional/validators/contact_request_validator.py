import logging

from src.libs.custom_logger import get_custom_logger

logger = get_custom_logger(__name__)


class ContactRequestValidator:
    def __init__(self, response):
        self.response = response

    def validate_response_structure(self):
        assert self.response.get("jsonrpc") == "2.0", "Invalid JSON-RPC version"
        assert "result" in self.response, "Missing 'result' in response"

    def validate_chat_data(self, expected_chat_id, expected_display_name, expected_text):
        chats = self.response["result"].get("chats", [])
        assert len(chats) > 0, "No chats found in the response"

        chat = chats[0]
        assert chat.get("id") == expected_chat_id, f"Chat ID mismatch: Expected {expected_chat_id}"
        assert chat.get("name").startswith("0x"), "Invalid chat name format"

        last_message = chat.get("lastMessage", {})
        assert last_message.get("text") == expected_text, "Message text mismatch"
        assert last_message.get("contactRequestState") == 1, "Unexpected contact request state"
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
                f"Mismatch for '{response_field}': expected '{response_value}', got '{event_value}'"
            )

    def run_all_validations(self, expected_chat_id, expected_display_name, expected_text):
        self.validate_response_structure()
        self.validate_chat_data(expected_chat_id, expected_display_name, expected_text)
        logger.info("All validations passed for the contact request response.")
