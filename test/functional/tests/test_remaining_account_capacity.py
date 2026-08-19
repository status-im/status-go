import copy
import re
import pytest
from clients.api import ApiResponseError
from resources.constants import new_account_data_1, new_account_data_2
import secrets


@pytest.mark.rpc
class TestRemainingAccountCapacity:

    @pytest.fixture()
    def backend(self, backend_new_profile):
        return backend_new_profile("account-backend")

    @pytest.fixture()
    def account_data(self):
        return copy.deepcopy(new_account_data_1)

    def test_remaining_account_capacity_decreases_on_add(self, backend, account_data):
        # 1) Query remaining capacity
        initial_capacity = backend.accounts_service.remaining_account_capacity()
        assert initial_capacity == 19

        # 2) Add a second account
        path = "m/44'/60'/0'/0/1"
        derived_addresses_response = backend.wallet_service.get_derived_addresses_for_mnemonic(backend.mnemonic, [path])
        account_data["key-uid"] = backend.key_uid  # keyuid of profile keypair
        account_data["path"] = path
        account_data["address"] = derived_addresses_response[0].get("address")
        account_data["public-key"] = derived_addresses_response[0].get("public-key")
        backend.accounts_service.add_account(backend.password, account_data)

        # 3) Query remaining capacity again and assert it decreased by exactly 1
        after_capacity = backend.accounts_service.remaining_account_capacity()
        assert after_capacity == initial_capacity - 1

        # 4) Add a watch-only account (simpler to add without deriving a wallet address)
        account_data2 = copy.deepcopy(new_account_data_2)
        account_data2["type"] = "watch"
        backend.accounts_service.add_account(backend.password, account_data2)

        # 5) Query remaining capacity again and assert it decreased by exactly 1
        after_capacity2 = backend.accounts_service.remaining_account_capacity()
        assert after_capacity2 == after_capacity - 1

    def test_no_more_accounts_can_be_added(self, backend, account_data):
        initial_capacity = backend.accounts_service.remaining_account_capacity()
        for _ in range(initial_capacity):
            account_data["type"] = "watch"
            account_data["address"] = "0x" + secrets.token_hex(20)
            backend.accounts_service.add_account(backend.password, account_data)

        accounts = backend.accounts_service.get_accounts()
        assert len(accounts) == 21

        with pytest.raises(ApiResponseError, match=re.escape("no more accounts can be added")):
            backend.accounts_service.remaining_account_capacity()
