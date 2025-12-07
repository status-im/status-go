import re
import pytest
from clients.api import ApiResponseError
from resources.constants import user_1, wallet_account_details_root, keypair_name


@pytest.mark.rpc
class TestUpdateKeypair:

    @pytest.fixture()
    def backend(self, backend_new_profile):
        return backend_new_profile("account-backend")

    def test_update_keypair(self, backend):
        add_resp = backend.accounts_service.add_keypair_via_private_key(
            user_1.private_key, backend.password, keypair_name, wallet_account_details_root
        )
        key_uid = add_resp.get("key-uid")

        kp_modified = backend.accounts_service.get_keypair_by_key_uid(key_uid)
        updated_name = "KeypairUpdatedName"
        updated_emoji = "🕵️"
        kp_modified["name"] = updated_name
        kp_modified["accounts"][0]["emoji"] = "🕵️"

        update_resp = backend.accounts_service.update_keypair(kp_modified)
        assert update_resp is None

        kp_after = backend.accounts_service.get_keypair_by_key_uid(key_uid)
        assert kp_after.get("name") == updated_name
        assert kp_after.get("accounts")[0].get("emoji") == updated_emoji

    def test_update_non_existent_keypair(self, backend):
        add_resp = backend.accounts_service.add_keypair_via_private_key(
            user_1.private_key, backend.password, keypair_name, wallet_account_details_root
        )
        key_uid = add_resp.get("key-uid")

        kp_modified = backend.accounts_service.get_keypair_by_key_uid(key_uid)
        kp_modified["key-uid"] = "213214"

        with pytest.raises(ApiResponseError, match=re.escape("cannot update non-existing keypair: keypair is not found")):
            backend.accounts_service.update_keypair(kp_modified)
