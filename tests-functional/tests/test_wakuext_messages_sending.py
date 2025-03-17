import pytest

from resources.enums import MessageContentType
from steps.messenger import MessengerSteps


@pytest.mark.usefixtures("setup_two_unprivileged_nodes")
@pytest.mark.rpc
class TestSendingChatMessages(MessengerSteps):
    def test_send_one_to_one_message(self):
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

    def test_send_community_message(self):
        self.create_community(self.sender)
        community_chat_id = self.join_community(self.receiver)

        text = "test_message"
        response = self.sender.wakuext_service.send_community_chat_message(community_chat_id, text)
        self.sender.verify_json_schema(response, method="wakuext_sendChatMessage")

        response = self.sender.wakuext_service.chat_messages(community_chat_id)
        messages = response.get("result", {}).get("messages", [])
        assert len(messages) == 1
        actual_text = messages[0].get("text", "")
        assert actual_text == text

    def test_send_group_message(self):
        self.make_contacts()
        private_group_id = self.join_private_group()

        sent_texts, responses = self.send_multiple_group_messages(private_group_id, 1)
        self.sender.verify_json_schema(responses[0], method="wakuext_sendGroupChatMessage")

        response = self.sender.wakuext_service.chat_messages(private_group_id)
        expected_message = self.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
        actual_text = expected_message.get("text", "")
        assert actual_text == sent_texts[0]
