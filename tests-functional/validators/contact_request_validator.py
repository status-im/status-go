import logging

from src.steps.common import logger


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

    def run_all_validations(self, expected_chat_id, expected_display_name, expected_text):
        self.validate_response_structure()
        self.validate_chat_data(expected_chat_id, expected_display_name, expected_text)
        logger.info("All validations passed for the contact request response.")
