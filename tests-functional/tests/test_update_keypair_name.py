import pytest


@pytest.mark.rpc
class TestUpdateKeypairName:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("sender")

    def test_update_name_for_profile_keypair_isnt_allowed(self):
        new_name = "Updated Keypair Name"
        response = self.account.accounts_service.update_keypair_name(self.account.key_uid, new_name, skip_validation=True)
        assert "error" not in response

        # Optionally, verify the keypair name was updated by fetching keypairs again
        keypairs_response_after = self.account.accounts_service.get_account_keypairs()
        keypairs_after = keypairs_response_after.get("result", [])
        updated_keypair = next((kp for kp in keypairs_after if kp["key-uid"] == new_name), None)
        assert updated_keypair is not None
        assert updated_keypair.get("name") == new_name

    def test_update_keypair_name(self):

        # need to do add keypaiar via addKeypairViaSeedPhrase and then change the name
        # waiting for https://github.com/status-im/status-go/pull/6777 to be merged for that
        # Get keypairs to find a key-uid
        keypairs_response = self.account.accounts_service.get_account_keypairs()
        keypairs = keypairs_response.get("result", [])
        assert len(keypairs) > 0
        key_uid = keypairs[0]["key-uid"]

        new_name = "Updated Keypair Name"

        # Call update_keypair_name RPC method
        response = self.account.accounts_service.update_keypair_name(key_uid, new_name)
        assert "error" not in response

        # Optionally, verify the keypair name was updated by fetching keypairs again
        keypairs_response_after = self.account.accounts_service.get_account_keypairs()
        keypairs_after = keypairs_response_after.get("result", [])
        updated_keypair = next((kp for kp in keypairs_after if kp["key-uid"] == key_uid), None)
        assert updated_keypair is not None
        assert updated_keypair.get("name") == new_name
