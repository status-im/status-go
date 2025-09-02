import copy
import pytest
from resources.constants import new_account_data_1


@pytest.mark.rpc
class TestGetAccountByAddress:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("sender")
        self.account_data = copy.deepcopy(new_account_data_1)

    def test_get_account_for_existing_addresses(self):
        path = "m/44'/60'/0'/0/1"
        derived_addresses_response = self.account.wallet_service.get_derived_addresses_for_mnemonic(self.account.mnemonic, [path])
        derived_addresses = derived_addresses_response["result"]
        self.account_data["key-uid"] = self.account.key_uid  # keyuid of profile keypair
        self.account_data["path"] = path
        self.account_data["address"] = derived_addresses[0].get("address")
        self.account_data["public-key"] = derived_addresses[0].get("public-key")
        self.account.accounts_service.add_account(self.account.password, self.account_data)

        accounts_response = self.account.accounts_service.get_accounts()
        for account in accounts_response.get("result"):
            resp = self.account.accounts_service.get_account_by_address(account.get("address"))
            get_account_by_address_resp = resp.get("result")
            assert get_account_by_address_resp == account

    @pytest.mark.parametrize(
        ("address", "error"),
        [
            ("0x0000000000000000000000000000000000000001", "accounts: account is not found"),
            ("0x00", "invalid argument 0: hex string has length 2, want 40 for types.Address"),
            ("foo", "invalid argument 0: json: cannot unmarshal hex string without 0x prefix into Go value of type types.Address"),
        ],
    )
    def test_get_account_by_invalid_address(self, address, error):
        resp = self.account.accounts_service.get_account_by_address(address)
        assert resp.get("error").get("message") == error
