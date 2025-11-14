import logging
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
        self.creator = backend_new_profile("creator")
        # Define codex as archive distribution preference
        self.creator.wakuext_service.set_archive_distribution_preference("codex")
        # Enable community history archive protocol
        self.creator.wakuext_service.enable_codex_community_history_archive_protocol(
            {
                "CodexNodeConfig.DiscoveryPort": 8091,
            }
        )

        info = self.creator.wakuext_service.debug()

        # Create a first member that will join the community first
        self.member = backend_new_profile("member", import_initial_delay=5)
        # Define codex as archive distribution preference
        self.member.wakuext_service.set_archive_distribution_preference("codex")
        self.member.wakuext_service.enable_codex_community_history_archive_protocol(
            {
                "CodexNodeConfig.DiscoveryPort": 8092,
                "CodexNodeConfig.BootstrapNodes": f'["{info["spr"]}"]',
            }
        )

        # Create another member that will join the community later after the first message is sent
        self.another_member = backend_new_profile("member", import_initial_delay=5)
        # Define codex as archive distribution preference
        self.another_member.wakuext_service.set_archive_distribution_preference("codex")
        self.another_member.wakuext_service.enable_codex_community_history_archive_protocol(
            {
                "CodexNodeConfig.DiscoveryPort": 8093,
                "CodexNodeConfig.BootstrapNodes": f'["{info["spr"]}"]',
            }
        )

        # Using bootstrap nodes does not seem to be working in our setup,
        # thus we need to connect members manually.
        self.member.wakuext_service.connect(info["id"], info["addrs"])
        self.another_member.wakuext_service.connect(info["id"], info["addrs"])

        # Create the community
        self.fake_address = "0x" + str(uuid4())[:8]
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
        message_archive_interval = 80
        self.creator.wakuext_service.update_message_archive_interval(message_archive_interval)

        community_id = self.create_community(self.creator, history_archive_support_enabled=True)

        # Ensure that no community archive exists initially
        has_archive_index = self.creator.wakuext_service.has_community_archive(community_id)
        assert has_archive_index is False, "Creator should not have community archive initially"
        has_archive_index = self.member.wakuext_service.has_community_archive(community_id)
        assert has_archive_index is False, "Member should not have community archive initially"
        has_archive_index = self.another_member.wakuext_service.has_community_archive(community_id)
        assert has_archive_index is False, "Another member should not have community archive initially"

        # Create community chat
        create_resp = self.creator.wakuext_service.create_community_chat(community_id, self.chat_payload)
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
        assert member_msgs_resp.get("messages") is not None, "Member should have messages after receiving signal"
        message_text = member_msgs_resp.get("messages")[0].get("text")
        assert message_text == text, "Member should have received the message"

        logging.info(f"Waiting {message_archive_interval + 10}s for community owner to create archive...")

        # Wait for the community archive to be created for the community owner
        self.creator.wait_for_signal(SignalType.COMMUNITY_HISTORY_ARCHIVES_CREATED.value, timeout=message_archive_interval + 10)

        logging.info("Checking that community owner has local index CID file...")

        # Ensure that the community archive index exists in the file system of the community owner.
        # We test this by checking the corresponding archive index CID file exists.
        # This index CID file contains the Codex CID of the archive index.
        has_archive_index = self.creator.wakuext_service.has_community_archive(community_id)
        assert has_archive_index is True, "Creator should have community archive index after messages are sent"

        logging.info("Success! History archive created and dispatched!")

        # The timeout is arbitrary set to 10 seconds
        # We need to wait for the archive dispatch + download + import which should not take more than 10 seconds
        archive_timeout = 10

        logging.info("Waiting for community member to download manifest of the archive index...")
        # Wait for the community member to download the archive index manifest
        self.member.wait_for_signal(SignalType.COMMUNITY_ARCHIVE_MANIFEST_FETCHED.value, timeout=archive_timeout)
        logging.info("Success! Manifest of the archive index fetched!")

        # When wait for index download completed signal - at this stage the index and index CID files
        # should both exist in the file system of the member.
        logging.info("Waiting for community member to download archive index...")
        self.member.wait_for_signal(SignalType.COMMUNITY_ARCHIVE_INDEX_DOWNLOAD_COMPLETED.value, timeout=archive_timeout)
        logging.info("Success! Archive index downloaded!")

        # Ensure that the community archive index CID file exists in the file system for the member.
        # After successfully downloading the archive index, its CID is stored in the the
        # index CID file and the file is written immediately after the archive index has been downloaded.
        # Notice that at this stage, the node still does not have any single archive downloaded.
        logging.info("Verifying that community member has index CID file...")
        has_archive_index = self.member.wakuext_service.has_community_archive(community_id)
        assert has_archive_index is True, "Member should have community archive index after messages are sent"
        logging.info("Success! Community member has index CID file!")

        # Wait for the community archives to be downloaded for the first member.
        logging.info("Waiting for community member to download ALL history archives...")
        self.member.wait_for_signal(SignalType.COMMUNITY_HISTORY_ARCHIVES_SEEDING.value, timeout=archive_timeout)
        logging.info("Success! Community member has downloaded ALL history archives!")

        # Once the historyArchivesSeeding signal is received, the database
        # should be already updated: archive ID (HASH) should be stored in the database.
        logging.info("Verifying that archive ID (HASH) is recorded in the database...")
        download_archive_ids = self.member.wakuext_service.get_downloaded_message_archive_ids(community_id)
        assert len(download_archive_ids) == 1, "Member should have exactly 1 archive ID downloaded"
        logging.info("Success! Archive ID (HASH) is recorded in the database!")

        # Note: We don't check get_message_archive_ids_to_import here because archives are automatically
        # imported in the background, and by the time we check, they might already be marked as imported.
        # The important thing is that the archive was downloaded (checked above) and will be imported.

        # Wait for another member to join the community
        logging.info("Another member is joining the community...")
        self.join_community(member=self.another_member, admin=self.creator)
        self.another_member.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=chat_id, timeout=10)
        logging.info("Another member has joined the community.")

        # Ensure that another member does not have the message before archive import
        member_msgs_resp = self.another_member.wakuext_service.chat_messages(chat_id)
        message = member_msgs_resp.get("messages")
        assert message is None, "Another member should not have messages before archive is dispatched, downloaded and imported"
        logging.info("Verified that another member does not have the message before archive import.")

        # Wait for another community member to download the archive index manifest
        logging.info("Waiting for another member to download manifest of the archive index...")
        self.another_member.wait_for_signal(SignalType.COMMUNITY_ARCHIVE_MANIFEST_FETCHED.value, timeout=archive_timeout)
        logging.info("Success! Manifest of the archive index fetched by another member!")

        # Then wait for index download completed signal - at this stage the index and index CID files
        # should both exist in the file system of another member.
        logging.info("Waiting for another member to download archive index...")
        self.another_member.wait_for_signal(SignalType.COMMUNITY_ARCHIVE_INDEX_DOWNLOAD_COMPLETED.value, timeout=archive_timeout)
        logging.info("Success! Archive index downloaded by another member!")

        # Ensure that the community archive index exists in the file system of another member
        logging.info("Verifying that another member has index CID file...")
        has_archive_index = self.another_member.wakuext_service.has_community_archive(community_id)
        assert has_archive_index is True, "Another member should have community archive index after messages are sent"
        logging.info("Success! Another member has index CID file.")

        # Wait for the community archives to be downloaded for another member.
        logging.info("Waiting for another member to download ALL history archives...")
        self.another_member.wait_for_signal(SignalType.COMMUNITY_HISTORY_ARCHIVES_SEEDING.value, timeout=archive_timeout)
        logging.info("Success! Another member has downloaded ALL history archives!")

        # Wait for the archive to be downloaded by another member
        # self.another_member.wait_for_signal(SignalType.COMMUNITY_ARCHIVE_DOWNLOAD_COMPLETED.value, timeout=archive_timeout)

        # Ensure that another member has downloaded the community archive and stored its ID in database
        logging.info("Verifying that another member has archive ID (HASH) recorded in the database...")
        download_archive_ids = self.another_member.wakuext_service.get_downloaded_message_archive_ids(community_id)
        assert len(download_archive_ids) == 1, "Another member should have exactly 1 archive ID downloaded"
        logging.info("Success! Another member has archive ID (HASH) recorded in the database!")

        # Note: Same as above - archives are automatically imported, so we skip checking
        # get_message_archive_ids_to_import to avoid racing with the import process.

        # Wait for the archive import to begin for another member
        logging.info("Waiting for another member to start importing history archive messages...")
        self.another_member.wait_for_signal(SignalType.COMMUNITY_IMPORTING_HISTORY_ARCHIVE_MESSAGES_STARTED.value, timeout=archive_timeout)
        logging.info("Another member has started importing history archive messages.")

        # Wait for the archive import to complete for another member
        logging.info("Waiting for another member to finish importing history archive messages...")
        self.another_member.wait_for_signal(SignalType.COMMUNITY_IMPORTING_HISTORY_ARCHIVE_MESSAGES_FINISHED.value, timeout=archive_timeout)
        logging.info("Another member has finished importing history archive messages.")

        # Verify that another member has the message after archive import
        logging.info("Verifying that another member has the message after archive import...")
        another_member_msgs_resp = self.another_member.wakuext_service.chat_messages(chat_id)
        assert another_member_msgs_resp.get("messages") is not None, "Another member should have messages after importing history archive"
        assert (
            another_member_msgs_resp.get("messages")[0].get("text") == text
        ), "Another member should have the message after importing history archive"
        logging.info("Success! Another member has the message after importing history archive.")

    def test_community_archive_exists_for_default_chat(self):
        message_archive_interval = 10
        self.creator.wakuext_service.update_message_archive_interval(message_archive_interval)

        # Create a community
        response = self.creator.wakuext_service.create_community("Codex community", "No one should join", history_archive_support_enabled=True)
        community_id = response.get("communities", [{}])[0].get("id")
        default_chat_id = response.get("chats", [{}])[0].get("id")

        # Ensure that no community archive exists initially
        has_archive_index = self.creator.wakuext_service.has_community_archive(community_id)
        assert has_archive_index is False, "Creator should not have community archive initially"

        # Send a message to the default community chat
        text = "Hi myself!"
        send_resp = self.creator.wakuext_service.send_chat_message(default_chat_id, text)
        assert send_resp.get("chats")[0].get("lastMessage").get("text") == text

        # Wait for the community archive to be created for the community owner
        self.creator.wait_for_signal(SignalType.COMMUNITY_HISTORY_ARCHIVES_CREATED.value, timeout=message_archive_interval + 10)

        # Ensure that the community archive exists in the file system for the community owner
        has_archive = self.creator.wakuext_service.has_community_archive(community_id)
        assert has_archive is True, "Creator should have community archive after messages are sent"

    def test_archive_is_not_created_without_messages(self):
        message_archive_interval = 10
        self.creator.wakuext_service.update_message_archive_interval(message_archive_interval)

        # Create a community
        response = self.creator.wakuext_service.create_community("Codex community", "No one should join", history_archive_support_enabled=True)
        community_id = response.get("communities", [{}])[0].get("id")

        time.sleep(message_archive_interval + 10)

        # Ensure that the community archive exists in the file system for the community owner
        has_archive = self.creator.wakuext_service.has_community_archive(community_id)
        assert has_archive is False, "Creator should not have community archive without message"

    def test_different_archives_are_created_with_multiple_messages(self):
        message_archive_interval = 10
        self.creator.wakuext_service.update_message_archive_interval(message_archive_interval)

        community_id = self.create_community(self.creator, history_archive_support_enabled=True)

        # Create community chat
        create_resp = self.creator.wakuext_service.create_community_chat(community_id, self.chat_payload)
        chat_id = create_resp.get("chats")[0].get("id")

        # Join community so the member can receive messages
        self.join_community(member=self.member, admin=self.creator)

        for i in range(2):
            # Send a message to the default community chat
            text = f"Hi @{self.member.public_key}!"
            self.creator.wakuext_service.send_chat_message(chat_id, text)

            self.creator.wait_for_signal(SignalType.COMMUNITY_HISTORY_ARCHIVES_CREATED.value, timeout=message_archive_interval + 10)

            # The timeout is arbitrary set to 10 seconds
            # We need to wait for the archive dispatch + download + import which should not take more than 10 seconds
            archive_timeout = 10

            # When wait for index download completed signal - at this stage the index and index CID files
            # should both exist in the file system of the member.
            logging.info("Waiting for community member to download archive index...")
            self.member.wait_for_signal(SignalType.COMMUNITY_ARCHIVE_INDEX_DOWNLOAD_COMPLETED.value, timeout=archive_timeout)
            logging.info("Success! Archive index downloaded!")

            # Ensure that the community archive index CID file exists in the file system for the member.
            # After successfully downloading the archive index, its CID is stored in the the
            # index CID file and the file is written immediately after the archive index has been downloaded.
            # Notice that at this stage, the node still does not have any single archive downloaded.
            logging.info("Verifying that community member has index CID file...")
            has_archive_index = self.member.wakuext_service.has_community_archive(community_id)
            assert has_archive_index is True, "Member should have community archive index after messages are sent"
            logging.info("Success! Community member has index CID file!")

            # Wait for the community archives to be downloaded for the first member.
            logging.info("Waiting for community member to download ALL history archives...")
            self.member.wait_for_signal(SignalType.COMMUNITY_HISTORY_ARCHIVES_SEEDING.value, timeout=archive_timeout)
            logging.info("Success! Community member has downloaded ALL history archives!")

            # Once the historyArchivesSeeding signal is received, the database
            # should be already updated: archive ID (HASH) should be stored in the database.
            logging.info("Verifying that archive ID (HASH) is recorded in the database...")
            download_archive_ids = self.member.wakuext_service.get_downloaded_message_archive_ids(community_id)
            assert len(download_archive_ids) == i + 1, "Member should have exactly 1 archive ID downloaded"
            logging.info("Success! Archive ID (HASH) is recorded in the database!")

    def test_archive_is_downloaded_after_logout_login(self):
        message_archive_interval = 10
        self.creator.wakuext_service.update_message_archive_interval(message_archive_interval)

        community_id = self.create_community(self.creator, history_archive_support_enabled=True)

        # Ensure that no community archive exists initially
        has_archive_index = self.creator.wakuext_service.has_community_archive(community_id)
        assert has_archive_index is False, "Creator should not have community archive initially"
        has_archive_index = self.member.wakuext_service.has_community_archive(community_id)
        assert has_archive_index is False, "Member should not have community archive initially"

        create_resp = self.creator.wakuext_service.create_community_chat(community_id, self.chat_payload)
        chat_id = create_resp.get("chats")[0].get("id")

        # Wait for member to receive chat creation signal
        self.join_community(member=self.member, admin=self.creator)
        self.member.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=chat_id, timeout=10)

        # Send a message to the community chat
        text = f"Hi @{self.member.public_key}"
        send_resp = self.creator.wakuext_service.send_chat_message(chat_id, text)
        message_id = send_resp.get("messages", [])[0].get("id", "")

        # Wait for member to receive the new message
        self.member.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=message_id, timeout=10)

        # Logout the member to simulate offline scenario
        key_uid = str(self.member.key_uid)
        self.member.logout()
        self.member.wait_for_logout()

        logging.info(f"Waiting {message_archive_interval + 10}s for community owner to create archive...")
        # Wait for the community archive to be created for the community owner
        self.creator.wait_for_signal(SignalType.COMMUNITY_HISTORY_ARCHIVES_CREATED.value, timeout=message_archive_interval + 10)
        logging.info("Success! History archive created and dispatched!")

        # Login the member back
        self.member.login(key_uid)
        self.member.wait_for_login()
        self.member.wakuext_service.start_messenger()

        # Re-connect member to community codex client
        info = self.creator.wakuext_service.debug()
        self.member.wakuext_service.connect(info["id"], info["addrs"])

        # The timeout is arbitrary set to 20 seconds
        # We need to wait for the archive dispatch + download + import which should not take more than 10 seconds
        archive_timeout = 20

        logging.info("Waiting for community member to download manifest of the archive index...")
        # Wait for the community member to download the archive index manifest
        self.member.wait_for_signal(SignalType.COMMUNITY_ARCHIVE_MANIFEST_FETCHED.value, timeout=archive_timeout)
        logging.info("Success! Manifest of the archive index fetched!")

        # When wait for index download completed signal - at this stage the index and index CID files
        # should both exist in the file system of the member.
        logging.info("Waiting for community member to download archive index...")
        self.member.wait_for_signal(SignalType.COMMUNITY_ARCHIVE_INDEX_DOWNLOAD_COMPLETED.value, timeout=archive_timeout)
        logging.info("Success! Archive index downloaded!")

        # Ensure that the community archive index CID file exists in the file system for the member.
        # After successfully downloading the archive index, its CID is stored in the the
        # index CID file and the file is written immediately after the archive index has been downloaded.
        # Notice that at this stage, the node still does not have any single archive downloaded.
        logging.info("Verifying that community member has index CID file...")
        has_archive_index = self.member.wakuext_service.has_community_archive(community_id)
        assert has_archive_index is True, "Member should have community archive index after messages are sent"
        logging.info("Success! Community member has index CID file!")
