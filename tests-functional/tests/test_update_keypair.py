import re
import pytest
from clients.api import ApiResponseError
from resources.constants import user_1, wallet_account_details_root, keypair_name


@pytest.mark.rpc
class TestUpdateKeypair:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("sender")

    def test_update_keypair(self):
        add_resp = self.account.accounts_service.add_keypair_via_private_key(
            user_1.private_key, self.account.password, keypair_name, wallet_account_details_root
        )
        key_uid = add_resp.get("key-uid")

        kp_modified = self.account.accounts_service.get_keypair_by_key_uid(key_uid)
        updated_name = "KeypairUpdatedName"
        updated_emoji = "🕵️"
        kp_modified["name"] = updated_name
        kp_modified["accounts"][0]["emoji"] = "🕵️"

        update_resp = self.account.accounts_service.update_keypair(kp_modified)
        assert update_resp is None

        kp_after = self.account.accounts_service.get_keypair_by_key_uid(key_uid)
        assert kp_after.get("name") == updated_name
        assert kp_after.get("accounts")[0].get("emoji") == updated_emoji

    def test_update_non_existent_keypair(self):
        add_resp = self.account.accounts_service.add_keypair_via_private_key(
            user_1.private_key, self.account.password, keypair_name, wallet_account_details_root
        )
        key_uid = add_resp.get("key-uid")

        kp_modified = self.account.accounts_service.get_keypair_by_key_uid(key_uid)
        kp_modified["key-uid"] = "213214"

        with pytest.raises(ApiResponseError, match=re.escape("cannot update non-existing keypair: keypair is not found")):
            self.account.accounts_service.update_keypair(kp_modified)
