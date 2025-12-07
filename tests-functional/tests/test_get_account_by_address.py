import copy
import re
import pytest
from resources.constants import new_account_data_1
from clients.api import ApiResponseError


@pytest.mark.rpc
class TestGetAccountByAddress:

    @pytest.fixture()
    def backend(self, backend_new_profile):
        return backend_new_profile("account-backend")

    @pytest.fixture()
    def account_data(self):
        return copy.deepcopy(new_account_data_1)

    def test_get_account_for_existing_addresses(self, backend, account_data):
        path = "m/44'/60'/0'/0/1"
        derived_addresses_response = backend.wallet_service.get_derived_addresses_for_mnemonic(backend.mnemonic, [path])
        derived_addresses = derived_addresses_response
        account_data["key-uid"] = backend.key_uid  # keyuid of profile keypair
        account_data["path"] = path
        account_data["address"] = derived_addresses[0].get("address")
        account_data["public-key"] = derived_addresses[0].get("public-key")
        backend.accounts_service.add_account(backend.password, account_data)

        accounts_response = backend.accounts_service.get_accounts()
        for acc in accounts_response:
            resp = backend.accounts_service.get_account_by_address(acc.get("address"))
            get_account_by_address_resp = resp
            assert get_account_by_address_resp == acc

    @pytest.mark.parametrize(
        ("address", "error"),
        [
            ("0x0000000000000000000000000000000000000001", "accounts: account is not found"),
            ("0x00", "invalid argument 0: hex string has length 2, want 40 for types.Address"),
            ("foo", "invalid argument 0: json: cannot unmarshal hex string without 0x prefix into Go value of type types.Address"),
        ],
    )
    def test_get_account_by_invalid_address(self, backend, address, error):
        with pytest.raises(ApiResponseError, match=re.escape(error)):
            backend.accounts_service.get_account_by_address(address)
