import pytest
from resources.constants import user_1, wallet_account_details_derivation, keypair_name


@pytest.mark.rpc
class TestGetNumOfAddressesToGenerateForKeypair:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("sender")

    def test_get_num_of_addresses_to_generate_for_keypair(self):
        add_resp = self.account.accounts_service.add_keypair_via_seed_phrase(
            user_1.passphrase, self.account.password, keypair_name, wallet_account_details_derivation
        )
        result = self.account.accounts_service.get_num_of_addresses_to_generate_for_keypair(add_resp.get("key-uid"))
        assert result == 100
