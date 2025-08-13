from uuid import uuid4
import pytest
from resources.enums import MessageContentType
from steps.messenger import MessengerSteps


@pytest.mark.rpc
@pytest.mark.parametrize("waku_light_client", [False, True], indirect=True, ids=["wakuV2LightClient_False", "wakuV2LightClient_True"])
class TestContactRequests(MessengerSteps):

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile, waku_light_client):
        """Initialize two backends (sender and receiver) for each test function"""
        self.sender = backend_new_profile("sender", waku_light_client=waku_light_client)
        self.receiver = backend_new_profile("receiver", waku_light_client=waku_light_client)

    def test_send_contact_request(self):
        message_text = "test_send_contact_request"
        response = self.sender.wakuext_service.send_contact_request(self.receiver.public_key, message_text)
        # TODO: Add more assertions on response

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
        # TODO: Add more assertions on response

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
        message_id = self.send_contact_request_and_wait_for_signal_to_be_received(self.sender, self.receiver)
        response = self.receiver.wakuext_service.accept_contact_request(message_id)
        # TODO: Add more assertions on response

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
        message_id = self.send_contact_request_and_wait_for_signal_to_be_received(self.sender, self.receiver)
        response = self.receiver.wakuext_service.decline_contact_request(message_id)
        # TODO: Add more assertions on response

        contacts = response.get("result", {}).get("contacts", [])
        assert len(contacts) >= 1, "Expected response to have at least one contact"
        assert contacts[0].get("displayName") == self.sender.display_name

        contact_request_messages = self.get_message_by_content_type(response, content_type=MessageContentType.CONTACT_REQUEST.value)
        assert len(contact_request_messages) == 1, f"Expected one message with contentType {MessageContentType.CONTACT_REQUEST.value}"
        assert contact_request_messages[0].get("text") == "contact_request"

    def test_accept_latest_contact_request_for_contact(self):
        self.send_contact_request_and_wait_for_signal_to_be_received(self.sender, self.receiver)
        response = self.receiver.wakuext_service.accept_latest_contact_request_for_contact(self.sender.public_key)
        # TODO: Add more assertions on response

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
        self.send_contact_request_and_wait_for_signal_to_be_received(self.sender, self.receiver)
        response = self.receiver.wakuext_service.dismiss_latest_contact_request_for_contact(self.sender.public_key)
        # TODO: Add more assertions on response

        contacts = response.get("result", {}).get("contacts", [])
        assert len(contacts) >= 1, "Expected response to have at least one contact"
        assert contacts[0].get("displayName") == self.sender.display_name

        contact_request_messages = self.get_message_by_content_type(response, content_type=MessageContentType.CONTACT_REQUEST.value)
        assert len(contact_request_messages) == 1, f"Expected one message with contentType {MessageContentType.CONTACT_REQUEST.value}"
        assert contact_request_messages[0].get("text") == "contact_request"

    def test_get_latest_contact_request_for_contact(self):
        self.send_contact_request_and_wait_for_signal_to_be_received(self.sender, self.receiver)
        response = self.receiver.wakuext_service.get_latest_contact_request_for_contact(self.sender.public_key)
        # TODO: Add more assertions on response

        contact_request_message = self.get_message_by_content_type(response, content_type=MessageContentType.CONTACT_REQUEST.value)
        assert len(contact_request_message) == 1, f"Expected one message with contentType {MessageContentType.CONTACT_REQUEST.value}"
        assert contact_request_message[0].get("text") == "contact_request"

    def test_retract_contact_request(self):
        self.send_contact_request_and_wait_for_signal_to_be_received(self.sender, self.receiver)
        response = self.sender.wakuext_service.retract_contact_request(self.receiver.public_key)
        # TODO: Add more assertions on response

        contacts = response.get("result", {}).get("contacts", [])
        assert len(contacts) >= 1, "Expected response to have at least one contact"

        retract_request_messages = self.get_message_by_content_type(
            response, content_type=MessageContentType.SYSTEM_MESSAGE_MUTUAL_EVENT_REMOVED.value
        )
        assert (
            len(retract_request_messages) == 1
        ), f"Expected one message with contentType {MessageContentType.SYSTEM_MESSAGE_MUTUAL_EVENT_REMOVED.value}"
        assert retract_request_messages[0].get("text") == f"You removed @{self.receiver.public_key} as a contact"

    def test_remove_contact(self):
        message_id = self.send_contact_request_and_wait_for_signal_to_be_received(self.sender, self.receiver)
        self.accept_contact_request_and_wait_for_signal_to_be_received(message_id)
        response = self.sender.wakuext_service.remove_contact(self.receiver.public_key)
        # TODO: Add more assertions on response

        contacts = response.get("result", {}).get("contacts", [])
        assert len(contacts) >= 1, "Expected response to have at least one contact"

        retract_request_messages = self.get_message_by_content_type(
            response, content_type=MessageContentType.SYSTEM_MESSAGE_MUTUAL_EVENT_REMOVED.value
        )
        assert (
            len(retract_request_messages) == 1
        ), f"Expected one message with contentType {MessageContentType.SYSTEM_MESSAGE_MUTUAL_EVENT_REMOVED.value}"
        assert retract_request_messages[0].get("text") == f"You removed @{self.receiver.public_key} as a contact"

    def test_set_contact_local_nickname(self):
        message_id = self.send_contact_request_and_wait_for_signal_to_be_received(self.sender, self.receiver)
        self.accept_contact_request_and_wait_for_signal_to_be_received(message_id)
        nickname = str(uuid4())
        response = self.sender.wakuext_service.set_contact_local_nickname(self.receiver.public_key, nickname)
        # TODO: Add more assertions on response

        contacts = response.get("result", {}).get("contacts", [])
        assert len(contacts) >= 1, "Expected response to have at least one contact"
        assert contacts[0].get("displayName") == self.receiver.display_name
        assert contacts[0].get("localNickname") == nickname
        assert contacts[0].get("primaryName") == nickname
