from time import sleep
import pytest

from clients.services.wakuext import SendChatMessagePayload
from clients.signals import SignalType
from resources.enums import MessageContentType
from steps.messenger import MessengerSteps


@pytest.mark.parametrize("setup_two_unprivileged_nodes", [False, True], indirect=True, ids=["wakuV2LightClient_False", "wakuV2LightClient_True"])
@pytest.mark.rpc
class TestSendingChatMessages(MessengerSteps):
    def test_send_one_to_one_message(self, setup_two_unprivileged_nodes):
        sent_texts, responses = self.send_multiple_one_to_one_messages(1)
        self.receiver.verify_json_schema(responses[0], method="wakuext_sendOneToOneMessage")

        chat = responses[0]["result"]["chats"][0]
        assert chat["id"] == self.receiver.public_key
        assert chat["lastMessage"]["displayName"] == self.sender.display_name

        response = self.sender.wakuext_service.chat_messages(self.receiver.public_key)
        messages = response.get("result", {}).get("messages", [])
        assert len(messages) == 1
        actual_text = messages[0].get("text", "")
        assert actual_text == sent_texts[0]

    def test_send_chat_message_community(self, setup_two_unprivileged_nodes):
        self.create_community(self.sender)
        community_chat_id = self.join_community(self.receiver)

        text = "test_message"
        response = self.sender.wakuext_service.send_chat_message(community_chat_id, text)
        self.sender.verify_json_schema(response, method="wakuext_sendChatMessage")

        response = self.sender.wakuext_service.chat_messages(community_chat_id)
        messages = response.get("result", {}).get("messages", [])
        assert len(messages) == 1
        actual_text = messages[0].get("text", "")
        assert actual_text == text

    def test_send_chat_message_private_group(self, setup_two_unprivileged_nodes):
        self.make_contacts()
        private_group_id = self.join_private_group()

        text = "test_message"
        response = self.sender.wakuext_service.send_chat_message(private_group_id, text)

        response = self.sender.wakuext_service.chat_messages(private_group_id)
        expected_message = self.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
        actual_text = expected_message.get("text", "")
        assert actual_text == text

    def test_send_chat_messages_same_chat(self, setup_two_unprivileged_nodes):
        self.create_community(self.sender)
        community_chat_id = self.join_community(self.receiver)

        payload = [
            SendChatMessagePayload(chat_id=community_chat_id, text=f"test_message_{i}", content_type=MessageContentType.TEXT_PLAIN.value)
            for i in range(5)
        ]
        response = self.sender.wakuext_service.send_chat_messages(payload)
        self.sender.verify_json_schema(response, method="wakuext_sendChatMessage")  # the same schema as for sendChatMessage

        response = self.sender.wakuext_service.chat_messages(community_chat_id)
        messages = response.get("result", {}).get("messages", [])
        assert len(payload) == 5

        actual_texts = [m.get("text", "") for m in messages]
        expected_texts = [m.get("text", "") for m in payload]
        expected_texts.reverse()
        assert actual_texts == expected_texts

    def test_send_chat_messages_different_chats(self, setup_two_unprivileged_nodes):
        # Group
        self.make_contacts()
        private_group_chat_id = self.join_private_group()

        # Community
        self.create_community(self.sender)
        community_chat_id = self.join_community(self.receiver)

        payload = [
            SendChatMessagePayload(chat_id=private_group_chat_id, text="test_message_group", content_type=MessageContentType.TEXT_PLAIN.value),
            SendChatMessagePayload(chat_id=community_chat_id, text="test_message_community", content_type=MessageContentType.TEXT_PLAIN.value),
        ]
        response = self.sender.wakuext_service.send_chat_messages(payload)

        chats = response.get("result", {}).get("chats", [])
        assert len(chats) == 2
        messages = response.get("result", {}).get("messages", [])
        assert len(messages) == 2

    def test_send_group_message(self, setup_two_unprivileged_nodes):
        self.make_contacts()
        private_group_id = self.join_private_group()

        text = "test_message_group"
        response = self.sender.wakuext_service.send_group_chat_message(private_group_id, text)
        self.sender.verify_json_schema(response, method="wakuext_sendGroupChatMessage")

        response = self.sender.wakuext_service.chat_messages(private_group_id)
        expected_message = self.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
        actual_text = expected_message.get("text", "")
        assert actual_text == text

    # Using delete_message is a workaround that might be considered an incorrect behaviour
    # TODO: create more realistic scenario where the message is intercepted in the network and not delivered,
    # use community messages to avoid 1-1 and group chats reliability mechanisms on protocol level
    def test_resend_one_to_one_message(self, setup_two_unprivileged_nodes):
        self.make_contacts()

        _, responses = self.send_multiple_one_to_one_messages(1)
        message_id = responses[0].get("result", {}).get("messages", [])[0].get("id", "")
        receiver_chat_id = self.sender.public_key

        self.receiver.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=message_id, timeout=5)

        response = self.receiver.wakuext_service.chat_messages(receiver_chat_id)
        messages = response.get("result", {}).get("messages", [])
        assert len(messages) == 4

        self.receiver.wakuext_service.delete_message(message_id)
        response = self.receiver.wakuext_service.chat_messages(receiver_chat_id)
        messages = response.get("result", {}).get("messages", [])
        assert len(messages) == 3

        self.sender.wakuext_service.resend_chat_message(message_id)
        sleep(5)

        response = self.receiver.wakuext_service.chat_messages(receiver_chat_id)
        messages = response.get("result", {}).get("messages", [])
        assert len(messages) == 4
