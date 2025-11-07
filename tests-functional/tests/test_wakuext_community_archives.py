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
        # Community owner
        self.creator = backend_new_profile("creator", codex_config_enabled=True, message_archive_interval=10)
        # Define codex as archive distribution preference
        self.creator.wakuext_service.set_archive_distribution_preference("codex")

        # Create a first member that will join the community first
        self.member = backend_new_profile("member", codex_config_enabled=True, import_initial_delay=5)
        # Define codex as archive distribution preference
        self.member.wakuext_service.set_archive_distribution_preference("codex")

        # Create another member that will join the community later after the first message is sent
        self.another_member = backend_new_profile("member", codex_config_enabled=True, import_initial_delay=5)
        # Define codex as archive distribution preference
        self.another_member.wakuext_service.set_archive_distribution_preference("codex")

        self.fake_address = "0x" + str(uuid4())[:8]
        self.community_id = self.create_community(self.creator, historyArchiveSupportEnabled=True)

        # Ensure that no community archive exists initially
        has_archive = self.creator.wakuext_service.has_community_archive(self.community_id)
        assert has_archive is False, "Creator should not have community archive initially"
        has_archive = self.member.wakuext_service.has_community_archive(self.community_id)
        assert has_archive is False, "Member should not have community archive initially"
        has_archive = self.another_member.wakuext_service.has_community_archive(self.community_id)
        assert has_archive is False, "Another member should not have community archive initially"

        # Connect members to community codex client
        # In the real life, this would be done via DHT discovery
        info = self.creator.wakuext_service.debug()
        self.member.wakuext_service.connect(info["id"], info["addrs"])
        self.another_member.wakuext_service.connect(info["id"], info["addrs"])

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
        self.join_community(member=self.member, admin=self.creator)
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

        # Just make sure that archive are generated
        time.sleep(10)

        # Ensure that the community archive is available for the creator
        has_archive = self.creator.wakuext_service.has_community_archive(self.community_id)
        assert has_archive is True, "Creator should have community archive after messages are sent"

        # TODO: try to disable the store node ??
        # self.member.wakuext_service.toggle_use_mail_servers(enabled=False)

        # Another member joins and checks for the message
        self.join_community(member=self.another_member, admin=self.creator)
        self.another_member.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=chat_id, timeout=10)
        member_msgs_resp = self.another_member.wakuext_service.chat_messages(chat_id)
        assert member_msgs_resp.get("messages") is None, "Another member should not have messages before archive is dispatched"

        # Ensure that the another member received the archive dispatch message
        time.sleep(5)

        has_archive = self.member.wakuext_service.has_community_archive(self.community_id)
        assert has_archive is True, "Member should have community archive after messages are sent"

        has_archive = self.another_member.wakuext_service.has_community_archive(self.community_id)
        assert has_archive is True, "Another member should have community archive after messages are sent"

        member_msgs_resp = self.member.wakuext_service.chat_messages(chat_id)
        assert member_msgs_resp.get("messages")[0].get("text") == text, "Member should have the message after archive is dispatched"

        # TODO: Verify in db
