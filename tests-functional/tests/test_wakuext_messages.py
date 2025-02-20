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
