import re
import pytest
from resources.constants import user_1, wallet_account_details_derivation, keypair_name
from clients.api import ApiResponseError


@pytest.mark.rpc
class TestUpdateKeypairName:

    @pytest.fixture()
    def backend(self, backend_new_profile):
        return backend_new_profile("account-backend")

    def test_update_name_for_profile_keypair_isnt_allowed(self, backend):
        new_name = "Updated Keypair Name"
        with pytest.raises(ApiResponseError, match=re.escape("cannot change profile keypair name")):
            backend.accounts_service.update_keypair_name(backend.key_uid, new_name)

    def test_update_keypair_name_for_seed_account(self, backend):
        backend.accounts_service.add_keypair_via_seed_phrase(user_1.passphrase, backend.password, keypair_name, wallet_account_details_derivation)

        keypairs_response = backend.accounts_service.get_account_keypairs()
        keypairs = keypairs_response
        assert len(keypairs) > 0
        seed_key_uid = keypairs[1].get("key-uid")
        new_name = "Updated Keypair Name"

        response = backend.accounts_service.update_keypair_name(seed_key_uid, new_name)
        assert response is None

        # Verify the keypair name was updated by fetching keypairs again
        keypairs_response_after = backend.accounts_service.get_account_keypairs()
        keypairs_after = keypairs_response_after
        updated_keypair = next((kp for kp in keypairs_after if kp.get("key-uid") == seed_key_uid), None)
        assert updated_keypair is not None
        assert updated_keypair.get("name") == new_name
