import re
import pytest
from clients.api import ApiResponseError
from resources.constants import user_1, wallet_account_details_root, keypair_name


@pytest.mark.rpc
class TestDeleteKeypair:

    @pytest.fixture()
    def backend(self, backend_new_profile):
        return backend_new_profile("account-backend")

    def test_delete_keypair(self, backend):
        add_resp = backend.accounts_service.add_keypair_via_private_key(
            user_1.private_key, backend.password, keypair_name, wallet_account_details_root
        )
        key_uid = add_resp.get("key-uid")
        address = add_resp.get("accounts")[0].get("address")

        resp = backend.accounts_service.verify_keystore_file_for_account(address, backend.password)
        assert resp is True

        del_resp = backend.accounts_service.delete_keypair(key_uid, backend.password)
        assert del_resp is None

        with pytest.raises(ApiResponseError, match=re.escape("keypair is not found")):
            backend.accounts_service.get_keypair_by_key_uid(key_uid)

        resp = backend.accounts_service.verify_keystore_file_for_account(address, backend.password)
        assert resp is False

    def test_delete_non_existent_keypair(self, backend):
        with pytest.raises(ApiResponseError, match=re.escape("keypair is not found")):
            backend.accounts_service.delete_keypair("key_uid", backend.password)
