import re
import pytest
from resources.constants import user_1
from clients.api import ApiResponseError


@pytest.mark.rpc
class TestUpdateKeypairName:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("sender")

    def test_update_name_for_profile_keypair_isnt_allowed(self):
        new_name = "Updated Keypair Name"
        with pytest.raises(ApiResponseError, match=re.escape("cannot change profile keypair name")):
            self.account.accounts_service.update_keypair_name(self.account.key_uid, new_name)

    def test_update_keypair_name_for_seed_account(self):
        wallet_account_details = {
            "name": "SeedImportedAccount",
            "path": "m/44'/60'/0'/0/0",
            "emoji": "🔑",
            "colorId": "primary",
        }
        self.account.accounts_service.add_keypair_via_seed_phrase(
            user_1.passphrase, self.account.password, "SeedImportedKeypair", wallet_account_details
        )

        keypairs_response = self.account.accounts_service.get_account_keypairs()
        keypairs = keypairs_response.get("result", [])
        assert len(keypairs) > 0
        seed_key_uid = keypairs[1].get("key-uid")
        new_name = "Updated Keypair Name"

        response = self.account.accounts_service.update_keypair_name(seed_key_uid, new_name)
        assert "result" in response
        assert response.get("result") is None

        # Verify the keypair name was updated by fetching keypairs again
        keypairs_response_after = self.account.accounts_service.get_account_keypairs()
        keypairs_after = keypairs_response_after.get("result", [])
        updated_keypair = next((kp for kp in keypairs_after if kp.get("key-uid") == seed_key_uid), None)
        assert updated_keypair is not None
        assert updated_keypair.get("name") == new_name
