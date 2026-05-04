from time import sleep, time
from uuid import uuid4
import pytest

from clients.services.wakuext import SendChatMessagePayload
from clients.signals import SignalType
from resources.enums import MessageContentType
from resources.constants import FULL_NODE, LIGHT_CLIENT
from steps import async_messenger


@pytest.mark.rpc
@pytest.mark.asyncio
@pytest.mark.parametrize(
    "waku_light_client",
    [
        pytest.param(False, id=FULL_NODE),
        pytest.param(True, id=LIGHT_CLIENT, marks=pytest.mark.xfail(reason="status-go#7393 filter subscription race", strict=False)),
    ],
    indirect=True,
)
class TestSendingChatMessages:

    @pytest.fixture
    async def sender(self, async_backend_new_profile, waku_light_client):
        return await async_backend_new_profile("sender", waku_light_client=waku_light_client)

    @pytest.fixture
    async def receiver(self, async_backend_new_profile, waku_light_client):
        return await async_backend_new_profile("receiver", waku_light_client=waku_light_client)

    async def test_send_one_to_one_message(self, sender, receiver):
        # Generate unique message text BEFORE registering waiter for race-free waiting
        message_text = f"test_message_0_{uuid4()}"

        # Register signal waiter BEFORE sending to eliminate race condition
        # The signal is automatically awaited when the context exits
        async with receiver.expect_signal(
            SignalType.MESSAGES_NEW,
            pattern=message_text,
            timeout=60,
        ):
            # Send message while waiter is already registered
            response = sender.wakuext_service.send_one_to_one_message(
                receiver.public_key,
                message_text,
            )
        # Signal is automatically awaited when context exits

        # Original assertions on response
        chat = response["chats"][0]
        assert chat["id"] == receiver.public_key
        assert chat["lastMessage"]["displayName"] == sender.display_name

        # Verify message in sender's chat history
        chat_response = sender.wakuext_service.chat_messages(receiver.public_key)
        messages = chat_response.get("messages", [])
        assert len(messages) == 1
        actual_text = messages[0].get("text", "")
        assert actual_text == message_text

    async def test_send_chat_message_community(self, sender, receiver):
        community_id = async_messenger.create_community(sender)
        community_chat_id = await async_messenger.join_community(member=receiver, admin=sender, community_id=community_id)

        text = "test_message"
        response = sender.wakuext_service.send_chat_message(community_chat_id, text)
        # TODO: Add more assertions on response

        response = sender.wakuext_service.chat_messages(community_chat_id)
        messages = response.get("messages", [])
        assert len(messages) == 1
        actual_text = messages[0].get("text", "")
        assert actual_text == text

    async def test_send_chat_message_private_group(self, sender, receiver):
        await async_messenger.make_contacts(sender, receiver)
        private_group_id = await async_messenger.join_private_group(admin=sender, member=receiver)

        text = "test_message"
        response = sender.wakuext_service.send_chat_message(private_group_id, text)

        response = sender.wakuext_service.chat_messages(private_group_id)
        expected_message = async_messenger.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
        actual_text = expected_message.get("text", "")
        assert actual_text == text

    async def test_send_chat_messages_same_chat(self, sender, receiver):
        community_id = async_messenger.create_community(sender)
        community_chat_id = await async_messenger.join_community(member=receiver, admin=sender, community_id=community_id)

        payload = [
            SendChatMessagePayload(chat_id=community_chat_id, text=f"test_message_{i}", content_type=MessageContentType.TEXT_PLAIN.value)
            for i in range(5)
        ]
        response = sender.wakuext_service.send_chat_messages(payload)
        # TODO: Add more assertions on response

        response = sender.wakuext_service.chat_messages(community_chat_id)
        messages = response.get("messages", [])
        assert len(payload) == 5

        actual_texts = [m.get("text", "") for m in messages]
        expected_texts = [m.get("text", "") for m in payload]
        expected_texts.reverse()
        assert actual_texts == expected_texts

    async def test_send_chat_messages_different_chats(self, sender, receiver):
        # Group
        await async_messenger.make_contacts(sender, receiver)
        private_group_chat_id = await async_messenger.join_private_group(admin=sender, member=receiver)

        # Community
        community_id = async_messenger.create_community(sender)
        community_chat_id = await async_messenger.join_community(member=receiver, admin=sender, community_id=community_id)

        payload = [
            SendChatMessagePayload(chat_id=private_group_chat_id, text="test_message_group", content_type=MessageContentType.TEXT_PLAIN.value),
            SendChatMessagePayload(chat_id=community_chat_id, text="test_message_community", content_type=MessageContentType.TEXT_PLAIN.value),
        ]
        response = sender.wakuext_service.send_chat_messages(payload)

        chats = response.get("chats", [])
        assert len(chats) == 2
        messages = response.get("messages", [])
        assert len(messages) == 2

    async def test_send_group_message(self, sender, receiver):
        await async_messenger.make_contacts(sender, receiver)
        private_group_id = await async_messenger.join_private_group(admin=sender, member=receiver)

        text = "test_message_group"
        response = sender.wakuext_service.send_group_chat_message(private_group_id, text)
        # TODO: Add more assertions on response

        response = sender.wakuext_service.chat_messages(private_group_id)
        expected_message = async_messenger.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
        actual_text = expected_message.get("text", "")
        assert actual_text == text

    # Using delete_message is a workaround that might be considered an incorrect behaviour
    # TODO: create more realistic scenario where the message is intercepted in the network and not delivered,
    # use community messages to avoid 1-1 and group chats reliability mechanisms on protocol level
    async def test_resend_one_to_one_message(self, sender, receiver):
        await async_messenger.make_contacts(sender, receiver)

        _, responses = async_messenger.send_multiple_one_to_one_messages(1, sender=sender, receiver=receiver)
        message_id = responses[0].get("messages", [])[0].get("id", "")
        receiver_chat_id = sender.public_key

        # Wait for message to be received
        await receiver.wait_for_signal(SignalType.MESSAGES_NEW, pattern=message_id, timeout=60, check_buffer=True)

        response = receiver.wakuext_service.chat_messages(receiver_chat_id)
        messages = response.get("messages", [])
        assert len(messages) == 4

        receiver.wakuext_service.delete_message(message_id)
        response = receiver.wakuext_service.chat_messages(receiver_chat_id)
        messages = response.get("messages", [])
        assert len(messages) == 3

        sender.wakuext_service.resend_chat_message(message_id)

        # Wait for resent message
        await receiver.wait_for_signal(SignalType.MESSAGES_NEW, pattern=message_id, timeout=60, check_buffer=True)

        deadline = 60
        start_time = time()
        while True:
            response = receiver.wakuext_service.chat_messages(receiver_chat_id)
            messages = response.get("messages", [])
            if len(messages) == 4:
                break

            if time() - start_time >= deadline:
                assert len(messages) == 4

            sleep(0.5)
