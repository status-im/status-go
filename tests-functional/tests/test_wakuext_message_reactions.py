import time

import pytest

from clients.signals import SignalType
from resources.enums import MessageContentType
from steps.messenger import MessengerSteps


@pytest.mark.parametrize("setup_two_unprivileged_nodes", [False, True], indirect=True, ids=["wakuV2LightClient_False", "wakuV2LightClient_True"])
@pytest.mark.rpc
class TestMessageReactions(MessengerSteps):
    def test_one_to_one_message_reactions(self, setup_two_unprivileged_nodes):
        self.make_contacts()
        response = self.sender.wakuext_service.send_one_to_one_message(self.receiver.public_key, "test_message")
        message = self.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
        message_id, sender_chat_id = message["id"], message["chatId"]
        receiver_chat_id = self.receiver.wakuext_service.rpc_request(method="chats").json()["result"][0]["id"]
        response = self.receiver.wakuext_service.rpc_request(method="sendEmojiReaction", params=[receiver_chat_id, message_id, 1]).json()
        self.sender.verify_json_schema(response, "wakuext_sendEmojiReaction")
        self.sender.find_signal_containing_pattern(
            SignalType.MESSAGES_NEW.value,
            event_pattern="emojiReactions",
            timeout=60,
        )

        response = self.sender.wakuext_service.rpc_request(
            method="emojiReactionsByChatIDMessageID",
            params=[sender_chat_id, message_id],
        ).json()
        self.sender.verify_json_schema(response, "wakuext_emojiReactionsByChatIDMessageID")
        result = response["result"]
        assert all(
            (
                len(result) == 1,
                result[0]["chatId"] == receiver_chat_id,
                result[0]["messageId"] == message_id,
                result[0]["emojiId"] == 1,
            )
        )
        emoji_id = result[0]["id"]

        response = self.receiver.wakuext_service.rpc_request(
            method="sendEmojiReactionRetraction",
            params=[
                emoji_id,
            ],
        ).json()
        self.sender.verify_json_schema(response, "wakuext_sendEmojiReactionRetraction")
        assert response["result"]["chats"][0]["id"] == receiver_chat_id

        self.sender.find_signal_containing_pattern(
            SignalType.MESSAGES_NEW.value,
            event_pattern="retracted",
            timeout=60,
        )
        response = self.sender.wakuext_service.rpc_request(
            method="emojiReactionsByChatIDMessageID",
            params=[sender_chat_id, message_id],
        )
        assert not response.json()["result"]

        response = self.sender.wakuext_service.send_one_to_one_message(self.receiver.public_key, "test_message 1")
        message_1 = self.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
        emoji_1_id = self.receiver.wakuext_service.rpc_request(method="sendEmojiReaction", params=[receiver_chat_id, message_1["id"], 2]).json()[
            "result"
        ]["emojiReactions"][0]["id"]
        self.sender.find_signal_containing_pattern(
            SignalType.MESSAGES_NEW.value,
            event_pattern=emoji_1_id,
            timeout=60,
        )

        response = self.receiver.wakuext_service.send_one_to_one_message(self.sender.public_key, "test_message 2")
        message_2 = self.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
        emoji_2_id = self.sender.wakuext_service.rpc_request(method="sendEmojiReaction", params=[sender_chat_id, message_2["id"], 3]).json()[
            "result"
        ]["emojiReactions"][0]["id"]
        self.receiver.find_signal_containing_pattern(
            SignalType.MESSAGES_NEW.value,
            event_pattern=emoji_2_id,
            timeout=60,
        )
        time.sleep(10)
        response = self.sender.wakuext_service.rpc_request(method="emojiReactionsByChatID", params=[sender_chat_id, None, 20]).json()
        self.sender.verify_json_schema(response, "wakuext_emojiReactionsByChatID")
        result = response["result"]
        assert len(result) == 2
        for item in result:
            assert all(
                (
                    item["chatId"] == sender_chat_id,
                    item["messageId"] == message_2["id"],
                    item["emojiId"] == 3,
                )
            ) or all(
                (
                    item["chatId"] == receiver_chat_id,
                    item["messageId"] == message_1["id"],
                    item["emojiId"] == 2,
                )
            )
