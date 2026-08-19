from datetime import datetime, timedelta, timezone

import pytest

from resources.enums import ChatType, ChatPreviewFilterType, MuteType
from steps import messenger
from waku_params import parametrize_waku_light_client


@pytest.mark.rpc
@parametrize_waku_light_client
@pytest.mark.parametrize("backend_factory", [{"privileged": False}], indirect=True, ids=["privileged_False"])
class TestChatActions:

    @pytest.fixture()
    def sender(self, backend_new_profile, waku_light_client):
        return backend_new_profile("sender", waku_light_client=waku_light_client)

    @pytest.fixture()
    def receiver(self, backend_new_profile, waku_light_client):
        return backend_new_profile("receiver", waku_light_client=waku_light_client)

    @pytest.mark.light_client_7393
    def test_all_chats(self, sender, receiver):
        messenger.make_contacts(sender, receiver)
        private_group_id = messenger.join_private_group(admin=sender, member=receiver)
        sender.wakuext_service.send_chat_message(private_group_id, "test_message")
        messenger.send_multiple_one_to_one_messages(1, sender=sender, receiver=receiver)

        chats = sender.wakuext_service.chats()
        assert len(chats) == 2

        private_group_chat = next((chat for chat in chats if chat.get("id") == private_group_id), None)
        assert private_group_chat is not None
        assert private_group_chat.get("chatType", "") == ChatType.PRIVATE_GROUP_CHAT.value

        one_to_one_chat = next((chat for chat in chats if chat.get("id") == receiver.public_key), None)
        assert one_to_one_chat is not None
        assert one_to_one_chat.get("chatType", "") == ChatType.ONE_TO_ONE.value

    def test_chat_by_chat_id(self, sender, receiver):
        sent_texts, _ = messenger.send_multiple_one_to_one_messages(1, sender=sender, receiver=receiver)
        chat_id = receiver.public_key

        chat = sender.wakuext_service.chat(chat_id)
        assert chat.get("chatType", 0) == ChatType.ONE_TO_ONE.value
        assert chat.get("lastMessage", {}).get("text", "") == sent_texts[0]

    @pytest.mark.light_client_7393
    def test_chats_preview(self, sender, receiver):
        # One to one
        messenger.make_contacts(sender, receiver)
        messenger.send_multiple_one_to_one_messages(1, sender=sender, receiver=receiver)
        one_to_one_chat_id = receiver.public_key

        # Group
        private_group_chat_id = messenger.join_private_group(admin=sender, member=receiver)
        sender.wakuext_service.send_group_chat_message(private_group_chat_id, "test_message_group")

        # Community
        community_id = messenger.create_community(sender)
        community_chat_id = messenger.join_community(member=receiver, admin=sender, community_id=community_id)
        sender.wakuext_service.send_chat_message(community_chat_id, "test_message_community")

        chats_previews = sender.wakuext_service.chats_preview(ChatPreviewFilterType.Community.value)
        assert len(chats_previews) == 1
        assert chats_previews[0].get("id", "") == community_chat_id

        chats_previews = sender.wakuext_service.chats_preview(ChatPreviewFilterType.NonCommunity.value)
        assert len(chats_previews) == 2
        assert {chat.get("id", "") for chat in chats_previews} == {one_to_one_chat_id, private_group_chat_id}

    @pytest.mark.light_client_7393
    def test_active_chats(self, sender, receiver):
        messenger.make_contacts(sender, receiver)
        messenger.send_multiple_one_to_one_messages(1, sender=sender, receiver=receiver)
        one_to_one_chat_id = receiver.public_key
        private_group_chat_id = messenger.join_private_group(admin=sender, member=receiver)

        chats = sender.wakuext_service.active_chats()
        assert len(chats) == 2

        sender.wakuext_service.deactivate_chat(private_group_chat_id, False)

        chats = sender.wakuext_service.active_chats()
        assert len(chats) == 1
        assert chats[0].get("id", 0) == one_to_one_chat_id

    def test_mute_chat(self, sender, receiver):
        messenger.send_multiple_one_to_one_messages(1, sender=sender, receiver=receiver)
        chat_id = receiver.public_key

        result = sender.wakuext_service.mute_chat(chat_id)
        assert result == "0001-01-01T00:00:00Z"

        chat = sender.wakuext_service.chat(chat_id)
        assert chat.get("muted", False) is True
        assert chat.get("muteTill", "") == result

    @pytest.mark.parametrize(
        "mute_type, time_delta",
        [
            # We use 3 cases here to reduce execution time.
            # Uncomment the other cases below if additional scenarios need to be tested
            # or if debugging specific mute durations is required.
            (MuteType.MUTE_FOR15_MIN.value, timedelta(minutes=15)),
            # (MuteType.MUTE_FOR1_HR.value, timedelta(hours=1)),
            # (MuteType.MUTE_FOR8_HR.value, timedelta(hours=8)),
            (MuteType.MUTE_FOR1_WEEK.value, timedelta(days=7)),
            # (MuteType.MUTE_TILL1_MIN.value, timedelta(minutes=1)),
            (MuteType.MUTE_FOR24_HR.value, timedelta(hours=24)),
        ],
    )
    def test_mute_chat_v2(self, sender, receiver, mute_type, time_delta):
        messenger.send_multiple_one_to_one_messages(1, sender=sender, receiver=receiver)
        chat_id = receiver.public_key

        result = sender.wakuext_service.mute_chat_v2(chat_id, mute_type)
        actual = datetime.fromisoformat(result.replace("Z", "+00:00"))

        expected = datetime.now(timezone.utc) + time_delta
        diff = expected - actual
        assert abs(diff.total_seconds()) < 2  # 2 sec margin

        chat = sender.wakuext_service.chat(chat_id)
        assert chat.get("muted", False) is True
        assert chat.get("muteTill", "") == result

    @pytest.mark.parametrize(
        "mute_type",
        [
            # As test above
            MuteType.MUTE_TILL_UNMUTED.value,
            # MuteType.UNMUTED.value,
        ],
    )
    def test_unmute_mute_chat_v2_till_unmuted(self, sender, receiver, mute_type):
        messenger.send_multiple_one_to_one_messages(1, sender=sender, receiver=receiver)
        chat_id = receiver.public_key

        result = sender.wakuext_service.mute_chat_v2(chat_id, mute_type)
        assert result == "0001-01-01T00:00:00Z"

        response = sender.wakuext_service.unmute_chat(chat_id)
        assert response is None

        chat = sender.wakuext_service.chat(chat_id)
        assert chat.get("muted", True) is False

    def test_clear_history(self, sender, receiver):
        messenger.send_multiple_one_to_one_messages(1, sender=sender, receiver=receiver)
        chat_id = receiver.public_key

        response = sender.wakuext_service.chat(chat_id)
        last_message = response.get("lastMessage", -1)
        assert isinstance(last_message, dict)

        response = sender.wakuext_service.clear_history(chat_id)
        # TODO: Add more assertions on response
        last_message = response.get("chats", [])[0].get("lastMessage", -1)
        assert last_message is None

    @pytest.mark.parametrize(
        "preserve_history, expected",
        [
            (False, type(None)),
            (True, dict),
        ],
    )
    def test_deactivate_chat(self, sender, receiver, preserve_history, expected):
        messenger.send_multiple_one_to_one_messages(1, sender=sender, receiver=receiver)
        chat_id = receiver.public_key

        response = sender.wakuext_service.deactivate_chat(chat_id, preserve_history)
        # TODO: Add more assertions on response

        chat = response.get("chats", [])[0]
        assert chat.get("active", -1) is False
        assert isinstance(chat.get("lastMessage", -1), expected)

    def test_save_chat(self, sender):
        chat_id = "123"
        response = sender.wakuext_service.save_chat(chat_id, active=True)
        assert response is None

        chat = sender.wakuext_service.chat(chat_id)
        assert chat.get("id", "") == chat_id
        assert chat.get("active", -1) is True

    def test_create_one_to_one_chat(self, sender, receiver):
        chat_id = receiver.public_key
        response = sender.wakuext_service.create_one_to_one_chat(chat_id, ens_name="")
        chats = response.get("chats", [])
        assert len(chats) == 1
        chat = chats[0]
        assert chat.get("id", "") == chat_id
        assert chat.get("active", -1) is True
