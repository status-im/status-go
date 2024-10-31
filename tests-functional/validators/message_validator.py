import json
import logging

logger = logging.getLogger(__name__)


class MessageValidator:
    def __init__(self, response):
        self.response = response

    def validate_response(self, expected_chat_id, expected_message, expected_sender):
        try:
            result = self.response.get("result")
            if not result:
                raise ValueError("No result found in the response.")

            chat_info = result.get("chats", [])[0]
            actual_chat_id = chat_info.get("id")
            if actual_chat_id != expected_chat_id:
                raise AssertionError(f"Chat ID mismatch. Expected: {expected_chat_id}, Found: {actual_chat_id}")

            last_message = chat_info.get("lastMessage")
            if not last_message:
                raise AssertionError("No lastMessage field in chat information.")

            actual_message_text = last_message.get("text")
            if actual_message_text != expected_message:
                raise AssertionError(
                    f"Message text mismatch. Expected: '{expected_message}', Found: '{actual_message_text}'")

            actual_sender = last_message.get("displayName")
            if actual_sender != expected_sender:
                raise AssertionError(f"Sender mismatch. Expected: '{expected_sender}', Found: '{actual_sender}'")

            sent_timestamp = last_message.get("timestamp")
            if not sent_timestamp:
                raise AssertionError("Timestamp is missing in lastMessage data.")
            logger.info(
                f"Message '{expected_message}' sent by '{expected_sender}' at timestamp {sent_timestamp} validated successfully.")

        except AssertionError as e:
            logger.error(f"Validation failed: {str(e)}")
            raise
        except Exception as e:
            logger.error(f"Unexpected error during validation: {str(e)}")
            raise