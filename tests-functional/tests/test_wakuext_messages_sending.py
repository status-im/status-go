import pytest

from steps.messenger import MessengerSteps


@pytest.mark.usefixtures("setup_two_unprivileged_nodes")
@pytest.mark.rpc
class TestSendingChatMessages(MessengerSteps):
    def test_one_to_one_messages(self):
        responses = self.one_to_one_message(1)
        self.receiver.verify_json_schema(responses[0], method="wakuext_sendOneToOneMessage")

        chat = responses[0]["result"]["chats"][0]
        assert chat["id"] == self.receiver.public_key
        assert chat["lastMessage"]["displayName"] == self.sender.display_name

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
