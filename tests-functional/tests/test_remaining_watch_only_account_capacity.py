import copy
import re
import pytest
from clients.api import ApiResponseError
from resources.constants import new_account_data_1
import secrets


@pytest.mark.rpc
class TestRemainingWatchOnlyAccountCapacity:

    @pytest.fixture()
    def account(self, backend_new_profile):
        return backend_new_profile("account-backend")

    @pytest.fixture()
    def account_data(self):
        data = copy.deepcopy(new_account_data_1)
        data["type"] = "watch"
        return data

    def test_remaining_watch_only_account_capacity_decreases_on_add(self, account, account_data):
        initial_capacity = account.accounts_service.remaining_watch_only_account_capacity()
        assert initial_capacity == 3

        account.accounts_service.add_account(account.password, account_data)

        after_capacity = account.accounts_service.remaining_watch_only_account_capacity()
        assert after_capacity == initial_capacity - 1

    def test_no_more_watch_only_accounts_can_be_added(self, account, account_data):
        initial_capacity = account.accounts_service.remaining_watch_only_account_capacity()
        for _ in range(initial_capacity):
            account_data["address"] = "0x" + secrets.token_hex(20)
            account.accounts_service.add_account(account.password, account_data)

        accounts = account.accounts_service.get_accounts()
        watch_accounts = [account for account in accounts if account["type"] == "watch"]
        assert len(watch_accounts) == 3

        with pytest.raises(ApiResponseError, match=re.escape("no more watch-only accounts can be added")):
            account.accounts_service.remaining_watch_only_account_capacity()
