import re
from uuid import uuid4
import pytest
from resources.constants import FULL_NODE, LIGHT_CLIENT
from resources.enums import MessageContentType
from steps import async_messenger
from clients.api import ApiResponseError


@pytest.mark.rpc
@pytest.mark.asyncio
@pytest.mark.parametrize(
    "waku_light_client",
    [
        pytest.param(False, id=FULL_NODE),
        pytest.param(True, id=LIGHT_CLIENT, marks=pytest.mark.light_client_7393),
    ],
    indirect=True,
)
class TestContactRequests:

    @pytest.fixture
    async def sender(self, async_backend_new_profile, waku_light_client):
        return await async_backend_new_profile("sender", waku_light_client=waku_light_client)

    @pytest.fixture
    async def receiver(self, async_backend_new_profile, waku_light_client):
        return await async_backend_new_profile("receiver", waku_light_client=waku_light_client)

    async def test_send_contact_request(self, sender, receiver):
        message_text = "test_send_contact_request"
        response = sender.wakuext_service.send_contact_request(receiver.public_key, message_text)
        # TODO: Add more assertions on response

        contacts = response.get("contacts", [])
        assert len(contacts) >= 1, "Expected response to have at least one contact"

        contact_request_message = async_messenger.get_message_by_content_type(response, MessageContentType.CONTACT_REQUEST.value)
        assert len(contact_request_message) == 1, f"Expected one message with contentType {MessageContentType.CONTACT_REQUEST.value}"
        assert contact_request_message[0].get("text") == message_text

        sent_request_messages = async_messenger.get_message_by_content_type(response, MessageContentType.SYSTEM_MESSAGE_MUTUAL_EVENT_SENT.value)
        assert len(sent_request_messages) == 1, f"Expected one message with contentType {MessageContentType.SYSTEM_MESSAGE_MUTUAL_EVENT_SENT.value}"
        assert sent_request_messages[0].get("text") == f"You sent a contact request to @{receiver.public_key}"

    async def test_add_contact(self, sender, receiver):
        response = sender.wakuext_service.add_contact(receiver.public_key, receiver.display_name)
        # TODO: Add more assertions on response

        contacts = response.get("contacts", [])
        assert len(contacts) >= 1, "Expected response to have at least one contact"
        assert contacts[0].get("displayName") == receiver.display_name

        contact_request_message = async_messenger.get_message_by_content_type(response, MessageContentType.CONTACT_REQUEST.value)
        assert len(contact_request_message) == 1, f"Expected one message with contentType {MessageContentType.CONTACT_REQUEST.value}"
        assert contact_request_message[0].get("text") == "Please add me to your contacts"

        sent_request_messages = async_messenger.get_message_by_content_type(response, MessageContentType.SYSTEM_MESSAGE_MUTUAL_EVENT_SENT.value)
        assert len(sent_request_messages) == 1, f"Expected one message with contentType {MessageContentType.SYSTEM_MESSAGE_MUTUAL_EVENT_SENT.value}"
        assert sent_request_messages[0].get("text") == f"You sent a contact request to @{receiver.public_key}"

    async def test_accept_contact_request(self, sender, receiver):
        message_id = await async_messenger.send_contact_request_and_wait(sender, receiver)
        response = receiver.wakuext_service.accept_contact_request(message_id, sender.public_key)
        # TODO: Add more assertions on response

        contacts = response.get("contacts", [])
        assert len(contacts) >= 1, "Expected response to have at least one contact"
        assert contacts[0].get("displayName") == sender.display_name

        contact_request_messages = async_messenger.get_message_by_content_type(response, MessageContentType.CONTACT_REQUEST.value)
        assert len(contact_request_messages) == 1, f"Expected one message with contentType {MessageContentType.CONTACT_REQUEST.value}"
        assert contact_request_messages[0].get("text") == "contact_request"

        accept_request_messages = async_messenger.get_message_by_content_type(response, MessageContentType.SYSTEM_MESSAGE_MUTUAL_EVENT_ACCEPTED.value)
        assert (
            len(accept_request_messages) == 1
        ), f"Expected one message with contentType {MessageContentType.SYSTEM_MESSAGE_MUTUAL_EVENT_ACCEPTED.value}"
        assert accept_request_messages[0].get("text") == f"You accepted @{sender.public_key}'s contact request"

    async def test_decline_contact_request(self, sender, receiver):
        message_id = await async_messenger.send_contact_request_and_wait(sender, receiver)
        # Send without contact ID first and expect an error
        with pytest.raises(ApiResponseError, match=re.escape("decline-contact-request: invalid contact id")):
            receiver.wakuext_service.decline_contact_request(message_id, "")
        response = receiver.wakuext_service.decline_contact_request(message_id, sender.public_key)
        # TODO: Add more assertions on response

        contacts = response.get("contacts", [])
        assert len(contacts) >= 1, "Expected response to have at least one contact"
        assert contacts[0].get("displayName") == sender.display_name

        contact_request_messages = async_messenger.get_message_by_content_type(response, MessageContentType.CONTACT_REQUEST.value)
        assert len(contact_request_messages) == 1, f"Expected one message with contentType {MessageContentType.CONTACT_REQUEST.value}"
        assert contact_request_messages[0].get("text") == "contact_request"

    async def test_accept_latest_contact_request_for_contact(self, sender, receiver):
        await async_messenger.send_contact_request_and_wait(sender, receiver)
        response = receiver.wakuext_service.accept_latest_contact_request_for_contact(sender.public_key)
        # TODO: Add more assertions on response

        contacts = response.get("contacts", [])
        assert len(contacts) >= 1, "Expected response to have at least one contact"
        assert contacts[0].get("displayName") == sender.display_name

        contact_request_messages = async_messenger.get_message_by_content_type(response, MessageContentType.CONTACT_REQUEST.value)
        assert len(contact_request_messages) == 1, f"Expected one message with contentType {MessageContentType.CONTACT_REQUEST.value}"
        assert contact_request_messages[0].get("text") == "contact_request"

        accept_request_messages = async_messenger.get_message_by_content_type(response, MessageContentType.SYSTEM_MESSAGE_MUTUAL_EVENT_ACCEPTED.value)
        assert (
            len(accept_request_messages) == 1
        ), f"Expected one message with contentType {MessageContentType.SYSTEM_MESSAGE_MUTUAL_EVENT_ACCEPTED.value}"
        assert accept_request_messages[0].get("text") == f"You accepted @{sender.public_key}'s contact request"

    async def test_dismiss_latest_contact_request_for_contact(self, sender, receiver):
        await async_messenger.send_contact_request_and_wait(sender, receiver)
        response = receiver.wakuext_service.dismiss_latest_contact_request_for_contact(sender.public_key)
        # TODO: Add more assertions on response

        contacts = response.get("contacts", [])
        assert len(contacts) >= 1, "Expected response to have at least one contact"
        assert contacts[0].get("displayName") == sender.display_name

        contact_request_messages = async_messenger.get_message_by_content_type(response, MessageContentType.CONTACT_REQUEST.value)
        assert len(contact_request_messages) == 1, f"Expected one message with contentType {MessageContentType.CONTACT_REQUEST.value}"
        assert contact_request_messages[0].get("text") == "contact_request"

    async def test_get_latest_contact_request_for_contact(self, sender, receiver):
        await async_messenger.send_contact_request_and_wait(sender, receiver)
        response = receiver.wakuext_service.get_latest_contact_request_for_contact(sender.public_key)
        # TODO: Add more assertions on response

        contact_request_message = async_messenger.get_message_by_content_type(response, MessageContentType.CONTACT_REQUEST.value)
        assert len(contact_request_message) == 1, f"Expected one message with contentType {MessageContentType.CONTACT_REQUEST.value}"
        assert contact_request_message[0].get("text") == "contact_request"

    async def test_retract_contact_request(self, sender, receiver):
        await async_messenger.send_contact_request_and_wait(sender, receiver)
        response = sender.wakuext_service.retract_contact_request(receiver.public_key)
        # TODO: Add more assertions on response

        contacts = response.get("contacts", [])
        assert len(contacts) >= 1, "Expected response to have at least one contact"

        retract_request_messages = async_messenger.get_message_by_content_type(response, MessageContentType.SYSTEM_MESSAGE_MUTUAL_EVENT_REMOVED.value)
        assert (
            len(retract_request_messages) == 1
        ), f"Expected one message with contentType {MessageContentType.SYSTEM_MESSAGE_MUTUAL_EVENT_REMOVED.value}"
        assert retract_request_messages[0].get("text") == f"You removed @{receiver.public_key} as a contact"

    async def test_remove_contact(self, sender, receiver):
        message_id = await async_messenger.send_contact_request_and_wait(sender, receiver)
        await async_messenger.accept_contact_request_and_wait(message_id, sender=sender, receiver=receiver)
        response = sender.wakuext_service.remove_contact(receiver.public_key)
        # TODO: Add more assertions on response

        contacts = response.get("contacts", [])
        assert len(contacts) >= 1, "Expected response to have at least one contact"

        retract_request_messages = async_messenger.get_message_by_content_type(response, MessageContentType.SYSTEM_MESSAGE_MUTUAL_EVENT_REMOVED.value)
        assert (
            len(retract_request_messages) == 1
        ), f"Expected one message with contentType {MessageContentType.SYSTEM_MESSAGE_MUTUAL_EVENT_REMOVED.value}"
        assert retract_request_messages[0].get("text") == f"You removed @{receiver.public_key} as a contact"

    async def test_set_contact_local_nickname(self, sender, receiver):
        message_id = await async_messenger.send_contact_request_and_wait(sender, receiver)
        await async_messenger.accept_contact_request_and_wait(message_id, sender=sender, receiver=receiver)
        nickname = str(uuid4())
        response = sender.wakuext_service.set_contact_local_nickname(receiver.public_key, nickname)
        # TODO: Add more assertions on response

        contacts = response.get("contacts", [])
        assert len(contacts) >= 1, "Expected response to have at least one contact"
        assert contacts[0].get("displayName") == receiver.display_name
        assert contacts[0].get("localNickname") == nickname
        assert contacts[0].get("primaryName") == nickname
