# Ad-hoc test — no backend spec. Restores coverage deleted in 33be7c3e7 in cold-wallet vocabulary.
import copy
import re
import pytest
from clients.api import ApiResponseError
from resources.constants import user_1, wallet_account_details_derivation, keypair_name


@pytest.mark.rpc
class TestDeleteColdWalletKeypair:

    @pytest.fixture()
    def backend(self, backend_new_profile):
        return backend_new_profile("account-backend")

    def _add_cold_keypair(self, backend):
        add_resp = backend.accounts_service.add_keypair_via_seed_phrase(
            user_1.passphrase,
            backend.password,
            keypair_name,
            "",
            wallet_account_details_derivation,
        )
        key_uid = add_resp.get("key-uid")
        assert key_uid is not None, "Expected addKeypairViaSeedPhrase to return the created keypair's key-uid"
        backend.accounts_service.migrate_non_profile_keypair_to_cold_wallet(key_uid, backend.password, "status-keycard")
        return key_uid

    def test_delete_cold_wallet_keypair_without_password(self, backend):
        key_uid = self._add_cold_keypair(backend)

        backend.accounts_service.delete_keypair(key_uid, "")

        with pytest.raises(ApiResponseError, match=re.escape("keypair is not found")):
            backend.accounts_service.get_keypair_by_key_uid(key_uid)

    def test_delete_nonexistent_keypair_errors(self, backend):
        unknown_key_uid = "0x3231d92c94548d14f097173765a50bebe28fbad8f2267c9e08cc4433a6f219a4"
        with pytest.raises(ApiResponseError, match=re.escape("keypair is not found")):
            backend.accounts_service.delete_keypair(unknown_key_uid, backend.password)

    def test_delete_account_of_cold_wallet_keypair_without_password(self, backend):
        add_resp = backend.accounts_service.add_keypair_via_seed_phrase(
            user_1.passphrase,
            backend.password,
            keypair_name,
            "",
            wallet_account_details_derivation,
        )
        key_uid = add_resp.get("key-uid")
        assert key_uid is not None, "Expected addKeypairViaSeedPhrase to return the created keypair's key-uid"

        second_path = "m/44'/60'/0'/0/1"
        derived = backend.wallet_service.get_derived_addresses_for_mnemonic(user_1.passphrase, [second_path])
        template = copy.deepcopy(add_resp.get("accounts")[0])
        template.update(
            {
                "address": derived[0].get("address"),
                "public-key": derived[0].get("public-key"),
                "path": second_path,
                "name": "second-account",
                "wallet": False,
                "chat": False,
            }
        )
        backend.accounts_service.add_account(backend.password, template)
        backend.accounts_service.migrate_non_profile_keypair_to_cold_wallet(key_uid, backend.password, "status-keycard")
        target_address = derived[0].get("address")

        backend.accounts_service.delete_account(target_address, "")

        kp = backend.accounts_service.get_keypair_by_key_uid(key_uid)
        remaining = [a for a in kp.get("accounts", []) if a.get("address", "").lower() == target_address.lower()]
        assert len(remaining) == 0, "Expected the account removed without a password because cold-wallet keypairs have no keystore files to unlock"
