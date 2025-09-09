import pytest

from datetime import datetime, timedelta
from resources.enums import ChatType, ChatPreviewFilterType, MuteType
from steps.messenger import MessengerSteps


@pytest.mark.rpc
@pytest.mark.parametrize("waku_light_client", [False, True], indirect=True, ids=["waku_light_client_False", "waku_light_client_True"])
@pytest.mark.parametrize("backend_factory", [{"privileged": False}], indirect=True, ids=["privileged_False"])
class TestChatActions(MessengerSteps):

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile, waku_light_client):
        """Initialize two backends (sender and receiver) for each test function"""
        self.sender = backend_new_profile("sender", waku_light_client)
        self.receiver = backend_new_profile("receiver", waku_light_client)

    def test_all_chats(self):
        self.make_contacts(self.sender, self.receiver)
        private_group_id = self.join_private_group(admin=self.sender, member=self.receiver)
        self.sender.wakuext_service.send_chat_message(private_group_id, "test_message")
        self.send_multiple_one_to_one_messages(1, sender=self.sender, receiver=self.receiver)

        chats = self.sender.wakuext_service.chats()
        assert len(chats) == 2
        assert chats[0].get("chatType", 0) == ChatType.ONE_TO_ONE.value
        assert chats[1].get("chatType", 0) == ChatType.PRIVATE_GROUP_CHAT.value

    def test_chat_by_chat_id(self):
        sent_texts, _ = self.send_multiple_one_to_one_messages(1, sender=self.sender, receiver=self.receiver)
        chat_id = self.receiver.public_key

        chat = self.sender.wakuext_service.chat(chat_id)
        assert chat.get("chatType", 0) == ChatType.ONE_TO_ONE.value
        assert chat.get("lastMessage", {}).get("text", "") == sent_texts[0]

    def test_chats_preview(self):
        # One to one
        self.make_contacts(self.sender, self.receiver)
        self.send_multiple_one_to_one_messages(1, sender=self.sender, receiver=self.receiver)
        one_to_one_chat_id = self.receiver.public_key

        # Group
        private_group_chat_id = self.join_private_group(admin=self.sender, member=self.receiver)
        self.sender.wakuext_service.send_group_chat_message(private_group_chat_id, "test_message_group")

        # Community
        self.create_community(self.sender)
        community_chat_id = self.join_community(member=self.receiver, admin=self.sender)
        self.sender.wakuext_service.send_chat_message(community_chat_id, "test_message_community")

        chats_previews = self.sender.wakuext_service.chats_preview(ChatPreviewFilterType.Community.value)
        assert len(chats_previews) == 1
        assert chats_previews[0].get("id", "") == community_chat_id

        chats_previews = self.sender.wakuext_service.chats_preview(ChatPreviewFilterType.NonCommunity.value)
        assert len(chats_previews) == 2
        assert chats_previews[0].get("id", "") == one_to_one_chat_id
        assert chats_previews[1].get("id", "") == private_group_chat_id

    def test_active_chats(self):
        self.make_contacts(self.sender, self.receiver)
        self.send_multiple_one_to_one_messages(1, sender=self.sender, receiver=self.receiver)
        one_to_one_chat_id = self.receiver.public_key
        private_group_chat_id = self.join_private_group(admin=self.sender, member=self.receiver)

        chats = self.sender.wakuext_service.active_chats()
        # TODO: Add more assertions on response
        assert len(chats) == 2

        self.sender.wakuext_service.deactivate_chat(private_group_chat_id, False)

        chats = self.sender.wakuext_service.active_chats()
        assert len(chats) == 1
        assert chats[0].get("id", 0) == one_to_one_chat_id

    def test_mute_chat(self):
        self.send_multiple_one_to_one_messages(1, sender=self.sender, receiver=self.receiver)
        chat_id = self.receiver.public_key

        result = self.sender.wakuext_service.mute_chat(chat_id)
        assert result == "0001-01-01T00:00:00Z"

        chat = self.sender.wakuext_service.chat(chat_id)
        assert chat.get("muted", False) is True
        assert chat.get("muteTill", "") == result

    @pytest.mark.skip(reason="Skipping mute chat tests due to failing on local build")
    # TODO: check in nightly build locally
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
        self.send_multiple_one_to_one_messages(1, sender=self.sender, receiver=self.receiver)
        chat_id = self.receiver.public_key

        result = self.sender.wakuext_service.mute_chat_v2(chat_id, mute_type)
        actual = datetime.strptime(result, "%Y-%m-%dT%H:%M:%SZ")

        expected = datetime.now() + time_delta
        diff = expected - actual
        assert diff.total_seconds() < 2  # 2sec margin

        chat = self.sender.wakuext_service.chat(chat_id)
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
        self.send_multiple_one_to_one_messages(1, sender=self.sender, receiver=self.receiver)
        chat_id = self.receiver.public_key

        result = self.sender.wakuext_service.mute_chat_v2(chat_id, mute_type)
        assert result == "0001-01-01T00:00:00Z"

        response = self.sender.wakuext_service.unmute_chat(chat_id)
        assert response is None

        chat = self.sender.wakuext_service.chat(chat_id)
        assert chat.get("muted", True) is False

    def test_clear_history(self):
        self.send_multiple_one_to_one_messages(1, sender=self.sender, receiver=self.receiver)
        chat_id = self.receiver.public_key

        response = self.sender.wakuext_service.chat(chat_id)
        last_message = response.get("lastMessage", -1)
        assert isinstance(last_message, dict)

        response = self.sender.wakuext_service.clear_history(chat_id)
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
    def test_deactivate_chat(self, preserve_history, expected):
        self.send_multiple_one_to_one_messages(1, sender=self.sender, receiver=self.receiver)
        chat_id = self.receiver.public_key

        response = self.sender.wakuext_service.deactivate_chat(chat_id, preserve_history)
        # TODO: Add more assertions on response

        chat = response.get("chats", [])[0]
        assert chat.get("active", -1) is False
        assert isinstance(chat.get("lastMessage", -1), expected)

    def test_save_chat(self):
        chat_id = "123"
        response = self.sender.wakuext_service.save_chat(chat_id, active=True)
        assert response is None

        chat = self.sender.wakuext_service.chat(chat_id)
        assert chat.get("id", "") == chat_id
        assert chat.get("active", -1) is True

    def test_create_one_to_one_chat(self):
        chat_id = self.receiver.public_key
        response = self.sender.wakuext_service.create_one_to_one_chat(chat_id, ens_name="")
        chats = response.get("chats", [])
        assert len(chats) == 1
        chat = chats[0]
        assert chat.get("id", "") == chat_id
        assert chat.get("active", -1) is True
