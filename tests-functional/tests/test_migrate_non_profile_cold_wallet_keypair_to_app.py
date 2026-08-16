import copy
import re

import pytest
from clients.api import ApiResponseError
from resources.constants import (
    user_1,
    wallet_account_details_derivation,
    keypair_name,
)


@pytest.mark.rpc
class TestMigrateNonProfileColdWalletKeypairToApp:

    @pytest.fixture()
    def backend(self, backend_new_profile):
        return backend_new_profile("account-backend")

    def _add_seed_keypair(self, backend):
        add_resp = backend.accounts_service.add_keypair_via_seed_phrase(
            user_1.passphrase,
            backend.password,
            keypair_name,
            "",
            wallet_account_details_derivation,
        )
        assert add_resp is not None, "Expected addKeypairViaSeedPhrase to return the created keypair"
        key_uid = add_resp.get("key-uid")
        assert key_uid is not None, "Expected the created keypair to carry a key-uid"
        accounts_list = add_resp.get("accounts", [])
        assert len(accounts_list) > 0, "Expected the created keypair to have at least one account"
        addresses = [a.get("address") for a in accounts_list] + [add_resp.get("derived-from")]
        return key_uid, addresses

    def _profile_addresses(self, backend):
        kp = backend.accounts_service.get_keypair_by_key_uid(backend.key_uid)
        assert kp is not None, "Expected the profile keypair to exist"
        return [a.get("address") for a in kp.get("accounts", [])]

    @pytest.mark.parametrize("cold_wallet_type", ["status-keycard", "ledger", "trezor"])
    def test_full_migrate_flow(self, backend, cold_wallet_type):
        # 1) Add a seed-derived keypair (regular, not cold-wallet)
        key_uid, addresses = self._add_seed_keypair(backend)

        # Keystore files for the keypair accounts should exist
        for address in addresses:
            resp = backend.accounts_service.verify_keystore_file_for_account(address, backend.password)
            assert resp is True, f"Expected a keystore file for {address} because the keypair was just imported with a password"

        # 2) Migrate the keypair to a cold wallet
        # This should remove the keystore files
        backend.accounts_service.migrate_non_profile_keypair_to_cold_wallet(key_uid, backend.password, cold_wallet_type)

        for address in addresses:
            resp = backend.accounts_service.verify_keystore_file_for_account(address, backend.password)
            assert resp is False, f"Expected no keystore file for {address} because cold-wallet migration deletes keystore files"

        # Verify the keypair is now flagged as cold-wallet backed, with the exact type round-tripped
        kp_resp = backend.accounts_service.get_keypair_by_key_uid(key_uid)
        assert kp_resp is not None
        assert (
            kp_resp.get("cold-wallet") == cold_wallet_type
        ), f"Expected cold-wallet '{cold_wallet_type}' to round-trip through the RPC layer unchanged"

        # 3) Migrate the non-profile cold-wallet keypair back to the app (full flow)
        # Keystore files should be restored and the cold-wallet flag cleared
        migrate_resp = backend.accounts_service.migrate_non_profile_cold_wallet_keypair_to_app(user_1.passphrase, backend.password)
        # many RPC "void" actions return result None
        assert migrate_resp is None

        for address in addresses:
            resp = backend.accounts_service.verify_keystore_file_for_account(address, backend.password)
            assert resp is True, f"Expected the keystore file for {address} to be restored by migrate-to-app"

        # 4) Verify the keypair still exists, cold-wallet flag is cleared, and accounts remain
        kp_resp = backend.accounts_service.get_keypair_by_key_uid(key_uid)
        assert kp_resp is not None
        assert kp_resp.get("cold-wallet", "") == "", "Expected the cold-wallet flag cleared after migrate-to-app"
        assert isinstance(kp_resp.get("accounts"), list)
        assert len(kp_resp.get("accounts")) >= 1

        # 5) The stored xpub must be RETAINED through migrate-to-app: adding a derived
        # account with an EMPTY password still works (xpub-validated, no keystore derive)
        next_path = "m/44'/60'/0'/0/1"
        derived = backend.wallet_service.get_derived_addresses_for_mnemonic(user_1.passphrase, [next_path])
        template = copy.deepcopy(kp_resp["accounts"][0])
        template.update(
            {
                "address": derived[0].get("address"),
                "public-key": derived[0].get("public-key"),
                "path": next_path,
                "name": "xpub-derived",
                "wallet": False,
                "chat": False,
            }
        )
        backend.accounts_service.add_account("", template)
        kp_resp = backend.accounts_service.get_keypair_by_key_uid(key_uid)
        added = [a for a in kp_resp.get("accounts", []) if a.get("path") == next_path]
        assert len(added) == 1, "Expected the empty-password add_account to succeed because migrate-to-app retains the stored xpub"

    def test_remigrate_same_cold_wallet_type_is_idempotent(self, backend):
        key_uid, addresses = self._add_seed_keypair(backend)
        backend.accounts_service.migrate_non_profile_keypair_to_cold_wallet(key_uid, backend.password, "status-keycard")

        backend.accounts_service.migrate_non_profile_keypair_to_cold_wallet(key_uid, backend.password, "status-keycard")

        kp_resp = backend.accounts_service.get_keypair_by_key_uid(key_uid)
        assert kp_resp.get("cold-wallet") == "status-keycard", "Expected re-migrating to the same cold-wallet type to be a no-op keeping the type"
        for address in addresses:
            resp = backend.accounts_service.verify_keystore_file_for_account(address, backend.password)
            assert resp is False, f"Expected keystore files for {address} to stay absent after an idempotent re-migrate"

    def test_migrate_profile_keypair_to_cold_wallet_is_rejected(self, backend):
        profile_addresses = self._profile_addresses(backend)

        with pytest.raises(ApiResponseError, match=re.escape("use ConvertToKeycardAccount instead")):
            backend.accounts_service.migrate_non_profile_keypair_to_cold_wallet(backend.key_uid, backend.password, "status-keycard")

        for address in profile_addresses:
            resp = backend.accounts_service.verify_keystore_file_for_account(address, backend.password)
            assert resp is True, f"Expected the profile keystore file for {address} untouched because the migrate RPC must reject profile keypairs"

    def test_migrate_profile_keypair_to_app_is_rejected(self, backend):
        with pytest.raises(ApiResponseError, match=re.escape("use ConvertToRegularAccount instead")):
            backend.accounts_service.migrate_non_profile_cold_wallet_keypair_to_app(backend.mnemonic, backend.password)

    def test_migrate_unknown_key_uid_to_cold_wallet_errors(self, backend):
        unknown_key_uid = "0x3231d92c94548d14f097173765a50bebe28fbad8f2267c9e08cc4433a6f219a4"
        with pytest.raises(ApiResponseError, match=re.escape("keypair is not found")):
            backend.accounts_service.migrate_non_profile_keypair_to_cold_wallet(unknown_key_uid, backend.password, "status-keycard")

    def test_migrate_to_app_rejects_non_cold_keypair(self, backend):
        self._add_seed_keypair(backend)
        with pytest.raises(ApiResponseError, match=re.escape("keypair is not a cold wallet keypair")):
            backend.accounts_service.migrate_non_profile_cold_wallet_keypair_to_app(user_1.passphrase, backend.password)

    def test_migrate_to_app_rejects_unknown_mnemonic(self, backend):
        random_mnemonic = backend.accounts_service.get_random_mnemonic()
        with pytest.raises(ApiResponseError, match=re.escape("keypair is not found")):
            backend.accounts_service.migrate_non_profile_cold_wallet_keypair_to_app(random_mnemonic, backend.password)

    def test_migrate_to_app_rejects_wrong_password(self, backend):
        key_uid, addresses = self._add_seed_keypair(backend)
        backend.accounts_service.migrate_non_profile_keypair_to_cold_wallet(key_uid, backend.password, "status-keycard")

        with pytest.raises(ApiResponseError, match=re.escape("wrong password provided")):
            backend.accounts_service.migrate_non_profile_cold_wallet_keypair_to_app(user_1.passphrase, "definitely-wrong-password")

        for address in addresses:
            resp = backend.accounts_service.verify_keystore_file_for_account(address, backend.password)
            assert (
                resp is False
            ), f"Expected no keystore file for {address} because migrate-to-app with a wrong password must not write keystore files"
