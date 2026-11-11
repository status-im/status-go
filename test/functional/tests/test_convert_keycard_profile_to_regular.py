import re
from collections import namedtuple

import pytest
from clients.api import ApiResponseError
from clients.signals import SignalType
from resources.constants import COLD_WALLET_TYPE_STATUS_KEYCARD, user_2
from steps.keycard import (
    KEYCARD_UID,
    KEYPAIR_FIELDS,
    account_shape,
    assert_keystore_state,
    derive_chat_private_key,
    derive_keycard_password,
    fresh_profile_keypair,
    keycard_pairing_of,
    keypair_addresses,
    mock_pairing,
)
from utils import fake


KeycardProfile = namedtuple("KeycardProfile", ["key_uid", "mnemonic", "old_password", "keycard_password", "pairing", "kp0"])


@pytest.mark.rpc
class TestConvertKeycardProfileToRegular:

    @pytest.fixture()
    def backend(self, backend_new_profile):
        return backend_new_profile("keycard-to-regular")

    def _put_profile_on_keycard(self, backend):
        kp0 = fresh_profile_keypair(backend)

        keycard_password = derive_keycard_password(backend, backend.mnemonic)
        old_password = backend.password
        pairing = mock_pairing(backend.key_uid)
        backend.convert_to_keycard_account_v2(backend.key_uid, pairing, KEYCARD_UID, old_password, keycard_password)

        kp = backend.accounts_service.get_keypair_by_key_uid(backend.key_uid)
        assert kp["cold-wallet"] == COLD_WALLET_TYPE_STATUS_KEYCARD, "Expected the setup to flag the profile keypair as keycard backed"
        assert kp["xpub"] == kp0["xpub"], "Expected the setup to keep the wallet xpub"
        assert_keystore_state(backend, keypair_addresses(kp), keycard_password, present=False)

        return KeycardProfile(backend.key_uid, backend.mnemonic, old_password, keycard_password, pairing, kp0)

    def _assert_regular(self, backend, kp0, password):
        kp = backend.accounts_service.get_keypair_by_key_uid(kp0["key-uid"])
        assert kp.get("cold-wallet", "") == "", "Expected the profile keypair to be back on the app keystore"
        for field in KEYPAIR_FIELDS:
            assert kp.get(field) == kp0.get(field), f"Expected keypair field {field} to survive the reverse migration"
        assert account_shape(kp) == account_shape(kp0), "Expected the profile accounts to be unchanged by the reverse migration"
        for account in kp["accounts"]:
            assert account["operable"] == "fully", f"Expected {account['address']} fully operable after the reverse migration"
        assert_keystore_state(backend, keypair_addresses(kp), password, present=True)
        assert backend.accounts_service.verify_password(password) is True, "Expected verifyPassword to succeed with the new password"
        return kp

    def test_stop_using_keycard_for_profile_keypair(self, backend):
        ctx = self._put_profile_on_keycard(backend)
        new_password = "NewRegular-" + fake.profile_password()

        backend.convert_to_regular_account_v2(ctx.mnemonic, ctx.keycard_password, new_password)

        self._assert_regular(backend, ctx.kp0, new_password)

        probe_address = ctx.kp0["accounts"][0]["address"]
        keystore_with_keycard_password = backend.accounts_service.verify_keystore_file_for_account(probe_address, ctx.keycard_password)
        keystore_with_old_password = backend.accounts_service.verify_keystore_file_for_account(probe_address, ctx.old_password)
        assert keystore_with_keycard_password is False, "Observed: the keycard password no longer resolves the re-created keystore file"
        assert keystore_with_old_password is False, "Observed: the pre-keycard password does not resolve the re-created keystore file"

        settings = backend.settings_service.get_settings()
        assert settings.get("profile-migration-needed", False) is False
        assert not settings.get("mnemonic"), "Expected the mnemonic to stay cleared because the conversion does not restore it"

        accounts = backend.reinit_and_get_accounts()
        assert keycard_pairing_of(accounts, ctx.key_uid) == "", "Expected the keycard pairing to be cleared on the multiaccount"

        with backend.expect_signal(SignalType.NODE_LOGIN, timeout=60) as exp:
            backend.login(ctx.key_uid, new_password)
        assert not exp.result["event"].get("error"), exp.result["event"].get("error")
        assert exp.result["event"]["account"]["key-uid"] == ctx.key_uid
        assert exp.result["event"]["account"].get("keycard-pairing", "") == "", "Expected the login event to carry no keycard pairing"
        backend.wait_for_wakuext_ready(timeout=30)

        self._assert_regular(backend, ctx.kp0, new_password)

        backend.change_database_password(new_password, new_password + "x")
        backend.reinit_and_get_accounts()
        with backend.expect_signal(SignalType.NODE_LOGIN, timeout=60) as exp:
            backend.login(ctx.key_uid, new_password + "x")
        assert not exp.result["event"].get("error"), exp.result["event"].get("error")
        assert exp.result["event"]["account"]["key-uid"] == ctx.key_uid, "Expected a normal password profile after the password change"
        assert backend.accounts_service.get_keypair_by_key_uid(ctx.key_uid) is not None

    def test_stop_using_keycard_rejects_wrong_password(self, backend):
        ctx = self._put_profile_on_keycard(backend)

        with pytest.raises(ApiResponseError, match="invalid key-encryption key|incorrect password provided"):
            backend.convert_to_regular_account_v2(ctx.mnemonic, "definitely-wrong", "whatever-new")

        kp = backend.accounts_service.get_keypair_by_key_uid(ctx.key_uid)
        assert kp["cold-wallet"] == COLD_WALLET_TYPE_STATUS_KEYCARD, "Expected the keypair to stay on the keycard after a rejected conversion"
        assert kp["xpub"] == ctx.kp0["xpub"]
        assert_keystore_state(backend, keypair_addresses(kp), ctx.keycard_password, present=False)
        assert_keystore_state(backend, keypair_addresses(kp), ctx.old_password, present=False)
        assert (
            keycard_pairing_of(backend.reinit_and_get_accounts(), ctx.key_uid) == ctx.pairing
        ), "Expected the pairing kept because the password check runs before it is cleared"

        with backend.expect_signal(SignalType.NODE_LOGIN, timeout=60) as exp:
            backend.login_with_keycard(ctx.key_uid, ctx.keycard_password, derive_chat_private_key(ctx.mnemonic))
        assert not exp.result["event"].get("error"), exp.result["event"].get("error")
        assert exp.result["event"]["account"]["key-uid"] == ctx.key_uid, "Expected keycard login to keep working after a rejected conversion"
        assert backend.accounts_service.get_keypair_by_key_uid(ctx.key_uid) is not None

    def test_stop_using_keycard_rejects_unknown_mnemonic(self, backend):
        ctx = self._put_profile_on_keycard(backend)

        with pytest.raises(ApiResponseError, match="no rows"):
            backend.convert_to_regular_account_v2(user_2.passphrase, ctx.keycard_password, "whatever-new")

        kp = backend.accounts_service.get_keypair_by_key_uid(ctx.key_uid)
        assert kp["cold-wallet"] == COLD_WALLET_TYPE_STATUS_KEYCARD, "Expected the keypair untouched by an unknown mnemonic"
        assert_keystore_state(backend, keypair_addresses(kp), ctx.keycard_password, present=False)
        assert (
            keycard_pairing_of(backend.reinit_and_get_accounts(), ctx.key_uid) == ctx.pairing
        ), "Expected the pairing kept because the mnemonic lookup fails first"

        with backend.expect_signal(SignalType.NODE_LOGIN, timeout=60) as exp:
            backend.login_with_keycard(ctx.key_uid, ctx.keycard_password, derive_chat_private_key(ctx.mnemonic))
        assert not exp.result["event"].get("error"), exp.result["event"].get("error")
        assert exp.result["event"]["account"]["key-uid"] == ctx.key_uid
        assert backend.accounts_service.get_keypair_by_key_uid(ctx.key_uid) is not None

    def test_stop_using_keycard_accepts_whitespace_padded_mnemonic(self, backend):
        ctx = self._put_profile_on_keycard(backend)
        padded = "  " + ctx.mnemonic.replace(" ", "   ") + " \n"
        new_password = "NewRegular-" + fake.profile_password()

        backend.convert_to_regular_account_v2(padded, ctx.keycard_password, new_password)

        self._assert_regular(backend, ctx.kp0, new_password)
        assert (
            keycard_pairing_of(backend.reinit_and_get_accounts(), ctx.key_uid) == ""
        ), "Expected the keycard pairing cleared after a padded-mnemonic conversion"

    def test_stop_using_secure_channel_v2_clears_the_marker(self, backend):
        key_uid, mnemonic = backend.key_uid, backend.mnemonic
        kp0 = fresh_profile_keypair(backend)
        keycard_password = derive_keycard_password(backend, mnemonic)
        old_password = backend.password
        backend.convert_to_keycard_account_v2(key_uid, "", KEYCARD_UID, old_password, keycard_password, keycard_secure_channel_v2=True)
        kp_on_card = backend.accounts_service.get_keypair_by_key_uid(key_uid)
        assert kp_on_card["cold-wallet"] == COLD_WALLET_TYPE_STATUS_KEYCARD, "Expected the profile on Keycard before stop using"
        new_password = "NewRegular-" + fake.profile_password()

        backend.convert_to_regular_account_v2(mnemonic, keycard_password, new_password)

        self._assert_regular(backend, kp0, new_password)
        assert (
            keycard_pairing_of(backend.reinit_and_get_accounts(), key_uid) == ""
        ), "Expected the secure-channel-v2 marker cleared because the profile no longer uses a keycard"

    def test_stop_using_keycard_on_regular_profile_is_rejected(self, backend):
        kp0 = backend.accounts_service.get_keypair_by_key_uid(backend.key_uid)
        assert kp0.get("cold-wallet", "") == "", "Expected a fresh profile to not be on a cold wallet"

        with pytest.raises(ApiResponseError, match=re.escape("keypair is not a cold wallet keypair")):
            backend.convert_to_regular_account_v2(backend.mnemonic, backend.password, "new-pw")

        kp = backend.accounts_service.get_keypair_by_key_uid(backend.key_uid)
        assert kp.get("cold-wallet", "") == ""
        assert_keystore_state(backend, keypair_addresses(kp), backend.password, present=True)
        assert keycard_pairing_of(backend.reinit_and_get_accounts(), backend.key_uid) == "", "Expected no keycard pairing on a regular profile"
