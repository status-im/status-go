"""Profile keypair migration to a keycard through ConvertToKeycardAccountV2, driven with a mock pairing.

Observable keycard state on this branch is keypair.cold-wallet, keystore presence and multiaccounts.keycard-pairing.
The keycards tables and their RPCs were dropped, so per-card rows are not assertable here.
"""

import copy

import pytest
from clients.api import ApiResponseError
from clients.signals import SignalType
from resources.constants import user_1
from steps.keycard import (
    assert_keystore_state,
    derive_chat_address,
    derive_chat_private_key,
    derive_keycard_password,
)

KEYCARD_UID = "mock-keycard-uid"
KEYPAIR_FIELDS = ("key-uid", "type", "name", "derived-from", "xpub")


def _pairing(key_uid, suffix=""):
    return f"mock-pairing-{key_uid[:8]}{suffix}"


def _addresses(keypair):
    return [a["address"] for a in keypair["accounts"]] + [keypair["derived-from"]]


def _account_shape(keypair):
    return sorted((a["address"], a["path"], a["wallet"], a["chat"]) for a in keypair["accounts"])


def _keycard_pairing_of(accounts, key_uid):
    entry = next((a for a in accounts if a.get("key-uid") == key_uid), None)
    assert entry is not None, f"Expected InitializeApplication to list the account {key_uid}"
    return entry.get("keycard-pairing", "")


@pytest.mark.rpc
class TestConvertProfileKeypairToKeycard:

    @pytest.fixture()
    def backend(self, backend_new_profile):
        return backend_new_profile("keycard-convert")

    def _profile_precondition(self, backend):
        kp = backend.accounts_service.get_keypair_by_key_uid(backend.key_uid)
        assert kp is not None, "Expected the profile keypair to exist"
        assert kp["type"] == "profile"
        assert kp.get("cold-wallet", "") == "", "Expected a fresh profile keypair to not be on a cold wallet"
        assert kp.get("xpub", "").startswith("xpub"), "Expected a fresh profile keypair to carry its wallet xpub"
        assert kp["derived-from"], "Expected the profile keypair to carry its master address"
        assert derive_chat_address(backend.mnemonic) in [a["address"] for a in kp["accounts"] if a.get("chat")]
        assert_keystore_state(backend, _addresses(kp), backend.password, present=True)
        assert backend.settings_service.get_settings().get("mnemonic"), "Expected a new profile to still hold its mnemonic"
        return kp

    def _assert_on_keycard(self, backend, kp0, old_password, keycard_password):
        kp1 = backend.accounts_service.get_keypair_by_key_uid(backend.key_uid)
        assert kp1["cold-wallet"] == "status-keycard", "Expected the profile keypair to be flagged as keycard backed"
        for field in KEYPAIR_FIELDS:
            assert kp1.get(field) == kp0.get(field), f"Expected keypair field {field} to survive the keycard migration"
        assert _account_shape(kp1) == _account_shape(kp0), "Expected the profile accounts to be unchanged by the keycard migration"
        for account in kp1["accounts"]:
            assert account["operable"] == "fully", f"Expected {account['address']} fully operable because the keycard now holds the key"
        # Files must be deleted, not re-encrypted: neither password may resolve them.
        assert_keystore_state(backend, _addresses(kp1), old_password, present=False)
        assert_keystore_state(backend, _addresses(kp1), keycard_password, present=False)
        return kp1

    def _convert(self, backend, pairing):
        keycard_password = derive_keycard_password(backend, backend.mnemonic)
        old_password = backend.password
        backend.convert_to_keycard_account_v2(backend.key_uid, pairing, KEYCARD_UID, old_password, keycard_password)
        return old_password, keycard_password

    def test_convert_profile_keypair_to_keycard(self, backend):
        key_uid, mnemonic = backend.key_uid, backend.mnemonic
        kp0 = self._profile_precondition(backend)
        pairing = _pairing(key_uid)

        old_password, keycard_password = self._convert(backend, pairing)

        kp1 = self._assert_on_keycard(backend, kp0, old_password, keycard_password)
        profiles = [kp for kp in backend.accounts_service.get_account_keypairs() if kp.get("type") == "profile"]
        assert len(profiles) == 1 and profiles[0]["key-uid"] == key_uid, "Expected exactly one profile keypair after the migration"
        assert profiles[0]["cold-wallet"] == "status-keycard"
        listed = {a["address"]: a for a in backend.accounts_service.get_accounts()}
        for account in kp1["accounts"]:
            assert listed[account["address"]]["wallet"] == account["wallet"]
            assert listed[account["address"]]["chat"] == account["chat"]

        assert (
            backend.accounts_service.verify_password(old_password) is False
        ), "Expected verifyPassword to fail because the chat keystore file is gone"
        assert backend.accounts_service.verify_password(keycard_password) is False

        settings = backend.settings_service.get_settings()
        assert not settings.get("mnemonic"), "Expected the mnemonic to be cleared because the keycard now owns the seed"
        assert settings.get("profile-migration-needed", False) is False
        assert settings["key-uid"] == key_uid

        accounts = backend.reinit_and_get_accounts()
        assert _keycard_pairing_of(accounts, key_uid) == pairing, "Expected the keycard pairing stored on the multiaccount"

        backend.login_with_keycard(key_uid, keycard_password, derive_chat_private_key(mnemonic))
        signal = backend.wait_for_login()
        assert signal["event"]["account"]["key-uid"] == key_uid
        backend.wait_for_wakuext_ready(timeout=30)

        kp2 = self._assert_on_keycard(backend, kp0, old_password, keycard_password)

        next_path = "m/44'/60'/0'/0/1"
        derived = backend.wallet_service.get_derived_addresses_for_mnemonic(mnemonic, [next_path])
        template = copy.deepcopy(kp2["accounts"][0])
        template.update(
            {
                "address": derived[0]["address"],
                "public-key": derived[0]["public-key"],
                "path": next_path,
                "name": "kc-derived",
                "wallet": False,
                "chat": False,
            }
        )
        backend.accounts_service.add_account("", template)
        kp3 = backend.accounts_service.get_keypair_by_key_uid(key_uid)
        added = [a for a in kp3["accounts"] if a["path"] == next_path]
        assert len(added) == 1, "Expected the empty-password add_account to succeed because the keycard migration retains the stored xpub"

    def test_convert_already_on_keycard_is_idempotent(self, backend):
        key_uid = backend.key_uid
        kp0 = self._profile_precondition(backend)
        old_password, keycard_password = self._convert(backend, _pairing(key_uid))
        pairing2 = _pairing(key_uid, "-repaired")

        backend.convert_to_keycard_account_v2(key_uid, pairing2, KEYCARD_UID, keycard_password, keycard_password)

        self._assert_on_keycard(backend, kp0, old_password, keycard_password)
        assert _keycard_pairing_of(backend.reinit_and_get_accounts(), key_uid) == pairing2, "Expected a re-convert to store the new pairing"

    def test_convert_rejects_wrong_old_password(self, backend):
        key_uid = backend.key_uid
        kp0 = self._profile_precondition(backend)
        old_password = backend.password
        keycard_password = derive_keycard_password(backend, backend.mnemonic)

        with pytest.raises(ApiResponseError, match="invalid key-encryption key|incorrect password provided"):
            backend.convert_to_keycard_account_v2(key_uid, "bad-pairing", KEYCARD_UID, "definitely-wrong", keycard_password)

        kp1 = backend.accounts_service.get_keypair_by_key_uid(key_uid)
        assert kp1.get("cold-wallet", "") == "", "Expected the keypair to stay off the keycard after a rejected conversion"
        assert kp1["xpub"] == kp0["xpub"]
        assert_keystore_state(backend, _addresses(kp1), old_password, present=True)
        assert backend.accounts_service.verify_password(old_password) is True
        assert backend.settings_service.get_settings().get("mnemonic"), "Expected the mnemonic kept because the password check failed first"
        # The pairing is written before the password check (status-im/status-go#7698); this pins the current behaviour.
        assert _keycard_pairing_of(backend.reinit_and_get_accounts(), key_uid) == "bad-pairing"

        backend.login(key_uid, old_password)
        signal = backend.wait_for_login()
        assert signal["event"]["account"]["key-uid"] == key_uid, "Expected password login to keep working because the keystore files still exist"

    def test_convert_unknown_key_uid_is_rejected(self, backend):
        key_uid = backend.key_uid
        self._profile_precondition(backend)
        keycard_password = derive_keycard_password(backend, backend.mnemonic)

        with pytest.raises(ApiResponseError):
            backend.convert_to_keycard_account_v2("0x" + "ab" * 32, _pairing(key_uid), KEYCARD_UID, backend.password, keycard_password)

        kp1 = backend.accounts_service.get_keypair_by_key_uid(key_uid)
        assert kp1.get("cold-wallet", "") == ""
        assert_keystore_state(backend, _addresses(kp1), backend.password, present=True)
        assert (
            _keycard_pairing_of(backend.reinit_and_get_accounts(), key_uid) == ""
        ), "Expected the real account untouched by a request for an unknown key-uid"

    def test_lost_keycard_login_with_mnemonic(self, backend):
        # Incidental coverage for the lost-keycard recovery flow (status-app#21627).
        key_uid, mnemonic = backend.key_uid, backend.mnemonic
        kp0 = self._profile_precondition(backend)
        pairing = _pairing(key_uid)
        old_password, keycard_password = self._convert(backend, pairing)
        backend.reinit_and_get_accounts()

        # LoginAccount reports failures through the node.login signal, not the HTTP response.
        with backend.expect_signal(SignalType.NODE_LOGIN, timeout=60) as exp:
            backend.login_with_mnemonic(key_uid, user_1.passphrase)
        assert "mnemonic does not match this account" in exp.result["event"].get("error", "")

        backend.login_with_mnemonic(key_uid, mnemonic)
        signal = backend.wait_for_login()
        assert signal["event"]["account"]["key-uid"] == key_uid
        backend.wait_for_wakuext_ready(timeout=30)
        self._assert_on_keycard(backend, kp0, old_password, keycard_password)
