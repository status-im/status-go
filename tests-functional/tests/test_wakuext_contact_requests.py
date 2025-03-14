import pytest
from steps.messenger import MessengerSteps
from resources.enums import MessageContentType


@pytest.mark.rpc
@pytest.mark.usefixtures("setup_two_unprivileged_nodes")
class TestContactRequests(MessengerSteps):

    def test_send_contact_request(self):
        message_text = "test_send_contact_request"
        response = self.sender.wakuext_service.send_contact_request(self.receiver.public_key, message_text)
        self.sender.verify_json_schema(response, "wakuext_sendContactRequest")

        contacts = response.get("result", {}).get("contacts", [])
        assert len(contacts) >= 1, "Expected response to have at least one contact"

        contact_request_message = self.get_message_by_content_type(response, content_type=MessageContentType.CONTACT_REQUEST.value)
        assert len(contact_request_message) == 1, f"Expected one message with contentType {MessageContentType.CONTACT_REQUEST.value}"
        assert contact_request_message[0].get("text") == message_text

        sent_request_messages = self.get_message_by_content_type(response, content_type=MessageContentType.SYSTEM_MESSAGE_MUTUAL_EVENT_SENT.value)
        assert len(sent_request_messages) == 1, f"Expected one message with contentType {MessageContentType.SYSTEM_MESSAGE_MUTUAL_EVENT_SENT.value}"
        assert sent_request_messages[0].get("text") == f"You sent a contact request to @{self.receiver.public_key}"

    def test_add_contact(self):
        response = self.sender.wakuext_service.add_contact(self.receiver.public_key, self.receiver.display_name)
        self.sender.verify_json_schema(response, "wakuext_addContact")

        contacts = response.get("result", {}).get("contacts", [])
        assert len(contacts) >= 1, "Expected response to have at least one contact"
        assert contacts[0].get("displayName") == self.receiver.display_name

        contact_request_message = self.get_message_by_content_type(response, content_type=MessageContentType.CONTACT_REQUEST.value)
        assert len(contact_request_message) == 1, f"Expected one message with contentType {MessageContentType.CONTACT_REQUEST.value}"
        assert contact_request_message[0].get("text") == "Please add me to your contacts"

        sent_request_messages = self.get_message_by_content_type(response, content_type=MessageContentType.SYSTEM_MESSAGE_MUTUAL_EVENT_SENT.value)
        assert len(sent_request_messages) == 1, f"Expected one message with contentType {MessageContentType.SYSTEM_MESSAGE_MUTUAL_EVENT_SENT.value}"
        assert sent_request_messages[0].get("text") == f"You sent a contact request to @{self.receiver.public_key}"

    def test_accept_contact_request(self):
        message_id = self.send_contact_request_and_wait_for_signal_to_be_received()
        response = self.receiver.wakuext_service.accept_contact_request(message_id)
        self.sender.verify_json_schema(response, "wakuext_acceptContactRequest")

        contacts = response.get("result", {}).get("contacts", [])
        assert len(contacts) >= 1, "Expected response to have at least one contact"
        assert contacts[0].get("displayName") == self.sender.display_name

        contact_request_messages = self.get_message_by_content_type(response, content_type=MessageContentType.CONTACT_REQUEST.value)
        assert len(contact_request_messages) == 1, f"Expected one message with contentType {MessageContentType.CONTACT_REQUEST.value}"
        assert contact_request_messages[0].get("text") == "contact_request"

        accept_request_messages = self.get_message_by_content_type(
            response, content_type=MessageContentType.SYSTEM_MESSAGE_MUTUAL_EVENT_ACCEPTED.value
        )
        assert (
            len(accept_request_messages) == 1
        ), f"Expected one message with contentType {MessageContentType.SYSTEM_MESSAGE_MUTUAL_EVENT_ACCEPTED.value}"
        assert accept_request_messages[0].get("text") == f"You accepted @{self.sender.public_key}'s contact request"

    def test_decline_contact_request(self):
        message_id = self.send_contact_request_and_wait_for_signal_to_be_received()
        response = self.receiver.wakuext_service.decline_contact_request(message_id)
        self.sender.verify_json_schema(response, "wakuext_declineContactRequest")

        contacts = response.get("result", {}).get("contacts", [])
        assert len(contacts) >= 1, "Expected response to have at least one contact"
        assert contacts[0].get("displayName") == self.sender.display_name

        contact_request_messages = self.get_message_by_content_type(response, content_type=MessageContentType.CONTACT_REQUEST.value)
        assert len(contact_request_messages) == 1, f"Expected one message with contentType {MessageContentType.CONTACT_REQUEST.value}"
        assert contact_request_messages[0].get("text") == "contact_request"

    def test_accept_latest_contact_request_for_contact(self):
        self.send_contact_request_and_wait_for_signal_to_be_received()
        response = self.receiver.wakuext_service.accept_latest_contact_request_for_contact(self.sender.public_key)
        self.sender.verify_json_schema(response, "wakuext_acceptLatestContactRequestForContact")

        contacts = response.get("result", {}).get("contacts", [])
        assert len(contacts) >= 1, "Expected response to have at least one contact"
        assert contacts[0].get("displayName") == self.sender.display_name

        contact_request_messages = self.get_message_by_content_type(response, content_type=MessageContentType.CONTACT_REQUEST.value)
        assert len(contact_request_messages) == 1, f"Expected one message with contentType {MessageContentType.CONTACT_REQUEST.value}"
        assert contact_request_messages[0].get("text") == "contact_request"

        accept_request_messages = self.get_message_by_content_type(
            response, content_type=MessageContentType.SYSTEM_MESSAGE_MUTUAL_EVENT_ACCEPTED.value
        )
        assert (
            len(accept_request_messages) == 1
        ), f"Expected one message with contentType {MessageContentType.SYSTEM_MESSAGE_MUTUAL_EVENT_ACCEPTED.value}"
        assert accept_request_messages[0].get("text") == f"You accepted @{self.sender.public_key}'s contact request"

    def test_dismiss_latest_contact_request_for_contact(self):
        self.send_contact_request_and_wait_for_signal_to_be_received()
        response = self.receiver.wakuext_service.dismiss_latest_contact_request_for_contact(self.sender.public_key)
        self.sender.verify_json_schema(response, "wakuext_dismissLatestContactRequestForContact")

        contacts = response.get("result", {}).get("contacts", [])
        assert len(contacts) >= 1, "Expected response to have at least one contact"
        assert contacts[0].get("displayName") == self.sender.display_name

        contact_request_messages = self.get_message_by_content_type(response, content_type=MessageContentType.CONTACT_REQUEST.value)
        assert len(contact_request_messages) == 1, f"Expected one message with contentType {MessageContentType.CONTACT_REQUEST.value}"
        assert contact_request_messages[0].get("text") == "contact_request"
