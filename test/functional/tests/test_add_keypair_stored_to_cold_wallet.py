import copy
import re
import pytest
from clients.api import ApiResponseError
from resources.constants import user_1, wallet_account_details_derivation, keypair_name


@pytest.mark.rpc
class TestAddKeypairStoredToColdWallet:

    @pytest.fixture()
    def backend(self, backend_new_profile):
        return backend_new_profile("account-backend")

    def test_add_keypair_stored_to_cold_wallet_flow(self, backend):
        add_keypair_resp = backend.accounts_service.add_keypair_via_seed_phrase(
            user_1.passphrase,
            backend.password,
            keypair_name,
            "",
            wallet_account_details_derivation,
        )
        addresses = [
            add_keypair_resp.get("accounts")[0].get("address"),
            add_keypair_resp.get("derived-from"),
        ]

        key_uid = add_keypair_resp.get("key-uid")
        backend.accounts_service.get_keypair_by_key_uid(key_uid)

        for address in addresses:
            resp = backend.accounts_service.verify_keystore_file_for_account(address, backend.password)
            assert resp is True

        # delete the keypair so we can add it back
        backend.accounts_service.delete_keypair(key_uid, backend.password)
        with pytest.raises(ApiResponseError, match=re.escape("keypair is not found")):
            backend.accounts_service.get_keypair_by_key_uid(key_uid)

        for address in addresses:
            resp = backend.accounts_service.verify_keystore_file_for_account(address, backend.password)
            assert resp is False

        new_name = "keypair-stored"
        added_stored_keypair = backend.accounts_service.add_keypair_stored_to_cold_wallet(
            key_uid,
            user_1.address,
            new_name,
            user_1.wallet_xpub,
            "",
            [wallet_account_details_derivation],
        )
        assert added_stored_keypair.get("key-uid") == key_uid
        assert added_stored_keypair.get("name") == new_name

        kp_after = backend.accounts_service.get_keypair_by_key_uid(key_uid)
        assert kp_after.get("key-uid") == key_uid
        assert kp_after.get("name") == new_name

        for address in addresses:
            resp = backend.accounts_service.verify_keystore_file_for_account(address, backend.password)
            assert resp is False

    def _add_then_delete_seed_keypair(self, backend):
        add_resp = backend.accounts_service.add_keypair_via_seed_phrase(
            user_1.passphrase,
            backend.password,
            keypair_name,
            "",
            wallet_account_details_derivation,
        )
        key_uid = add_resp.get("key-uid")
        assert key_uid is not None, "Expected addKeypairViaSeedPhrase to return the created keypair's key-uid"
        backend.accounts_service.delete_keypair(key_uid, backend.password)
        return key_uid

    def test_add_account_with_empty_password_on_cold_keypair_derives_from_xpub(self, backend):
        key_uid = self._add_then_delete_seed_keypair(backend)

        backend.accounts_service.add_keypair_stored_to_cold_wallet(
            key_uid,
            user_1.address,
            "cold-keypair",
            user_1.wallet_xpub,
            "status-keycard",
            [wallet_account_details_derivation],
        )
        kp = backend.accounts_service.get_keypair_by_key_uid(key_uid)
        assert (
            kp.get("cold-wallet") == "status-keycard"
        ), "Expected the keypair to be cold-wallet backed because it was added via addKeypairStoredToColdWallet"

        next_path = "m/44'/60'/0'/0/1"
        derived = backend.wallet_service.get_derived_addresses_for_mnemonic(user_1.passphrase, [next_path])
        template = copy.deepcopy(kp["accounts"][0])
        template.update(
            {
                "address": derived[0].get("address"),
                "public-key": derived[0].get("public-key"),
                "path": next_path,
                "name": "xpub-derived-cold",
                "wallet": False,
                "chat": False,
                "key-uid": key_uid,
            }
        )
        backend.accounts_service.add_account("", template)

        kp_after_add = backend.accounts_service.get_keypair_by_key_uid(key_uid)
        added = [a for a in kp_after_add.get("accounts", []) if a.get("path") == next_path]
        assert (
            len(added) == 1
        ), "Expected the empty-password add_account to succeed because the cold keypair carries a wallet xpub to validate against"
        assert (
            added[0].get("address", "").lower() == derived[0].get("address", "").lower()
        ), "Expected the stored account address to match the xpub-derived address for the requested path"
        resp = backend.accounts_service.verify_keystore_file_for_account(added[0].get("address"), backend.password)
        assert resp is False, "Expected no keystore file for the new account because cold-wallet keypair accounts must keep key material off disk"

    def test_add_keypair_stored_to_cold_wallet_rejects_empty_wallet_accounts(self, backend):
        key_uid = self._add_then_delete_seed_keypair(backend)

        with pytest.raises(ApiResponseError, match=re.escape("keypair must have at least one wallet account")):
            backend.accounts_service.add_keypair_stored_to_cold_wallet(
                key_uid,
                user_1.address,
                "cold-keypair",
                user_1.wallet_xpub,
                "status-keycard",
                [],
            )
        with pytest.raises(ApiResponseError, match=re.escape("keypair is not found")):
            backend.accounts_service.get_keypair_by_key_uid(key_uid)

    def test_add_keypair_stored_to_cold_wallet_rejects_non_wallet_path(self, backend):
        key_uid = self._add_then_delete_seed_keypair(backend)

        bad_account = copy.deepcopy(wallet_account_details_derivation)
        bad_account["path"] = "m/43'/60'/0'/0/0"
        with pytest.raises(ApiResponseError, match=re.escape("unsupported profile or seed imported key pair wallet account")):
            backend.accounts_service.add_keypair_stored_to_cold_wallet(
                key_uid,
                user_1.address,
                "cold-keypair",
                user_1.wallet_xpub,
                "status-keycard",
                [bad_account],
            )
        with pytest.raises(ApiResponseError, match=re.escape("keypair is not found")):
            backend.accounts_service.get_keypair_by_key_uid(key_uid)

    def test_add_keypair_stored_to_cold_wallet_rejects_already_added_keypair(self, backend):
        add_resp = backend.accounts_service.add_keypair_via_seed_phrase(
            user_1.passphrase,
            backend.password,
            keypair_name,
            "",
            wallet_account_details_derivation,
        )
        key_uid = add_resp.get("key-uid")

        with pytest.raises(ApiResponseError, match=re.escape("keypair already added")):
            backend.accounts_service.add_keypair_stored_to_cold_wallet(
                key_uid,
                user_1.address,
                "cold-keypair",
                user_1.wallet_xpub,
                "status-keycard",
                [wallet_account_details_derivation],
            )
        kp = backend.accounts_service.get_keypair_by_key_uid(key_uid)
        assert (
            kp.get("name") == keypair_name
        ), "Expected the original keypair untouched because addKeypairStoredToColdWallet must reject an already-added keypair"
