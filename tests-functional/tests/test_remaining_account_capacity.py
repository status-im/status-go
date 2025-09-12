import copy
import re
import pytest
from clients.api import ApiResponseError
from resources.constants import new_account_data_1, new_account_data_2
import secrets


@pytest.mark.rpc
class TestRemainingAccountCapacity:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("sender")
        self.account_data = copy.deepcopy(new_account_data_1)

    def test_remaining_account_capacity_decreases_on_add(self):
        # 1) Query remaining capacity
        initial_capacity = self.account.accounts_service.remaining_account_capacity()
        assert initial_capacity == 19

        # 2) Add a second account
        path = "m/44'/60'/0'/0/1"
        derived_addresses_response = self.account.wallet_service.get_derived_addresses_for_mnemonic(self.account.mnemonic, [path])
        self.account_data["key-uid"] = self.account.key_uid  # keyuid of profile keypair
        self.account_data["path"] = path
        self.account_data["address"] = derived_addresses_response[0].get("address")
        self.account_data["public-key"] = derived_addresses_response[0].get("public-key")
        self.account.accounts_service.add_account(self.account.password, self.account_data)

        # 3) Query remaining capacity again and assert it decreased by exactly 1
        after_capacity = self.account.accounts_service.remaining_account_capacity()
        assert after_capacity == initial_capacity - 1

        # 4) Add a watch-only account (simpler to add without deriving a wallet address)
        self.account_data2 = copy.deepcopy(new_account_data_2)
        self.account_data2["type"] = "watch"
        self.account.accounts_service.add_account(self.account.password, self.account_data2)

        # 5) Query remaining capacity again and assert it decreased by exactly 1
        after_capacity2 = self.account.accounts_service.remaining_account_capacity()
        assert after_capacity2 == after_capacity - 1

    @pytest.mark.skip(reason="Skipped due to https://github.com/status-im/status-go/issues/6922")
    def test_no_more_accounts_can_be_added(self):
        initial_capacity = self.account.accounts_service.remaining_account_capacity()
        for _ in range(initial_capacity):
            self.account_data["type"] = "watch"
            self.account.accounts_service.add_account(self.account.password, self.account_data)
            self.account_data["address"] = "0x" + secrets.token_hex(20)

        accounts = self.account.accounts_service.get_accounts()
        assert len(accounts) == 20

        with pytest.raises(ApiResponseError, match=re.escape("no more accounts can be added")):
            self.account.accounts_service.remaining_account_capacity()

        self.account.accounts_service.add_account(self.account.password, self.account_data)

        accounts = self.account.accounts_service.get_accounts()
        assert len(accounts) == 20
