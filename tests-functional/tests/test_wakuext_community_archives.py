from uuid import uuid4
import pytest
import time

from steps.messenger import MessengerSteps
from clients.signals import SignalType


@pytest.mark.rpc
class TestCommunityArchives(MessengerSteps):
    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        """Initialize three backends (creator, member and another_member) for each test function"""
        self.creator = backend_new_profile("creator", codex_config_enabled=True)
        self.creator.wakuext_service.set_archive_distribution_preference("codex")
        self.creator.wakuext_service.update_message_archive_interval(15)

        self.member = backend_new_profile("member", codex_config_enabled=True)
        self.member.wakuext_service.set_archive_distribution_preference("codex")
        self.member.wakuext_service.update_message_archive_interval(15)

        self.another_member = backend_new_profile("member", codex_config_enabled=True)
        self.another_member.wakuext_service.set_archive_distribution_preference("codex")
        self.another_member.wakuext_service.update_message_archive_interval(15)

        self.fake_address = "0x" + str(uuid4())[:8]
        self.community_id = self.create_community(self.creator, historyArchiveSupportEnabled=True)
        self.join_community(member=self.member, admin=self.creator)
        self.display_name = "chat_" + str(uuid4())
        self.chat_payload = {
            "identity": {
                "displayName": self.display_name,
                "emoji": "😀",
                "color": "#1f2c75",
                "description": self.display_name,
            },
            "viewersCanPostReactions": False,
            "hideIfPermissionsNotMet": False,
            "permissions": {"access": 1},
        }

    def test_community_archive_index_exists(self):
        # Create community chat
        create_resp = self.creator.wakuext_service.create_community_chat(self.community_id, self.chat_payload)
        chat_id = create_resp.get("chats")[0].get("id")

        # Wait for member to receive chat creation signal
        self.member.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=chat_id, timeout=10)
        # Send a message to the community chat
        text = f"Hi @{self.member.public_key}"
        send_resp = self.creator.wakuext_service.send_chat_message(chat_id, text)
        assert send_resp.get("chats")[0].get("lastMessage").get("text") == text
        message_id = send_resp.get("messages", [])[0].get("id", "")

        # Wait for member to receive the new message
        self.member.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=message_id, timeout=10)
        member_msgs_resp = self.member.wakuext_service.chat_messages(chat_id)
        assert member_msgs_resp.get("messages")[0].get("text") == text

        # self.join_community(member=self.another_member, admin=self.creator)
        # self.another_member.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=chat_id, timeout=10)
        # # member_msgs_resp = self.another_member.wakuext_service.chat_messages(chat_id)
        # messages = member_msgs_resp.get("messages", [])

        time.sleep(30)

        # Ensure that the community archive is available for the creator
        has_archive = self.creator.wakuext_service.has_community_archive(self.community_id)
        assert has_archive is True
