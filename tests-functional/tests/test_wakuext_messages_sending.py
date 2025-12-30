from time import sleep, time
import pytest

from clients.services.wakuext import SendChatMessagePayload
from clients.signals import SignalType
from resources.enums import MessageContentType
from steps.messenger import MessengerSteps


@pytest.mark.rpc
@pytest.mark.parametrize("waku_light_client", [False, True], indirect=True, ids=["wakuV2LightClient_False", "wakuV2LightClient_True"])
class TestSendingChatMessages(MessengerSteps):

    @pytest.fixture()
    def sender(self, backend_new_profile, waku_light_client):
        return backend_new_profile("sender", waku_light_client=waku_light_client)

    @pytest.fixture()
    def receiver(self, backend_new_profile, waku_light_client):
        return backend_new_profile("receiver", waku_light_client=waku_light_client)

    def test_send_one_to_one_message(self, sender, receiver):
        sent_texts, responses = self.send_multiple_one_to_one_messages(1, sender=sender, receiver=receiver)
        # TODO: Add more assertions on response

        chat = responses[0]["chats"][0]
        assert chat["id"] == receiver.public_key
        assert chat["lastMessage"]["displayName"] == sender.display_name

        response = sender.wakuext_service.chat_messages(receiver.public_key)
        messages = response.get("messages", [])
        assert len(messages) == 1
        actual_text = messages[0].get("text", "")
        assert actual_text == sent_texts[0]

    def test_send_chat_message_community(self, sender, receiver):
        self.create_community(sender)
        community_chat_id = self.join_community(member=receiver, admin=sender)

        text = "test_message"
        response = sender.wakuext_service.send_chat_message(community_chat_id, text)
        # TODO: Add more assertions on response

        response = sender.wakuext_service.chat_messages(community_chat_id)
        messages = response.get("messages", [])
        assert len(messages) == 1
        actual_text = messages[0].get("text", "")
        assert actual_text == text

    def test_send_chat_message_private_group(self, sender, receiver):
        self.make_contacts(sender, receiver)
        private_group_id = self.join_private_group(admin=sender, member=receiver)

        text = "test_message"
        response = sender.wakuext_service.send_chat_message(private_group_id, text)

        response = sender.wakuext_service.chat_messages(private_group_id)
        expected_message = self.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
        actual_text = expected_message.get("text", "")
        assert actual_text == text

    def test_send_chat_messages_same_chat(self, sender, receiver):
        self.create_community(sender)
        community_chat_id = self.join_community(member=receiver, admin=sender)

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

    def test_send_chat_messages_different_chats(self, sender, receiver):
        # Group
        self.make_contacts(sender, receiver)
        private_group_chat_id = self.join_private_group(admin=sender, member=receiver)

        # Community
        self.create_community(sender)
        community_chat_id = self.join_community(member=receiver, admin=sender)

        payload = [
            SendChatMessagePayload(chat_id=private_group_chat_id, text="test_message_group", content_type=MessageContentType.TEXT_PLAIN.value),
            SendChatMessagePayload(chat_id=community_chat_id, text="test_message_community", content_type=MessageContentType.TEXT_PLAIN.value),
        ]
        response = sender.wakuext_service.send_chat_messages(payload)

        chats = response.get("chats", [])
        assert len(chats) == 2
        messages = response.get("messages", [])
        assert len(messages) == 2

    def test_send_group_message(self, sender, receiver):
        self.make_contacts(sender, receiver)
        private_group_id = self.join_private_group(admin=sender, member=receiver)

        text = "test_message_group"
        response = sender.wakuext_service.send_group_chat_message(private_group_id, text)
        # TODO: Add more assertions on response

        response = sender.wakuext_service.chat_messages(private_group_id)
        expected_message = self.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
        actual_text = expected_message.get("text", "")
        assert actual_text == text

    # Using delete_message is a workaround that might be considered an incorrect behaviour
    # TODO: create more realistic scenario where the message is intercepted in the network and not delivered,
    # use community messages to avoid 1-1 and group chats reliability mechanisms on protocol level
    def test_resend_one_to_one_message(self, sender, receiver):
        self.make_contacts(sender, receiver)

        receiver_start = len(receiver.received_signals[SignalType.MESSAGES_NEW])
        _, responses = self.send_multiple_one_to_one_messages(1, sender=sender, receiver=receiver)
        message_id = responses[0].get("messages", [])[0].get("id", "")
        receiver_chat_id = sender.public_key

        with receiver.expect_signal(
            SignalType.MESSAGES_NEW,
            pattern=message_id,
            timeout=60,
            start=receiver_start,
        ):
            pass

        response = receiver.wakuext_service.chat_messages(receiver_chat_id)
        messages = response.get("messages", [])
        assert len(messages) == 4

        receiver.wakuext_service.delete_message(message_id)
        response = receiver.wakuext_service.chat_messages(receiver_chat_id)
        messages = response.get("messages", [])
        assert len(messages) == 3

        receiver_start = len(receiver.received_signals[SignalType.MESSAGES_NEW])
        sender.wakuext_service.resend_chat_message(message_id)
        with receiver.expect_signal(
            SignalType.MESSAGES_NEW,
            pattern=message_id,
            timeout=60,
            start=receiver_start,
        ):
            pass

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
