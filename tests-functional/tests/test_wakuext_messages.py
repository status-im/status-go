import pytest
from uuid import uuid4

from tests.test_cases import MessengerTestCase


@pytest.mark.usefixtures("setup_two_unprivileged_nodes")
@pytest.mark.rpc
class TestMessages(MessengerTestCase):

    def test_get_chat_messages(self):
        message_text = f"test_message_{uuid4()}"
        self.sender.wakuext_service.send_message(self.receiver.public_key, message_text)

        sender_chat_id = self.receiver.public_key
        response = self.sender.wakuext_service.get_chat_messages(sender_chat_id)

        self.sender.verify_json_schema(response, method="wakuext_chatMessages")

        messages = response.get("result", {}).get("messages", [])
        assert len(messages) == 1
        actual_text = messages[0].get("text", "")
        assert actual_text == message_text

    def test_get_chat_messages_with_pagination(self):
        sender_chat_id = self.receiver.public_key
        sent_texts = []
        for i in range(5):
            message_text = f"test_message_{i}_{uuid4()}"
            sent_texts.insert(0, message_text)  # sent_texts ordered from the latest
            self.sender.wakuext_service.send_message(self.receiver.public_key, message_text)

        # Page 1
        chat_messages_res1 = self.sender.wakuext_service.get_chat_messages(sender_chat_id, cursor="", limit=3)

        cursor1 = chat_messages_res1.get("result", {}).get("cursor", "")
        messages_page1 = chat_messages_res1.get("result", {}).get("messages", [])
        assert len(messages_page1) == 3
        assert messages_page1[0].get("text", "") == sent_texts[0]
        assert messages_page1[1].get("text", "") == sent_texts[1]
        assert messages_page1[2].get("text", "") == sent_texts[2]

        # Page 2
        chat_messages_res2 = self.sender.wakuext_service.get_chat_messages(sender_chat_id, cursor=cursor1, limit=3)

        cursor2 = chat_messages_res2.get("result", {}).get("cursor", "")
        messages_page2 = chat_messages_res2.get("result", {}).get("messages", [])
        assert len(messages_page2) == 2
        assert messages_page2[0].get("text", "") == sent_texts[3]
        assert messages_page2[1].get("text", "") == sent_texts[4]
        assert cursor2 == ""
