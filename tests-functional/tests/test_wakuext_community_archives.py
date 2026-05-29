import logging
from uuid import uuid4
import pytest
import time

from clients.signals import SignalType
from steps import messenger


@pytest.mark.rpc
@pytest.mark.logos_storage
class TestCommunityArchives:
    @pytest.fixture()
    def creator(self, backend_new_profile):
        node = backend_new_profile("creator")
        node.wakuext_service.enable_logos_storage_community_history_archive_protocol({"NodeConfig.DiscoveryPort": "8091"})
        return node

    @pytest.fixture()
    def member(self, backend_new_profile, creator):
        node = backend_new_profile("member", import_initial_delay=5)
        info = creator.wakuext_service.debug()
        node.wakuext_service.enable_logos_storage_community_history_archive_protocol(
            {
                "NodeConfig.DiscoveryPort": 8092,
                "NodeConfig.BootstrapNodes": f'["{info["spr"]}"]',
            }
        )
        # Using bootstrap nodes does not seem to be working in our setup,
        # thus we need to connect members manually.
        node.wakuext_service.connect(info["id"], info["addrs"])
        return node

    @pytest.fixture()
    def another_member(self, backend_new_profile, creator):
        node = backend_new_profile("member", import_initial_delay=5)
        info = creator.wakuext_service.debug()
        node.wakuext_service.enable_logos_storage_community_history_archive_protocol(
            {
                "NodeConfig.DiscoveryPort": 8093,
                "NodeConfig.BootstrapNodes": f'["{info["spr"]}"]',
            }
        )
        # Using bootstrap nodes does not seem to be working in our setup,
        # thus we need to connect members manually.
        node.wakuext_service.connect(info["id"], info["addrs"])
        return node

    @pytest.fixture()
    def chat_payload(self):
        display_name = "chat_" + str(uuid4())
        return {
            "identity": {
                "displayName": display_name,
                "emoji": "😀",
                "color": "#1f2c75",
                "description": display_name,
            },
            "viewersCanPostReactions": False,
            "hideIfPermissionsNotMet": False,
            "permissions": {"access": 1},
        }

    def test_community_archive_index_exists(self, creator, member, another_member, chat_payload):
        # Set message archive interval to 80 seconds which is longer that the retention policy
        # of the Waku node.
        # So we are expecting to retrieve the archive from LogosStorage even after the Waku node
        # has already deleted the messages locally.
        message_archive_interval = 80
        creator.wakuext_service.update_message_archive_interval(message_archive_interval)

        community_id = messenger.create_community(creator, history_archive_support_enabled=True)

        # Ensure that no community archive exists initially
        has_archive_index = creator.wakuext_service.has_community_archive(community_id)
        assert has_archive_index is False, "Creator should not have community archive initially"
        has_archive_index = member.wakuext_service.has_community_archive(community_id)
        assert has_archive_index is False, "Member should not have community archive initially"
        has_archive_index = another_member.wakuext_service.has_community_archive(community_id)
        assert has_archive_index is False, "Another member should not have community archive initially"

        # Create community chat
        create_resp = creator.wakuext_service.create_community_chat(community_id, chat_payload)
        chat_id = create_resp.get("chats")[0].get("id")

        # Wait for member to receive chat creation signal
        with member.expect_signal(SignalType.MESSAGES_NEW.value, timeout=10, pattern=chat_id):
            messenger.join_community(
                member=member,
                admin=creator,
                community_id=community_id,
                prejoin_warmup=False,
            )

        # Send a message to the community chat
        text = f"Hi @{member.public_key}"
        send_resp = creator.wakuext_service.send_chat_message(chat_id, text)
        assert send_resp.get("chats")[0].get("lastMessage").get("text") == text
        message_id = send_resp.get("messages", [])[0].get("id", "")

        # Wait for member to receive the new message
        with member.expect_signal(SignalType.MESSAGES_NEW.value, timeout=10, pattern=message_id):
            pass
        member_msgs_resp = member.wakuext_service.chat_messages(chat_id)
        assert member_msgs_resp.get("messages") is not None, "Member should have messages after receiving signal"
        message_text = member_msgs_resp.get("messages")[0].get("text")
        assert message_text == text, "Member should have received the message"

        logging.info(f"Waiting {message_archive_interval + 10}s for community owner to create archive...")

        # Wait for the community archive to be created for the community owner
        with creator.expect_signal(
            SignalType.COMMUNITY_HISTORY_ARCHIVES_CREATED.value,
            timeout=message_archive_interval + 10,
        ):
            pass

        logging.info("Checking that community owner has local index CID file...")

        # Ensure that the community archive index is being seeded by the community owner.
        # has_community_archive returns true if lastSeenIndexCid from DB is not empty
        # and HasCid on the LogosStorageClient returns true.
        has_archive_index = creator.wakuext_service.has_community_archive(community_id)
        assert has_archive_index is True, "Creator should have community archive index after messages are sent"

        logging.info("Success! History archive created and dispatched!")

        # The timeout is arbitrary set to 10 seconds
        # We need to wait for the archive dispatch + download + import which should not take more than 10 seconds
        archive_timeout = 10

        # Wait for index download completed signal. This signal is emitted
        # immediately after archive index is downloaded from LogosStorage node.
        logging.info("Waiting for community member to download archive index...")
        with member.expect_signal(
            SignalType.COMMUNITY_ARCHIVE_INDEX_DOWNLOAD_COMPLETED.value,
            timeout=archive_timeout,
            start="beginning",
            predicate=lambda signal: (signal.get("event", {}).get("communityId") == community_id),
        ):
            pass
        logging.info("Success! Archive index downloaded!")

        # The HistoryArchivesSeedingSignal is emitted right after all archives
        # are downloaded to the LogosStorage node and the corresponding index CID is
        # recorded in the database as "lastSeenIndexCid".
        logging.info("Waiting for community member to download ALL history archives...")
        with member.expect_signal(
            SignalType.COMMUNITY_HISTORY_ARCHIVES_SEEDING.value,
            timeout=archive_timeout,
            start="beginning",
            predicate=lambda signal: (signal.get("event", {}).get("communityId") == community_id),
        ):
            pass
        logging.info("Success! Community member has downloaded ALL history archives!")

        # The archive index should be "seeding": index CID in the database and
        # HasCid on the LogosStorageClient returns true.
        logging.info("Verifying that community member has index CID file...")
        has_archive_index = member.wakuext_service.has_community_archive(community_id)
        assert has_archive_index is True, "Member should have community archive index after messages are sent"
        logging.info("Success! Community member has index CID file!")

        # Once the historyArchivesSeeding signal is received, the database
        # should be already updated: archive ID (HASH) should be stored in the database.
        logging.info("Verifying that archive ID (HASH) is recorded in the database...")
        download_archive_ids = member.wakuext_service.get_downloaded_message_archive_ids(community_id)
        assert len(download_archive_ids) == 1, "Member should have exactly 1 archive ID downloaded"
        logging.info("Success! Archive ID (HASH) is recorded in the database!")

        # Note: We don't check get_message_archive_ids_to_import here because archives are automatically
        # imported in the background, and by the time we check, they might already be marked as imported.
        # The important thing is that the archive was downloaded (checked above) and will be imported.

        # Wait for another member to join the community
        logging.info("Another member is joining the community...")
        with another_member.expect_signal(SignalType.MESSAGES_NEW.value, timeout=10, pattern=chat_id):
            messenger.join_community(
                member=another_member,
                admin=creator,
                community_id=community_id,
                prejoin_warmup=False,
            )
        logging.info("Another member has joined the community.")

        # Ensure that another member does not have the message before archive import
        member_msgs_resp = another_member.wakuext_service.chat_messages(chat_id)
        message = member_msgs_resp.get("messages")
        assert message is None, "Another member should not have messages before archive is dispatched, downloaded and imported"
        logging.info("Verified that another member does not have the message before archive import.")

        # Wait for index download completed signal - index should be now downloaded
        # for LogosStorage.
        logging.info("Waiting for another member to download archive index...")
        with another_member.expect_signal(
            SignalType.COMMUNITY_ARCHIVE_INDEX_DOWNLOAD_COMPLETED.value,
            timeout=archive_timeout,
            start="beginning",
            predicate=lambda signal: (signal.get("event", {}).get("communityId") == community_id),
        ):
            pass
        logging.info("Success! Archive index downloaded by another member!")

        # Wait for seeding signal - all archives should be now downloaded to LogosStorage node
        # and index should be seeding.
        logging.info("Waiting for another member to download ALL history archives...")
        with another_member.expect_signal(
            SignalType.COMMUNITY_HISTORY_ARCHIVES_SEEDING.value,
            timeout=archive_timeout,
            start="beginning",
            predicate=lambda signal: (signal.get("event", {}).get("communityId") == community_id),
        ):
            pass
        logging.info("Success! Another member has downloaded ALL history archives!")

        # IndexCid in the database and HasCid on the LogosStorageClient returns true (seeding).
        logging.info("Verifying that another member has index CID file...")
        has_archive_index = another_member.wakuext_service.has_community_archive(community_id)
        assert has_archive_index is True, "Another member should have community archive index after messages are sent"
        logging.info("Success! Another member has index CID file.")

        # Ensure that another member has downloaded the community archive and stored its ID in database
        logging.info("Verifying that another member has archive ID (HASH) recorded in the database...")
        download_archive_ids = another_member.wakuext_service.get_downloaded_message_archive_ids(community_id)
        assert len(download_archive_ids) == 1, "Another member should have exactly 1 archive ID downloaded"
        logging.info("Success! Another member has archive ID (HASH) recorded in the database!")

        # Note: Same as above - archives are automatically imported, so we skip checking
        # get_message_archive_ids_to_import to avoid racing with the import process.

        # Wait for the archive import to begin for another member
        logging.info("Waiting for another member to start importing history archive messages...")
        with another_member.expect_signal(
            SignalType.COMMUNITY_IMPORTING_HISTORY_ARCHIVE_MESSAGES_STARTED.value,
            timeout=archive_timeout,
            start="beginning",
            predicate=lambda signal: (signal.get("event", {}).get("communityId") == community_id),
        ):
            pass
        logging.info("Another member has started importing history archive messages.")

        # Wait for the archive import to complete for another member
        logging.info("Waiting for another member to finish importing history archive messages...")
        with another_member.expect_signal(
            SignalType.COMMUNITY_IMPORTING_HISTORY_ARCHIVE_MESSAGES_FINISHED.value,
            timeout=archive_timeout,
            start="beginning",
            predicate=lambda signal: (signal.get("event", {}).get("communityId") == community_id),
        ):
            pass
        logging.info("Another member has finished importing history archive messages.")

        # Verify that another member has the message after archive import
        logging.info("Verifying that another member has the message after archive import...")
        another_member_msgs_resp = another_member.wakuext_service.chat_messages(chat_id)
        assert another_member_msgs_resp.get("messages") is not None, "Another member should have messages after importing history archive"
        assert (
            another_member_msgs_resp.get("messages")[0].get("text") == text
        ), "Another member should have the message after importing history archive"
        logging.info("Success! Another member has the message after importing history archive.")

    def test_community_archive_exists_for_default_chat(self, creator):
        # Set message archive interval to 10 seconds for faster test,
        # we only want to check that the archive is created for the default chat.
        message_archive_interval = 10
        creator.wakuext_service.update_message_archive_interval(message_archive_interval)

        # Create a community
        response = creator.wakuext_service.create_community(
            "LogosStorage community",
            "No one should join",
            history_archive_support_enabled=True,
        )
        community_id = response.get("communities", [{}])[0].get("id")
        default_chat_id = response.get("chats", [{}])[0].get("id")

        # Ensure that no community archive exists initially
        has_archive_index = creator.wakuext_service.has_community_archive(community_id)
        assert has_archive_index is False, "Creator should not have community archive initially"

        # Send a message to the default community chat
        text = "Hi myself!"
        send_resp = creator.wakuext_service.send_chat_message(default_chat_id, text)
        assert send_resp.get("chats")[0].get("lastMessage").get("text") == text

        # Wait for the community archive to be created for the community owner
        with creator.expect_signal(
            SignalType.COMMUNITY_HISTORY_ARCHIVES_CREATED.value,
            timeout=message_archive_interval + 10,
        ):
            pass

        # Ensure that the archive index is seeding.
        has_archive = creator.wakuext_service.has_community_archive(community_id)
        assert has_archive is True, "Creator should have community archive after messages are sent"

    def test_archive_is_not_created_without_messages(self, creator):
        # Set message archive interval to 10 seconds for faster test,
        # we only want to check that no archive is created when there is no message.
        message_archive_interval = 10
        creator.wakuext_service.update_message_archive_interval(message_archive_interval)

        # Create a community
        response = creator.wakuext_service.create_community(
            "LogosStorage community",
            "No one should join",
            history_archive_support_enabled=True,
        )
        community_id = response.get("communities", [{}])[0].get("id")

        time.sleep(message_archive_interval + 10)

        # Ensure that no archive index is seeding.
        has_archive = creator.wakuext_service.has_community_archive(community_id)
        assert has_archive is False, "Creator should not have community archive without message"

    def test_different_archives_are_created_with_multiple_messages(self, creator, member, chat_payload):
        # Set message archive interval to 10 seconds for faster test.
        # We want to check that different archives are created for multiple messages,
        # so it does not matter if the Waku stode node has the messages locally.
        message_archive_interval = 10
        creator.wakuext_service.update_message_archive_interval(message_archive_interval)

        community_id = messenger.create_community(creator, history_archive_support_enabled=True)

        # Create community chat
        create_resp = creator.wakuext_service.create_community_chat(community_id, chat_payload)
        chat_id = create_resp.get("chats")[0].get("id")

        # Join community so the member can receive messages
        messenger.join_community(
            member=member,
            admin=creator,
            community_id=community_id,
            prejoin_warmup=False,
        )

        for i in range(2):
            # Send a message to the default community chat
            text = f"Hi @{member.public_key}!"
            creator.wakuext_service.send_chat_message(chat_id, text)

            with creator.expect_signal(
                SignalType.COMMUNITY_HISTORY_ARCHIVES_CREATED.value,
                timeout=message_archive_interval + 10,
                start="now",
            ):
                pass

            # The timeout is arbitrary set to 10 seconds
            # We need to wait for the archive dispatch + download + import which should not take more than 10 seconds
            archive_timeout = 10

            # Wait for the archive index to be downloaded from LogosStorage node.
            logging.info("Waiting for community member to download archive index...")
            with member.expect_signal(
                SignalType.COMMUNITY_ARCHIVE_INDEX_DOWNLOAD_COMPLETED.value,
                count=i + 1,
                timeout=archive_timeout,
                start="beginning",
                predicate=lambda signal: (signal.get("event", {}).get("communityId") == community_id),
            ):
                pass
            logging.info("Success! Archive index downloaded!")

            # Wait for the seeding signal.
            logging.info("Waiting for community member to download ALL history archives...")
            with member.expect_signal(
                SignalType.COMMUNITY_HISTORY_ARCHIVES_SEEDING.value,
                count=i + 1,
                timeout=archive_timeout,
                start="beginning",
                predicate=lambda signal: (signal.get("event", {}).get("communityId") == community_id),
            ):
                pass
            logging.info("Success! Community member has downloaded ALL history archives!")

            # The archive index should be "seeding": index CID in the database and
            # HasCid on the LogosStorageClient returns true.
            logging.info("Verifying that community member has index CID file...")
            has_archive_index = member.wakuext_service.has_community_archive(community_id)
            assert has_archive_index is True, "Member should have community archive index after messages are sent"
            logging.info("Success! Community member has index CID file!")

            # Once the historyArchivesSeeding signal is received, the database
            # should be already updated: archive ID (HASH) should be stored in the database.
            logging.info("Verifying that archive ID (HASH) is recorded in the database...")
            download_archive_ids = member.wakuext_service.get_downloaded_message_archive_ids(community_id)
            assert len(download_archive_ids) == i + 1, f"Member should have exactly {i + 1} archive IDs downloaded"
            logging.info("Success! Archive ID (HASH) is recorded in the database!")

    def test_archive_is_downloaded_after_logout_login(self, creator, member, chat_payload):
        # Set message archive interval to 10 seconds for faster test.
        # We want to check that the archive is downloaded after logout/login,
        # so it does not matter if the Waku stode node has the messages locally.

        message_archive_interval = 10
        creator.wakuext_service.update_message_archive_interval(message_archive_interval)

        community_id = messenger.create_community(creator, history_archive_support_enabled=True)

        # Ensure that no community archive exists initially
        has_archive_index = creator.wakuext_service.has_community_archive(community_id)
        assert has_archive_index is False, "Creator should not have community archive initially"
        has_archive_index = member.wakuext_service.has_community_archive(community_id)
        assert has_archive_index is False, "Member should not have community archive initially"

        create_resp = creator.wakuext_service.create_community_chat(community_id, chat_payload)
        chat_id = create_resp.get("chats")[0].get("id")

        # Wait for member to receive chat creation signal
        with member.expect_signal(SignalType.MESSAGES_NEW.value, timeout=10, pattern=chat_id):
            messenger.join_community(
                member=member,
                admin=creator,
                community_id=community_id,
                prejoin_warmup=False,
            )

        # Send a message to the community chat
        text = f"Hi @{member.public_key}"
        send_resp = creator.wakuext_service.send_chat_message(chat_id, text)
        message_id = send_resp.get("messages", [])[0].get("id", "")

        # Wait for member to receive the new message
        with member.expect_signal(SignalType.MESSAGES_NEW.value, timeout=10, pattern=message_id):
            pass

        # Logout the member to simulate offline scenario
        key_uid = str(member.key_uid)
        password = member.password
        logging.info("Waiting for NODE_STOPPED signal...")
        # Arm the expectation before logout to avoid matching stale NODE_STOPPED from earlier lifecycle events.
        with member.expect_signal(SignalType.NODE_STOPPED.value, timeout=30, start="now"):
            member.logout()
        logging.info("NODE_STOPPED signal received.")

        logging.info(f"Waiting {message_archive_interval + 10}s for community owner to create archive...")
        # Wait for the community archive to be created for the community owner
        with creator.expect_signal(
            SignalType.COMMUNITY_HISTORY_ARCHIVES_CREATED.value,
            timeout=message_archive_interval + 10,
        ):
            pass
        logging.info("Success! History archive created and dispatched!")

        # Login the member back
        member.login(key_uid, password)
        member.wait_for_login()
        member.wait_for_wakuext_ready(timeout=30, start_messenger=False)
        member.wakuext_service.start_messenger()

        # Re-connect member to community LogosStorage client
        info = creator.wakuext_service.debug()
        member.wakuext_service.connect(info["id"], info["addrs"])

        # The timeout is arbitrary set to 20 seconds
        # We need to wait for the archive dispatch + download + import which should not take more than 10 seconds
        archive_timeout = 20

        # When wait for index download completed signal.
        logging.info("Waiting for community member to download archive index...")
        with member.expect_signal(
            SignalType.COMMUNITY_ARCHIVE_INDEX_DOWNLOAD_COMPLETED.value,
            timeout=archive_timeout,
            start="beginning",
            predicate=lambda signal: (signal.get("event", {}).get("communityId") == community_id),
        ):
            pass
        logging.info("Success! Archive index downloaded!")

        # Wait for the seeding signal.
        logging.info("Waiting for community member to download ALL history archives...")
        with member.expect_signal(
            SignalType.COMMUNITY_HISTORY_ARCHIVES_SEEDING.value,
            timeout=archive_timeout,
            start="beginning",
            predicate=lambda signal: (signal.get("event", {}).get("communityId") == community_id),
        ):
            pass
        logging.info("Success! Community member has downloaded ALL history archives!")

        # Confirm that the archive index is seeding.
        logging.info("Verifying that community member has index CID file...")
        has_archive_index = member.wakuext_service.has_community_archive(community_id)
        assert has_archive_index is True, "Member should have community archive index after messages are sent"
        logging.info("Success! Community member has index CID file!")
