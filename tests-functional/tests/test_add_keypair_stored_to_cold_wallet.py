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
