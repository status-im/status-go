from clients.signals import SignalType
from resources.enums import MessageContentType
from utils import fake


class AsyncMessengerSteps:
    """Base class with helper methods for async messenger tests.

    RPC calls are sync (fast, no need for async).
    Only signal waiting is async.
    """

    def get_message_by_content_type(self, response: dict, content_type: int, message_pattern: str = "") -> list[dict]:
        """Filter messages from response by content type."""
        matched_messages = []
        messages = response.get("messages", [])
        for message in messages:
            if message.get("contentType") != content_type:
                continue
            if not message_pattern or message_pattern in str(message):
                matched_messages.append(message)
        if matched_messages:
            return matched_messages
        raise ValueError(f"Failed to find a message with contentType '{content_type}' and message_pattern: `{message_pattern}` in response")

    def get_message_id(self, response: dict, index: int = 0) -> str:
        return response.get("messages", [])[index].get("id", "")

    def get_message_by_message_id(self, response: dict, message_id: str) -> dict:
        messages = response.get("messages", [])
        for message in messages:
            if message.get("id", "") == message_id:
                return message
        raise ValueError(f"Failed to find a message with message id '{message_id}' in response")

    async def send_contact_request_and_wait(self, sender, receiver, message_text: str = "contact_request") -> str:
        """Send contact request and wait for receiver to get the signal.

        Returns:
            str: The message ID of the contact request
        """
        # Sync RPC call
        response = sender.wakuext_service.send_contact_request(receiver.public_key, message_text)
        expected_message = self.get_message_by_content_type(response, MessageContentType.CONTACT_REQUEST.value)[0]
        message_id = expected_message.get("id")
        assert message_id, "Message ID should not be empty"

        # Async signal waiting
        await receiver.wait_for_signal(SignalType.MESSAGES_NEW, pattern=message_id, timeout=60, check_buffer=True)
        return message_id

    async def accept_contact_request_and_wait(self, message_id: str, sender, receiver) -> None:
        """Accept contact request and wait for sender to get the acceptance signal."""
        accepted_signal = f"@{receiver.public_key} accepted your contact request"
        # Sync RPC call
        receiver.wakuext_service.accept_contact_request(message_id)
        # Async signal waiting
        await sender.wait_for_signal(SignalType.MESSAGES_NEW, pattern=accepted_signal, timeout=60, check_buffer=True)

    async def make_contacts(self, sender, receiver) -> str:
        """Create a mutual contact between sender and receiver.

        Returns:
            str: The message ID of the contact request
        """
        # Sync RPC call
        existing_contacts = receiver.wakuext_service.get_contacts()
        if sender.public_key in str(existing_contacts):
            return ""

        message_id = await self.send_contact_request_and_wait(sender, receiver)
        await self.accept_contact_request_and_wait(message_id, sender, receiver)
        return message_id

    def validate_signal_event_against_response(self, signal_event: dict, fields_to_validate: dict, expected_message: dict) -> None:
        """Validate that signal event contains expected message with matching fields."""
        expected_message_id = expected_message.get("id")
        signal_event_messages = signal_event.get("event", {}).get("messages")
        assert signal_event_messages and len(signal_event_messages) > 0, "No messages found in the signal event"

        message = next(
            (msg for msg in signal_event_messages if msg.get("id") == expected_message_id),
            None,
        )
        assert message, f"Message with ID {expected_message_id} not found in the signal event"

        message_mismatch = []
        for response_field, event_field in fields_to_validate.items():
            response_value = expected_message[response_field]
            event_value = message[event_field]
            if response_value != event_value:
                message_mismatch.append(f"Field '{response_field}': Expected '{response_value}', Found '{event_value}'")

        if message_mismatch:
            raise AssertionError(
                "Some Sender RPC responses are not matching the signals received by the receiver.\n"
                "Details of mismatches:\n" + "\n".join(message_mismatch)
            )

    def create_community(self, node) -> str:
        """Create a community using the given node.

        This is a sync RPC call, no async needed.
        """
        # Handle both sync and async backends
        wakuext = getattr(node, "wakuext_service", None)
        if wakuext is None and hasattr(node, "backend"):
            wakuext = node.backend.wakuext_service
        assert wakuext is not None, "Node must have wakuext_service attribute"
        response = wakuext.create_community(fake.community_name(), fake.community_description())
        self.community_id = response.get("communities", [{}])[0].get("id")
        return self.community_id
