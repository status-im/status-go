import pytest
from resources.constants import user_1, wallet_account_details_derivation, keypair_name


@pytest.mark.rpc
class TestGetNumOfAddressesToGenerateForKeypair:

    @pytest.fixture()
    def account(self, backend_new_profile):
        return backend_new_profile("account-backend")

    def test_get_num_of_addresses_to_generate_for_keypair(self, account):
        add_resp = account.accounts_service.add_keypair_via_seed_phrase(
            user_1.passphrase, account.password, keypair_name, wallet_account_details_derivation
        )
        result = account.accounts_service.get_num_of_addresses_to_generate_for_keypair(add_resp.get("key-uid"))
        assert result == 100
