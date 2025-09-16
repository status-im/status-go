import re
import pytest
from clients.api import ApiResponseError
from resources.constants import user_1, wallet_account_details_derivation, keypair_name


@pytest.mark.rpc
class TestAddKeypairStoredToKeycard:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("sender")

    def test_add_keypair_stored_to_keycard_flow(self):
        add_keypair_resp = self.account.accounts_service.add_keypair_via_seed_phrase(
            user_1.passphrase, self.account.password, keypair_name, wallet_account_details_derivation
        )
        addresses = [add_keypair_resp.get("accounts")[0].get("address"), add_keypair_resp.get("derived-from")]

        key_uid = add_keypair_resp.get("key-uid")
        self.account.accounts_service.get_keypair_by_key_uid(key_uid)

        for address in addresses:
            resp = self.account.accounts_service.verify_keystore_file_for_account(address, self.account.password)
            assert resp is True

        # delete the keypair so we can add it back
        self.account.accounts_service.delete_keypair(key_uid, self.account.password)
        with pytest.raises(ApiResponseError, match=re.escape("keypair is not found")):
            self.account.accounts_service.get_keypair_by_key_uid(key_uid)

        for address in addresses:
            resp = self.account.accounts_service.verify_keystore_file_for_account(address, self.account.password)
            assert resp is False

        new_name = "keypair-stored"
        added_stored_keypair = self.account.accounts_service.add_keypair_stored_to_keycard(
            key_uid, user_1.address, new_name, [wallet_account_details_derivation]
        )
        assert added_stored_keypair.get("key-uid") == key_uid
        assert added_stored_keypair.get("name") == new_name

        kp_after = self.account.accounts_service.get_keypair_by_key_uid(key_uid)
        assert kp_after.get("key-uid") == key_uid
        assert kp_after.get("name") == new_name

        for address in addresses:
            resp = self.account.accounts_service.verify_keystore_file_for_account(address, self.account.password)
            assert resp is False
