import pytest

from datetime import datetime, timedelta
from resources.enums import ChatType, ChatPreviewFilterType, MuteType
from steps.messenger import MessengerSteps


@pytest.mark.usefixtures("setup_two_unprivileged_nodes")
@pytest.mark.rpc
class TestChatActions(MessengerSteps):

    def test_all_chats(self):
        self.make_contacts()
        private_group_id = self.join_private_group()
        self.sender.wakuext_service.send_chat_message(private_group_id, "test_message")
        self.send_multiple_one_to_one_messages(1)

        response = self.sender.wakuext_service.chats()
        self.sender.verify_json_schema(response, method="wakuext_chats")

        chats = response.get("result", [])
        assert len(chats) == 2
        assert chats[0].get("chatType", 0) == ChatType.ONE_TO_ONE.value
        assert chats[1].get("chatType", 0) == ChatType.PRIVATE_GROUP_CHAT.value

    def test_chat_by_chat_id(self):
        sent_texts, _ = self.send_multiple_one_to_one_messages(1)
        chat_id = self.receiver.public_key

        response = self.sender.wakuext_service.chat(chat_id)
        self.sender.verify_json_schema(response, method="wakuext_chat")

        chat = response.get("result", {})
        assert chat.get("chatType", 0) == ChatType.ONE_TO_ONE.value
        assert chat.get("lastMessage", {}).get("text", "") == sent_texts[0]

    def test_chats_preview(self):
        # One to one
        self.make_contacts()
        self.send_multiple_one_to_one_messages(1)
        one_to_one_chat_id = self.receiver.public_key

        # Group
        private_group_chat_id = self.join_private_group()
        self.sender.wakuext_service.send_group_chat_message(private_group_chat_id, "test_message_group")

        # Community
        self.create_community(self.sender)
        community_chat_id = self.join_community(self.receiver)
        self.sender.wakuext_service.send_chat_message(community_chat_id, "test_message_community")

        response = self.sender.wakuext_service.chats_preview(ChatPreviewFilterType.Community.value)
        self.sender.verify_json_schema(response, method="wakuext_chatsPreview")

        chats_previews = response.get("result", [])
        assert len(chats_previews) == 1
        assert chats_previews[0].get("id", "") == community_chat_id

        response = self.sender.wakuext_service.chats_preview(ChatPreviewFilterType.NonCommunity.value)
        self.sender.verify_json_schema(response, method="wakuext_chatsPreview")
        chats_previews = response.get("result", [])
        assert len(chats_previews) == 2
        assert chats_previews[0].get("id", "") == one_to_one_chat_id
        assert chats_previews[1].get("id", "") == private_group_chat_id

    def test_active_chats(self):
        self.make_contacts()
        self.send_multiple_one_to_one_messages(1)
        one_to_one_chat_id = self.receiver.public_key
        private_group_chat_id = self.join_private_group()

        response = self.sender.wakuext_service.active_chats()
        self.sender.verify_json_schema(response, method="wakuext_chats")
        chats = response.get("result", [])
        assert len(chats) == 2

        self.sender.wakuext_service.deactivate_chat(private_group_chat_id, False)

        response = self.sender.wakuext_service.active_chats()

        chats = response.get("result", [])
        assert len(chats) == 1
        assert chats[0].get("id", 0) == one_to_one_chat_id

    def test_mute_chat(self):
        self.send_multiple_one_to_one_messages(1)
        chat_id = self.receiver.public_key

        response = self.sender.wakuext_service.mute_chat(chat_id)
        result = response.get("result", "")
        assert result == "0001-01-01T00:00:00Z"

        response = self.sender.wakuext_service.chat(chat_id)
        chat = response.get("result", {})
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
    def test_mute_chat_v2(self, mute_type, time_delta):
        self.send_multiple_one_to_one_messages(1)
        chat_id = self.receiver.public_key

        response = self.sender.wakuext_service.mute_chat_v2(chat_id, mute_type)
        result = response.get("result", "")
        actual = datetime.strptime(result, "%Y-%m-%dT%H:%M:%SZ")

        expected = datetime.now() + time_delta
        diff = expected - actual
        assert diff.total_seconds() < 2  # 2sec margin

        response = self.sender.wakuext_service.chat(chat_id)
        chat = response.get("result", {})
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
    def test_unmute_mute_chat_v2_till_unmuted(self, mute_type):
        self.send_multiple_one_to_one_messages(1)
        chat_id = self.receiver.public_key

        response = self.sender.wakuext_service.mute_chat_v2(chat_id, mute_type)
        result = response.get("result", "")
        assert result == "0001-01-01T00:00:00Z"

        response = self.sender.wakuext_service.unmute_chat(chat_id)
        assert response.get("result", "") is None

        response = self.sender.wakuext_service.chat(chat_id)
        chat = response.get("result", {})
        assert chat.get("muted", True) is False

    def test_clear_history(self):
        self.send_multiple_one_to_one_messages(1)
        chat_id = self.receiver.public_key

        response = self.sender.wakuext_service.chat(chat_id)
        last_message = response.get("result", {}).get("lastMessage", -1)
        assert isinstance(last_message, dict)

        response = self.sender.wakuext_service.clear_history(chat_id)
        self.sender.verify_json_schema(response, method="wakuext_clearHistory")

        last_message = response.get("result", {}).get("chats", [])[0].get("lastMessage", -1)
        assert last_message is None

    @pytest.mark.parametrize(
        "preserve_history, expected",
        [
            (False, type(None)),
            (True, dict),
        ],
    )
    def test_deactivate_chat(self, preserve_history, expected):
        self.send_multiple_one_to_one_messages(1)
        chat_id = self.receiver.public_key

        response = self.sender.wakuext_service.deactivate_chat(chat_id, preserve_history)
        self.sender.verify_json_schema(response, method="wakuext_deactivateChat")

        chat = response.get("result", {}).get("chats", [])[0]
        assert chat.get("active", -1) is False
        assert isinstance(chat.get("lastMessage", -1), expected)

    def test_save_chat(self):
        chat_id = "123"
        response = self.sender.wakuext_service.save_chat(chat_id, active=True)
        assert response.get("result", -1) is None

        response = self.sender.wakuext_service.chat(chat_id)
        chat = response.get("result", {})
        assert chat.get("id", "") == chat_id
        assert chat.get("active", -1) is True

    def test_create_one_to_one_chat(self):
        chat_id = self.receiver.public_key
        response = self.sender.wakuext_service.create_one_to_one_chat(chat_id, ens_name="")
        self.sender.verify_json_schema(response, method="wakuext_createOneToOneChat")

        chats = response.get("result", {}).get("chats", [])
        assert len(chats) == 1
        chat = chats[0]
        assert chat.get("id", "") == chat_id
        assert chat.get("active", -1) is True
