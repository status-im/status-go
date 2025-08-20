import random
import string
import copy
from clients.status_backend import StatusBackend
from clients.signals import SignalType
import pytest

from resources.constants import user_mnemonic_12, user_mnemonic_15, user_mnemonic_24, user_keycard_1
from resources.utils import assert_response_attributes


@pytest.mark.create_account
@pytest.mark.rpc
class TestBackupMnemonicAndRestore:

    await_signals = [
        SignalType.MEDIASERVER_STARTED.value,
        SignalType.NODE_STARTED.value,
        SignalType.NODE_READY.value,
        SignalType.NODE_LOGIN.value,
        SignalType.NODE_LOGOUT.value,
    ]

    @pytest.fixture(autouse=True)
    def setup_cleanup(self, close_status_backend_containers):
        """Automatically cleanup containers after each test"""
        yield

    def test_profile_creation_and_mnemonics_backup(self):
        # Create a new account container and initialize
        account = StatusBackend(self.await_signals)
        account.init_status_backend()
        account.create_account_and_login()
        account.wait_for_login()

        # Retrieve and verify the mnemonic
        settings = account.settings_service.get_settings()
        mnemonic = settings.get("result", {}).get("mnemonic", None)
        assert mnemonic is not None
        assert isinstance(mnemonic, str)
        assert len(mnemonic.split()) == 12  # Basic check for mnemonic length

    def test_backup_account_and_restore_it_via_mnemonics(self):
        # Create original account and backup mnemonic
        original_account = StatusBackend(self.await_signals)
        original_account.init_status_backend()
        original_account.create_account_and_login()
        original_account.wait_for_login()
        original_get_settings_response = original_account.settings_service.get_settings()
        original_settings = original_get_settings_response.get("result", {})
        mnemonic = original_settings.get("mnemonic", None)
        assert mnemonic is not None

        user = copy.deepcopy(user_mnemonic_12)
        user.passphrase = mnemonic

        # Restore account in a new container
        restored_account = StatusBackend(self.await_signals)
        restored_account.init_status_backend()
        restored_account.restore_account_and_login(user=user)
        restored_account.wait_for_login()

        # Verify mnemonic is not exposed after restore
        restored_get_settings_response = restored_account.settings_service.get_settings()
        restored_settings = restored_get_settings_response.get("result", {})
        assert restored_settings.get("mnemonic", None) is None

        # Verify keys in the restored respone match the original respones
        for key in [
            "address",
            "compressedKey",
            "current-user-status",
            "dapps-address",
            "eip1581-address",
            "emojiHash",
            "name",
            "public-key",
            "wallet-root-address",
        ]:
            assert original_settings.get(key) == restored_settings.get(key), f"Restored {key} doesn't match with original"

        # But some are different as expected
        assert original_settings.get("installation-id") != restored_settings.get("installation-id"), "installation-id shouldn't match"

    @pytest.mark.parametrize(
        "user_mnemonic",
        [user_mnemonic_24, user_mnemonic_15, user_mnemonic_12],
        ids=["mnemonic_24", "mnemonic_15", "mnemonic_12"],
    )
    def test_restore_app_different_valid_size_mnemonics(self, user_mnemonic):
        # Initialize backend client and restore account using user_mnemonic.
        restored_account = StatusBackend(self.await_signals)
        restored_account.init_status_backend()
        restored_account.restore_account_and_login(user=user_mnemonic)
        restored_account.wait_for_login()

        # Request getAccounts and check restoreed accounts attributes.
        get_accounts_response = restored_account.accounts_service.get_accounts()
        assert_response_attributes(get_accounts_response.get("result", {}), user_mnemonic.accounts)

        # Request getSettings and check restoreed profile data attributes.
        get_settings_response = restored_account.settings_service.get_settings()
        restored_settings = get_settings_response.get("result", {})
        assert_response_attributes(restored_settings, user_mnemonic.profile_data)

        assert restored_settings.get("mnemonic", None) is None
        assert restored_settings.get("address", None)

    @pytest.mark.parametrize(
        "mnemonic_size",
        [1, 3, 7, 13, 29],
    )
    def test_restore_with_arbitrary_size_mnemonics(self, mnemonic_size):
        # Restore with an arbitrary length mnemonic
        user = user_mnemonic_12
        user.passphrase = " ".join("".join(random.choice(string.ascii_lowercase) for _ in range(random.randint(2, 10))) for _ in range(mnemonic_size))

        restored_account = StatusBackend(self.await_signals)
        restored_account.init_status_backend()
        restored_account.restore_account_and_login(user=user)
        restored_account.wait_for_login()

        get_settings_response = restored_account.settings_service.get_settings()
        restored_settings = get_settings_response.get("result", {})
        assert restored_settings.get("mnemonic", None) is None
        assert restored_settings.get("address", None)

    def test_restore_with_mnemonic_with_special_chars(self):
        # Restore with an mnemonic with special chars
        user = user_mnemonic_12
        user.passphrase = "<>?`~!@#$%^&*()_+1 $fgdg ^&*()"

        restored_account = StatusBackend(self.await_signals)
        restored_account.init_status_backend()
        restored_account.restore_account_and_login(user=user)
        restored_account.wait_for_login()
        get_settings_response = restored_account.settings_service.get_settings()
        restored_settings = get_settings_response.get("result", {})
        assert restored_settings.get("mnemonic", None) is None
        assert restored_settings.get("address", None)

    def test_restore_with_empty_mnemonic(self):
        # Restore with empty mnemonic isn't allowed
        user = user_mnemonic_12

        restored_account = StatusBackend(self.await_signals)
        restored_account.init_status_backend()
        restored_account._set_display_name()
        data = restored_account._create_account_request(user)
        data["mnemonic"] = ""
        restored_account_response = restored_account.api_request("RestoreAccountAndLogin", data)
        assert restored_account_response.json().get("error") == "restore-account: mnemonic is not set"

    def test_restore_with_both_mnemonic_and_keycard(self):
        # Restore with both keycard and mnemonic isn't allowed
        user = user_mnemonic_12
        restored_account = StatusBackend(self.await_signals)
        restored_account.init_status_backend()
        restored_account._set_display_name()
        data = restored_account._create_account_request(user)
        data["mnemonic"] = user.passphrase
        data["keycard"] = user_keycard_1
        restored_account_response = restored_account.api_request("RestoreAccountAndLogin", data)
        assert restored_account_response.json().get("error") == "restore-account: mnemonic is set for keycard account"

    def test_restored_on_existing_restored_account_fails(self):
        user = user_mnemonic_12
        restored_account = StatusBackend(self.await_signals)
        restored_account.init_status_backend()
        restored_account.restore_account_and_login(user=user)
        restored_account.wait_for_login()
        restored_account.restore_account_and_login(user=user)
        signal = restored_account.wait_for_signal(SignalType.NODE_LOGIN.value)

        assert user.profile_data is not None, "User profile_data should not be None"
        key_uid = user.profile_data.get("key-uid", "")
        expected_error = f"[validation] keypair already added -  keyuid: {key_uid}"

        assert expected_error in signal.get("event").get("error")
