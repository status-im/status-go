import pytest

from resources.enums import ChatType
from steps.messenger import MessengerSteps


@pytest.mark.usefixtures("setup_two_unprivileged_nodes")
@pytest.mark.rpc
class TestChatActions(MessengerSteps):

    def test_chats(self):
        sent_texts, _ = self.send_multiple_one_to_one_messages(1)

        response = self.sender.wakuext_service.chats()
        self.sender.verify_json_schema(response, method="wakuext_chats")

        chats = response.get("result", [])
        assert len(chats) == 1
        assert chats[0].get("chatType", 0) == ChatType.ONE_TO_ONE.value
        assert chats[0].get("lastMessage", {}).get("text", "") == sent_texts[0]
