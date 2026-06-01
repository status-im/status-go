import re
import time
import pytest

from clients.api import ApiResponseError
from clients.signals import SignalType
from resources.enums import MessageContentType
from resources.constants import FULL_NODE, LIGHT_CLIENT
from steps import async_messenger


@pytest.mark.rpc
@pytest.mark.asyncio
class TestMessageReactions:

    @pytest.fixture
    async def sender(self, async_backend_new_profile, waku_light_client):
        return await async_backend_new_profile("sender", waku_light_client=waku_light_client)

    @pytest.fixture
    async def receiver(self, async_backend_new_profile, waku_light_client):
        return await async_backend_new_profile("receiver", waku_light_client=waku_light_client)

    @pytest.mark.parametrize(
        "waku_light_client",
        [
            pytest.param(False, id=FULL_NODE),
            pytest.param(True, id=LIGHT_CLIENT, marks=pytest.mark.light_client_7393),
        ],
        indirect=True,
    )
    async def test_one_to_one_message_reactions(self, sender, receiver, waku_light_client):
        """Test message reactions with different wakuV2LightClient configurations"""
        await async_messenger.make_contacts(sender, receiver)
        response = sender.wakuext_service.send_one_to_one_message(receiver.public_key, "test_message")
        message = async_messenger.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
        message_id, sender_chat_id = message["id"], message["chatId"]
        receiver_chat_id = receiver.wakuext_service.chats()[0]["id"]

        # Send emoji reaction
        response = receiver.wakuext_service.send_emoji_reaction(receiver_chat_id, message_id, "2764")
        # TODO: Add more assertions on response

        # Wait for sender to receive the reaction
        await sender.wait_for_signal(SignalType.MESSAGES_NEW, pattern="emojiReactions", timeout=60, check_buffer=True)

        result = sender.wakuext_service.emoji_reactions_by_chat_id_message_id(sender_chat_id, message_id)
        # TODO: Add more assertions on response
        assert all(
            (
                len(result) == 1,
                result[0]["chatId"] == receiver_chat_id,
                result[0]["messageId"] == message_id,
                result[0]["emoji"] == "2764",
            )
        )
        emoji_id = result[0]["id"]

        response = receiver.wakuext_service.send_emoji_reaction_retraction(emoji_id)
        # TODO: Add more assertions on response
        assert response["chats"][0]["id"] == receiver_chat_id

        await sender.wait_for_signal(SignalType.MESSAGES_NEW, pattern="retracted", timeout=60, check_buffer=True)
        response = sender.wakuext_service.emoji_reactions_by_chat_id_message_id(sender_chat_id, message_id)
        assert not response

        response = sender.wakuext_service.send_one_to_one_message(receiver.public_key, "test_message 1")
        message_1 = async_messenger.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]

        # Send emoji reaction
        response = receiver.wakuext_service.send_emoji_reaction(receiver_chat_id, message_1["id"], "1f642")
        emoji_1_id = response["emojiReactions"][0]["id"]
        await sender.wait_for_signal(SignalType.MESSAGES_NEW, pattern=emoji_1_id, timeout=60, check_buffer=True)

        response = receiver.wakuext_service.send_one_to_one_message(sender.public_key, "test_message 2")
        message_2 = async_messenger.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
        response = sender.wakuext_service.send_emoji_reaction(sender_chat_id, message_2["id"], "1f641")
        emoji_2_id = response["emojiReactions"][0]["id"]
        await receiver.wait_for_signal(SignalType.MESSAGES_NEW, pattern=emoji_2_id, timeout=60, check_buffer=True)

        time.sleep(10)
        result = sender.wakuext_service.emoji_reactions_by_chat_id(sender_chat_id, 20)
        # TODO: Add more assertions on response
        assert len(result) == 2
        for item in result:
            assert all(
                (
                    item["chatId"] == sender_chat_id,
                    item["messageId"] == message_2["id"],
                    item["emoji"] == "1f641",
                )
            ) or all(
                (
                    item["chatId"] == receiver_chat_id,
                    item["messageId"] == message_1["id"],
                    item["emoji"] == "1f642",
                )
            )

    @pytest.mark.parametrize(
        "waku_light_client",
        [
            pytest.param(False, id=FULL_NODE),
        ],
        indirect=True,
    )
    async def test_limit_of_20_reactions(self, sender, receiver, waku_light_client):
        """Test that you cannot send more than 20 message reactions on a single message"""
        await async_messenger.make_contacts(sender, receiver)
        response = sender.wakuext_service.send_one_to_one_message(receiver.public_key, "test_message")
        message = async_messenger.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
        message_id, sender_chat_id = message["id"], message["chatId"]
        receiver_chat_id = receiver.wakuext_service.chats()[0]["id"]

        # Send 20 emojis
        emojis = [
            "1f600",
            "1f913",
            "1f604",
            "1f601",
            "1f606",
            "1f605",
            "1f923",
            "1f602",
            "1f97a",
            "1f642",
            "1f643",
            "1f609",
            "1f60a",
            "1f607",
            "1f970",
            "1f60d",
            "1f929",
            "1f618",
            "1f617",
            "263a",
        ]
        last_emoji_id = None
        for emoji in emojis:
            response = sender.wakuext_service.send_emoji_reaction(sender_chat_id, message_id, emoji)
            assert response["emojiReactions"][0]["emoji"] == emoji
            last_emoji_id = response["emojiReactions"][0]["id"]

        # The 21st emoji sent should fail
        with pytest.raises(ApiResponseError, match=re.escape("too many emoji reactions for message")):
            _ = sender.wakuext_service.send_emoji_reaction(sender_chat_id, message_id, "1f60b")

        # Test retract the 20th emoji and adding a new one
        response = sender.wakuext_service.send_emoji_reaction_retraction(last_emoji_id)

        response = sender.wakuext_service.send_emoji_reaction(sender_chat_id, message_id, "2693")
        assert response["emojiReactions"][0]["emoji"] == "2693"
        new_emoji_id = response["emojiReactions"][0]["id"]

        # Wait for receiver to get the reaction
        await receiver.wait_for_signal(SignalType.MESSAGES_NEW, pattern=new_emoji_id, timeout=60, check_buffer=True)

        # Test receiver sending the SAME type of a previous reaction (should be allowed)
        response = receiver.wakuext_service.send_emoji_reaction(receiver_chat_id, message_id, "2693")
        emoji_2_id = response["emojiReactions"][0]["id"]
        assert response["emojiReactions"][0]["emoji"] == "2693"

        await sender.wait_for_signal(SignalType.MESSAGES_NEW, pattern=emoji_2_id, timeout=60, check_buffer=True)
