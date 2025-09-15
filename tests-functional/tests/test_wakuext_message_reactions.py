import time

import pytest

from clients.signals import SignalType
from resources.enums import MessageContentType
from steps.messenger import MessengerSteps


@pytest.mark.rpc
class TestMessageReactions(MessengerSteps):

    @pytest.mark.parametrize("waku_light_client", [False, True], indirect=True, ids=["wakuV2LightClient_False", "wakuV2LightClient_True"])
    def test_one_to_one_message_reactions(self, backend_new_profile, waku_light_client):
        """Test message reactions with different wakuV2LightClient configurations"""
        # Initialize two backends (sender and receiver) for this test
        self.sender = backend_new_profile("sender", waku_light_client=waku_light_client)
        self.receiver = backend_new_profile("receiver", waku_light_client=waku_light_client)

        self.make_contacts(self.sender, self.receiver)
        response = self.sender.wakuext_service.send_one_to_one_message(self.receiver.public_key, "test_message")
        message = self.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
        message_id, sender_chat_id = message["id"], message["chatId"]
        receiver_chat_id = self.receiver.wakuext_service.rpc_request(method="chats")[0]["id"]
        # Send emoji reaction (V1)
        response = self.receiver.wakuext_service.rpc_request(method="sendEmojiReaction", params=[receiver_chat_id, message_id, 1])
        # TODO: Add more assertions on response
        self.sender.find_signal_containing_pattern(
            SignalType.MESSAGES_NEW.value,
            event_pattern="emojiReactions",
            timeout=60,
        )

        result = self.sender.wakuext_service.rpc_request(
            method="emojiReactionsByChatIDMessageID",
            params=[sender_chat_id, message_id],
        )
        # TODO: Add more assertions on response
        assert all(
            (
                len(result) == 1,
                result[0]["chatId"] == receiver_chat_id,
                result[0]["messageId"] == message_id,
                result[0]["emoji"] == "❤️",
            )
        )
        emoji_id = result[0]["id"]

        response = self.receiver.wakuext_service.rpc_request(
            method="sendEmojiReactionRetraction",
            params=[
                emoji_id,
            ],
        )
        # TODO: Add more assertions on response
        assert response["chats"][0]["id"] == receiver_chat_id

        self.sender.find_signal_containing_pattern(
            SignalType.MESSAGES_NEW.value,
            event_pattern="retracted",
            timeout=60,
        )
        response = self.sender.wakuext_service.rpc_request(
            method="emojiReactionsByChatIDMessageID",
            params=[sender_chat_id, message_id],
        )
        assert not response

        response = self.sender.wakuext_service.send_one_to_one_message(self.receiver.public_key, "test_message 1")
        message_1 = self.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
        # Send emoji reaction (V2)
        emoji_1_id = self.receiver.wakuext_service.rpc_request(method="sendEmojiReactionV2", params=[receiver_chat_id, message_1["id"], "🙂"])[
            "emojiReactions"
        ][0]["id"]
        self.sender.find_signal_containing_pattern(
            SignalType.MESSAGES_NEW.value,
            event_pattern=emoji_1_id,
            timeout=60,
        )

        response = self.receiver.wakuext_service.send_one_to_one_message(self.sender.public_key, "test_message 2")
        message_2 = self.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
        emoji_2_id = self.sender.wakuext_service.rpc_request(method="sendEmojiReactionV2", params=[sender_chat_id, message_2["id"], "🙁"])[
            "emojiReactions"
        ][0]["id"]
        self.receiver.find_signal_containing_pattern(
            SignalType.MESSAGES_NEW.value,
            event_pattern=emoji_2_id,
            timeout=60,
        )
        time.sleep(10)
        result = self.sender.wakuext_service.rpc_request(method="emojiReactionsByChatID", params=[sender_chat_id, None, 20])
        # TODO: Add more assertions on response
        assert len(result) == 2
        for item in result:
            assert all(
                (
                    item["chatId"] == sender_chat_id,
                    item["messageId"] == message_2["id"],
                    item["emoji"] == "🙁",
                )
            ) or all(
                (
                    item["chatId"] == receiver_chat_id,
                    item["messageId"] == message_1["id"],
                    item["emoji"] == "🙂",
                )
            )
