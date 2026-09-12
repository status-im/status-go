"""Restoring an existing keycard account through RestoreAccountAndLogin, the path the app uses for keycard login.
The card's contents are read off a real profile restored from the same seed rather than made up."""

import copy

import pytest
from clients.api import ApiResponseError
from clients.signals import SignalType
from resources.constants import user_1, user_mnemonic_12
from steps.keycard import PATH_EIP1581_CHAT, PATH_EIP1581_ENCRYPTION, derive_chat_private_key, keycard_pairing_of

PATH_WALLET_ROOT = "m/44'/60'/0'/0"
PATH_EIP1581_ROOT = "m/43'/60'/1581'"
PATH_DEFAULT_WALLET = "m/44'/60'/0'/0/0"

KEYCARD_INSTANCE_UID = "mock-keycard-instance-uid"
KEYCARD_PAIRING_KEY = "mock-keycard-pairing-key"


@pytest.mark.rpc
class TestRestoreKeycardAccount:

    @pytest.fixture()
    def card(self, backend_factory):
        origin = backend_factory("card-origin")
        origin.init_status_backend()
        with origin.expect_signal(SignalType.NODE_LOGIN, timeout=60) as exp:
            origin.restore_account_and_login(user=user_1)
        assert not exp.result["event"].get("error"), exp.result["event"].get("error")
        key_uid = exp.result["event"]["account"]["key-uid"]
        origin.wallet_service.start_wallet()

        keypair = origin.accounts_service.get_keypair_by_key_uid(key_uid)
        # Cross-checked against the recorded constant so a mis-derived payload fails here, not downstream.
        assert keypair["xpub"] == user_1.wallet_xpub, "Expected the source profile's xpub to match the recorded constant"
        paths = [PATH_WALLET_ROOT, PATH_EIP1581_ROOT, PATH_EIP1581_CHAT, PATH_DEFAULT_WALLET, PATH_EIP1581_ENCRYPTION]
        derived = dict(zip(paths, origin.wallet_service.get_derived_addresses_for_mnemonic(user_1.passphrase, paths)))

        data = {
            "keyUID": key_uid,
            "address": keypair["derived-from"],
            "whisperPrivateKey": derive_chat_private_key(user_1.passphrase),
            "whisperPublicKey": derived[PATH_EIP1581_CHAT]["public-key"],
            "whisperAddress": derived[PATH_EIP1581_CHAT]["address"],
            "walletPublicKey": derived[PATH_DEFAULT_WALLET]["public-key"],
            "walletAddress": derived[PATH_DEFAULT_WALLET]["address"],
            "walletRootAddress": derived[PATH_WALLET_ROOT]["address"],
            "eip1581Address": derived[PATH_EIP1581_ROOT]["address"],
            "encryptionPublicKey": derived[PATH_EIP1581_ENCRYPTION]["public-key"],
            "walletXPub": keypair["xpub"],
            "coldWallet": "status-keycard",
        }
        origin.logout()
        return data

    @pytest.fixture()
    def restoring(self, backend_factory):
        backend = backend_factory("keycard-restore")
        backend.init_status_backend()
        return backend

    def _restore(self, backend, card, pairing_key=KEYCARD_PAIRING_KEY, instance_uid=KEYCARD_INSTANCE_UID):
        with backend.expect_signal(SignalType.NODE_LOGIN, timeout=60) as exp:
            backend.restore_keycard_account_and_login(card, instance_uid, pairing_key)
        return exp.result["event"]

    def test_restore_an_account_that_already_lives_on_a_keycard(self, restoring, card):
        event = self._restore(restoring, card)
        assert not event.get("error"), event.get("error")
        assert event["account"]["key-uid"] == card["keyUID"], "Expected the restored profile to keep the card's key-uid"

        keypair = restoring.accounts_service.get_keypair_by_key_uid(card["keyUID"])
        assert keypair["type"] == "profile"
        assert keypair["cold-wallet"] == "status-keycard", "Expected the restored profile keypair to be flagged as card backed"
        assert keypair["xpub"] == card["walletXPub"], "Expected the card's xpub to be stored, since the seed is not available to derive one"
        assert keypair["derived-from"].lower() == card["address"].lower()

        addresses = {a["address"].lower() for a in keypair["accounts"]}
        assert card["whisperAddress"].lower() in addresses, "Expected the chat account to come from the card's whisper key"
        assert card["walletAddress"].lower() in addresses, "Expected the default wallet account to come from the card"

        for address in [a["address"] for a in keypair["accounts"]] + [keypair["derived-from"]]:
            assert (
                restoring.accounts_service.verify_keystore_file_for_account(address, restoring.password) is False
            ), f"Expected no keystore file for {address} because the key stays on the card"

        settings = restoring.settings_service.get_settings()
        assert not settings.get("mnemonic"), "Expected no mnemonic stored for a keycard restore"
        assert settings["key-uid"] == card["keyUID"]

        assert (
            keycard_pairing_of(restoring.reinit_and_get_accounts(), card["keyUID"]) == KEYCARD_PAIRING_KEY
        ), "Expected the pairing key to reach the multiaccount"

    def test_the_restored_keycard_profile_logs_in_again(self, restoring, card):
        self._restore(restoring, card)
        keycard_password = restoring.password
        restoring.logout()

        with restoring.expect_signal(SignalType.NODE_LOGIN, timeout=60) as exp:
            restoring.login_with_keycard(card["keyUID"], keycard_password, card["whisperPrivateKey"])
        assert not exp.result["event"].get("error"), exp.result["event"].get("error")
        assert exp.result["event"]["account"]["key-uid"] == card["keyUID"]

        keypair = restoring.accounts_service.get_keypair_by_key_uid(card["keyUID"])
        assert keypair["cold-wallet"] == "status-keycard", "Expected the card flag to survive a re-login"
        assert keypair["xpub"] == card["walletXPub"]

    def test_login_accepts_a_chat_key_that_does_not_match_the_profile(self, restoring, card):
        # Characterises current behaviour, not a contract: the mnemonic login path rejects a seed that
        # does not derive the account, the keycard path takes the supplied chat key verbatim.
        event = self._restore(restoring, card)
        public_key = event["settings"]["public-key"]
        keycard_password = restoring.password
        restoring.logout()

        foreign_chat_key = derive_chat_private_key(user_mnemonic_12.passphrase)
        with restoring.expect_signal(SignalType.NODE_LOGIN, timeout=60) as exp:
            restoring.login_with_keycard(card["keyUID"], keycard_password, foreign_chat_key)

        assert not exp.result["event"].get("error"), "Characterises current behaviour: a foreign chat key is accepted without error"
        assert exp.result["event"]["account"]["key-uid"] == card["keyUID"]
        assert (
            exp.result["event"]["settings"]["public-key"] == public_key
        ), "The stored identity does not follow the key that was actually selected, so the two have diverged silently"

    def test_restore_via_keycard_rejects_an_empty_whisper_private_key(self, restoring, card):
        without_chat_key = copy.deepcopy(card)
        without_chat_key["whisperPrivateKey"] = ""

        with pytest.raises(ApiResponseError, match="restore-account: chat private key is not set"):
            restoring.restore_keycard_account_and_login(without_chat_key, KEYCARD_INSTANCE_UID, KEYCARD_PAIRING_KEY)

        assert (restoring.init_status_backend().get("accounts") or []) == [], "Expected a rejected restore to create no account"

    def test_restore_via_keycard_without_a_pairing_key_is_rejected(self, restoring, card):
        # An empty pairing key sends the backend to a pairings file this device never wrote; the
        # failure arrives on node.login rather than in the HTTP response.
        event = self._restore(restoring, card, pairing_key="")
        assert "failed to prepare for keycard" in event.get("error", ""), event.get("error")
        assert "keycard pairings" in event.get("error", ""), event.get("error")

        assert (restoring.init_status_backend().get("accounts") or []) == [], "Expected no account to survive a restore that failed the pairings gate"
