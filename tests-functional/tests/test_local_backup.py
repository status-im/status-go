import os
import re

import pytest

from clients.status_backend import StatusBackend
from resources.constants import Account, user_1
from resources.test_data import profile_showcase_utils
from utils import fake
from clients.api import ApiResponseError

test_one_to_one_chat_id = (
    "0x043329fc08727f15c4ec9a17bf7d6a3dc44e9d0d8f782a55804f3660b28827194a365dc1765cf96b6fa5cd666b9ee5298e8ac82f51f7952c4110cbf321d4f63864"
)


@pytest.mark.rpc
class TestLocalBackup:

    @pytest.mark.init
    def test_local_backup(self, tmp_path):
        backend_client = StatusBackend()
        assert backend_client is not None

        # Init and login
        backend_client.init_status_backend()
        backend_client.create_account_and_login(password=fake.profile_password())
        backend_client.wait_for_login()
        backend_client.wakuext_service.start_messenger()

        assert backend_client.mnemonic != ""

        user_mnemonic = backend_client.mnemonic

        # Change settings
        backend_client.wakuext_service.set_display_name("myname")

        # Move image to the container and apply it to the first backend client
        backend_client.import_data(
            os.path.abspath(os.path.join(os.path.dirname(__file__), "../resources/images/test-image-200x200.jpg")), "/tmp/images/"
        )
        backend_client.multiaccounts_service.store_identity_image(backend_client.key_uid, "/tmp/images/test-image-200x200.jpg", 0, 0, 200, 200)

        backend_client.settings_service.save_setting("bio", "new bio")
        backend_client.settings_service.save_setting("preferred-name", "new-prefered-name")
        backend_client.settings_service.save_setting("url-unfurling-mode", 2)
        backend_client.settings_service.save_setting("messages-from-contacts-only", True)
        backend_client.settings_service.save_setting("currency", "eth")
        backend_client.settings_service.save_setting("mnemonic", "")  # Clear mnemonic setting

        profile_showcase_preferences = profile_showcase_utils.dummy_profile_showcase_preferences(False)
        backend_client.wakuext_service.set_profile_showcase_preferences(profile_showcase_preferences)

        # Add watchonly account
        backend_client.accounts_service.add_account(
            "",
            {
                "address": "0xE04DC5f84918F567E9Fc8D0c38B26E3A7D013f7b",
                "key-uid": "",
                "wallet": False,
                "chat": False,  # this refers to Status chat account, set when the Status account is created, cannot be changed later
                "type": "watch",
                "path": "",
                "public-key": "",
                "name": "test-watchonly-account",
                "emoji": "🕵️",
                "colorId": "green",
                "hidden": True,
            },
        )

        # Add contacts
        backend_client.wakuext_service.send_contact_request("zQ3shnZxb3McGZqKgKYBioaRUzJACqoes1m2ivYPBRAWEnVR2", "Test message")
        backend_client.wakuext_service.send_contact_request("zQ3shVMbHwBhLapryZRYhJh6Temcpd3sqZH2X8zXkCsoHjFjK", "Test message")

        # Create a community
        backend_client.wakuext_service.create_community_from_payload(
            {
                "name": "Test Community",
                "description": "This is a test community",
                "image": "",
                "imageAx": 0,
                "imageAy": 0,
                "imageBx": 0,
                "imageBy": 0,
                "Membership": 1,
                "introMessage": "Welcome to the test community!",
                "outroMessage": "Thank you for joining the test community!",
                "ensOnly": False,
                "color": "#00FF00",
                "tags": [],
                "historyArchiveSupportEnabled": False,
                "pinMessageAllMembersEnabled": False,
            }
        )

        # Create another community
        result = backend_client.wakuext_service.create_community_from_payload(
            {
                "name": "Test Community 2",
                "description": "This is another test community",
                "image": "",
                "imageAx": 0,
                "imageAy": 0,
                "imageBx": 0,
                "imageBy": 0,
                "Membership": 1,
                "introMessage": "Welcome to the second test community!",
                "outroMessage": "Thank you for joining the second test community!",
                "ensOnly": False,
                "color": "#FF0000",
                "tags": [],
                "historyArchiveSupportEnabled": False,
                "pinMessageAllMembersEnabled": False,
            }
        )
        assert result is not None, "Community creation failed"
        communities = result.get("communities", [])
        assert communities is not None, "Community was not created (None)"
        assert len(communities) == 1, "Community was not created (empty)"
        community_id = communities[0].get("id", "")
        assert community_id != "", "Community ID was not returned"

        # Leave community
        backend_client.wakuext_service.leave_community(community_id)

        # Create chats
        result = backend_client.wakuext_service.create_group_chat_with_members([], "test-group-chat")
        assert result is not None, "Group chat creation failed"
        chats = result.get("chats", [])
        assert chats is not None, "Group chat was not created (None)"
        assert len(chats) == 1, "Group chat was not created (empty)"
        group_chat_id = chats[0].get("id", "")
        assert group_chat_id != "", "Group chat ID was not returned"

        result = backend_client.wakuext_service.create_one_to_one_chat(test_one_to_one_chat_id, "")

        assert result is not None, "One-to-one chat creation failed"
        chats = result.get("chats", [])
        assert chats is not None, "One-to-one chat was not created (None)"
        assert len(chats) == 1, "One-to-one chat was not created (empty)"

        # Validate settings on the first backend client
        result = backend_client.settings_service.get_settings()
        assert result is not None, "Settings were not set"
        assert result.get("display-name") == "myname", "Display name setting was not set"
        assert result.get("bio") == "new bio", "Bio setting was not set"
        assert result.get("preferred-name") == "new-prefered-name", "Preferred name setting was not set"
        assert result.get("url-unfurling-mode") == 2, "URL unfurling mode setting was not set"
        assert result.get("messages-from-contacts-only") is True, "Messages from contacts only setting was not set"
        assert result.get("currency") == "eth", "Currency setting was not set"
        assert result.get("mnemonic-removed?") is True, "Mnemonic setting was not set"

        # Validate profile showcase preferences
        result = backend_client.wakuext_service.get_profile_showcase_preferences()
        assert result is not None, "Profile showcase preferences were not set"
        assert result.get("clock") > 0, "Clock is 0 in the profile showcase preferences"
        profile_showcase_preferences["clock"] = result.get("clock")  # override clock for simpler comparison
        assert result == profile_showcase_preferences, "Profile showcase preferences were not set correctly"

        # Validate watchonly account
        result = backend_client.accounts_service.get_accounts()
        assert result is not None, "Accounts were not restored"
        assert len(result) == 3, "Watchonly account was not restored"
        assert result[2].get("name") == "test-watchonly-account", "Watchonly account name was not set correctly"
        assert result[2].get("emoji") == "🕵️", "Watchonly account emoji was not set correctly"
        assert result[2].get("type") == "watch", "Watchonly account type was not set correctly"

        # Validate contacts
        result = backend_client.wakuext_service.get_contacts()
        assert result is not None, "Contacts were not restored"
        assert len(result) == 2, "Contacts were not restored"

        # Validate communities
        result = backend_client.wakuext_service.communities()
        assert result is not None, "Communities were not restored"
        assert len(result) == 2, "Communities were not restored correctly"
        assert result[0].get("name") == "Test Community", "Community name was not restored correctly"
        assert result[0].get("description") == "This is a test community", "Community description was not restored correctly"
        assert result[1].get("name") == "Test Community 2", "Second community name was not restored correctly"
        assert result[1].get("description") == "This is another test community", "Second community description was not restored correctly"
        assert not result[1].get("joined"), "Second community was not left correctly"

        # Validate chats
        result = backend_client.wakuext_service.chats()
        assert result is not None, "Chats were not restored"
        group_chat_recovered = False
        one_on_one_chat_recovered = False
        for chat in result:
            if chat.get("id") == group_chat_id:
                assert chat.get("name") == "test-group-chat", "Group chat name was not restored correctly"
                group_chat_recovered = True
            elif chat.get("id") == test_one_to_one_chat_id:
                one_on_one_chat_recovered = True

        assert group_chat_recovered, "Group chat was not restored correctly"
        assert one_on_one_chat_recovered, "One-to-one chat was not restored correctly"

        # Test that it fails if no backupPath is set
        backend_client.settings_service.save_setting("backup-path", "")
        with pytest.raises(ApiResponseError, match=re.escape("backup path is not set")):
            backend_client.api_request_json("PerformLocalBackup", "")

        # Change the backup path (Docker restricts access to a lot of folders)
        backend_client.settings_service.save_setting("backup-path", "/usr/status-user/backups")

        # Perform the backup
        response = backend_client.api_request_json("PerformLocalBackup", "")
        backup_path = response.get("filePath", "")

        # Extract the backup file from the container
        backup_file = backend_client.extract_data(backup_path)
        assert backup_file is not None

        # Create a new installation
        backend_client2 = StatusBackend()
        assert backend_client2 is not None

        # Init and login
        backend_client2.init_status_backend()
        backend_client2.restore_account_and_login(
            user=Account(
                passphrase=user_mnemonic,
                password=backend_client.password,
                # The private key and address are not needed for mnemonic recovery
                private_key="",
                address="",
            )
        )
        backend_client2.wait_for_login()
        backend_client2.wakuext_service.start_messenger()

        # Import the backup file into the new container
        backend_client2.import_data(backup_file, "/usr/status-user/backups")

        # Import the backup file
        backend_client2.api_request_json("LoadLocalBackup", {"filePath": backup_path})

        # Validate profile settings
        result = backend_client2.multiaccounts_service.get_identity_images(backend_client2.key_uid)
        assert result is not None, "Identity images were not restored"
        assert len(result) == 2, "Identity image was not restored"

        # Validate settings
        result = backend_client2.settings_service.get_settings()
        assert result is not None, "Settings were not restored"
        # FIXME display name doesn't work because we set the display name when creating the account
        # and the clock is somehow too low, so the display name is not updated
        # assert result.get("display-name") == "myname", "Display name setting was not restored"
        assert result.get("bio") == "new bio", "Bio setting was not restored"
        assert result.get("preferred-name") == "new-prefered-name", "Preferred name setting was not restored"
        assert result.get("url-unfurling-mode") == 2, "URL unfurling mode setting was not restored"
        assert result.get("messages-from-contacts-only") is True, "Messages from contacts only setting was not restored"
        assert result.get("currency") == "eth", "Currency setting was not restored"
        assert result.get("mnemonic-removed?") is True, "Mnemonic setting was not restored"

        # Validate profile showcase preferences
        result = backend_client2.wakuext_service.get_profile_showcase_preferences()
        assert result is not None, "Profile showcase preferences were not restored"
        assert result.get("clock") > 0, "Clock is 0 in the restored profile showcase preferences"
        profile_showcase_preferences["clock"] = result.get("clock")  # override clock for simpler comparison
        assert result == profile_showcase_preferences, "Profile showcase preferences were not restored correctly"

        # Validate watchonly account
        result = backend_client2.accounts_service.get_accounts()
        assert result is not None, "Accounts were not restored"
        assert len(result) == 3, "Watchonly account was not restored"
        assert result[2].get("name") == "test-watchonly-account", "Watch only account name was not restored correctly"
        assert result[2].get("emoji") == "🕵️", "Watch only account emoji was not restored correctly"
        assert result[2].get("type") == "watch", "Watch only account type was not restored correctly"

        # Validate contacts
        result = backend_client2.wakuext_service.get_contacts()
        assert result is not None, "Contacts were not restored"
        assert len(result) == 2, "Contacts were not restored correctly"

        # Validate communities
        result = backend_client2.wakuext_service.joined_communities()
        assert result is not None, "Communities were not restored"
        assert len(result) == 1, "Communities were not restored correctly"
        assert result[0].get("name") == "Test Community", "Community name was not restored correctly"
        assert result[0].get("description") == "This is a test community", "Community description was not restored correctly"

        # Validate chats
        result = backend_client2.wakuext_service.chats()
        assert result is not None, "Chats were not restored"
        group_chat_recovered = False
        one_on_one_chat_recovered = False
        for chat in result:
            if chat.get("id") == group_chat_id:
                assert chat.get("name") == "test-group-chat", "Group chat name was not restored correctly"
                group_chat_recovered = True
            elif chat.get("id") == test_one_to_one_chat_id:
                one_on_one_chat_recovered = True
        assert group_chat_recovered, "Group chat was not restored correctly"
        assert one_on_one_chat_recovered, "One-to-one chat was not restored correctly"

    def test_local_backup_backwards_compatibility(self, tmp_path):
        backend_client = StatusBackend()
        assert backend_client is not None
        # Restore the account with a hardcoded mnemonic to get the same private key
        backend_client.init_status_backend()
        backend_client.restore_account_and_login(
            user=Account(
                # Hardcoded mnemonic for backwards compatibility test
                # Do not change this mnemonic, the backup file was created with it
                passphrase="flush okay stadium refuse shrug fit actual stairs puzzle brisk flat moral",
                password=user_1.password,
                # The private key and address are not needed for mnemonic recovery
                private_key="",
                address="",
            )
        )
        backend_client.wait_for_login()
        backend_client.wakuext_service.start_messenger()

        # Import the backup file into the container
        backend_client.import_data(
            os.path.abspath(os.path.join(os.path.dirname(__file__), "../resources/test_data/test_old_user_data.bkp")), "/usr/status-user/backups"
        )

        # Import the backup file
        # This is the old backup file that does not contain the BackedUpProfile protobuf
        # This means that the display name, identity image and profile showcase preferences will not be restored
        backend_client.api_request_json("LoadLocalBackup", {"filePath": os.path.join("/usr/status-user/backups", "test_old_user_data.bkp")})

        # This "old" backup file does not contain the identity image, so we expect it to be empty
        result = backend_client.multiaccounts_service.get_identity_images(backend_client.key_uid)
        assert result is None, "Identity images shouldn't be restored from the old backup file"

        # Profile showcase preferences were not set in the old backup file, so we expect it to be empty
        result = backend_client.wakuext_service.get_profile_showcase_preferences()
        assert result is not None, "Profile showcase preferences were not restored"
        assert result.get("clock") == 0, "Clock should be 0 as it was not set in the old backup file"
        assert result.get("accounts") is None, "Accounts should not be set in the old backup file"
        assert len(result.get("collectibles")) == 0, "Collectibles should not be set in the old backup file"
        assert result.get("communities") is None, "Communities should not be set in the old backup file"
        assert len(result.get("socialLinks")) == 0, "Social links should not be set in the old backup file"
        assert len(result.get("unverifiedTokens")) == 0, "Unverified tokens should not be set in the old backup file"
        assert len(result.get("verifiedTokens")) == 0, "Verified tokens should not be set in the old backup file"

        # The rest of the data should be restored correctly, so we can check the settings, watchonly account, contacts, communities and chats
        # Validate settings
        result = backend_client.settings_service.get_settings()
        assert result is not None, "Settings were not restored"
        assert result.get("bio") == "new bio", "Bio setting was not restored"
        assert result.get("preferred-name") == "new-prefered-name", "Preferred name setting was not restored"
        assert result.get("url-unfurling-mode") == 2, "URL unfurling mode setting was not restored"
        assert result.get("messages-from-contacts-only") is True, "Messages from contacts only setting was not restored"
        assert result.get("currency") == "eth", "Currency setting was not restored"
        assert result.get("mnemonic-removed?") is True, "Mnemonic setting was not restored"

        # Validate watchonly account
        result = backend_client.accounts_service.get_accounts()
        assert result is not None, "Accounts were not restored"
        assert len(result) == 3, "Watchonly account was not restored"
        assert result[2].get("name") == "test-watchonly-account", "Watch only account name was not restored correctly"
        assert result[2].get("emoji") == "🕵️", "Watch only account emoji was not restored correctly"
        assert result[2].get("type") == "watch", "Watch only account type was not restored correctly"

        # Validate contacts
        result = backend_client.wakuext_service.get_contacts()
        assert result is not None, "Contacts were not restored"
        assert len(result) == 2, "Contacts were not restored correctly"

        # Validate communities
        result = backend_client.wakuext_service.joined_communities()
        assert result is not None, "Communities were not restored"
        assert len(result) == 1, "Communities were not restored correctly"
        assert result[0].get("name") == "Test Community", "Community name was not restored correctly"
        assert result[0].get("description") == "This is a test community", "Community description was not restored correctly"

        # Validate chats
        result = backend_client.wakuext_service.chats()
        assert result is not None, "Chats were not restored"
        # Chat ids are hardcoded becuase they are in the backup file, so we can check them directly
        test_group_chat_id = "1fb5f429-1e23-4156-b216-2799fa430742-0x045b1f2d9bfbbeee1f6995a2086de353399074073d213079d3e0dc2ddc1593e7c50e56aead4a42e25a76effa4208f842f9356f1a5c16aaf16c179ed020277a9901"  # noqa: E501
        group_chat_recovered = False
        one_on_one_chat_recovered = False
        for chat in result:
            if chat.get("id") == test_group_chat_id:
                assert chat.get("name") == "test-group-chat", "Group chat name was not restored correctly"
                group_chat_recovered = True
            elif chat.get("id") == test_one_to_one_chat_id:
                one_on_one_chat_recovered = True
        assert group_chat_recovered, "Group chat was not restored correctly"
        assert one_on_one_chat_recovered, "One-to-one chat was not restored correctly"
