import copy
import re
import pytest
from clients.api import ApiResponseError
from resources.constants import new_account_data_1
import secrets


@pytest.mark.rpc
class TestRemainingWatchOnlyAccountCapacity:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("sender")
        self.account_data = copy.deepcopy(new_account_data_1)
        self.account_data["type"] = "watch"

    def test_remaining_watch_only_account_capacity_decreases_on_add(self):
        initial_capacity = self.account.accounts_service.remaining_watch_only_account_capacity()
        assert initial_capacity == 3

        self.account.accounts_service.add_account(self.account.password, self.account_data)

        after_capacity = self.account.accounts_service.remaining_watch_only_account_capacity()
        assert after_capacity == initial_capacity - 1

    def test_no_more_watch_only_accounts_can_be_added(self):
        initial_capacity = self.account.accounts_service.remaining_watch_only_account_capacity()
        for _ in range(initial_capacity):
            self.account_data["address"] = "0x" + secrets.token_hex(20)
            self.account.accounts_service.add_account(self.account.password, self.account_data)

        accounts = self.account.accounts_service.get_accounts()
        watch_accounts = [account for account in accounts if account["type"] == "watch"]
        assert len(watch_accounts) == 3

        with pytest.raises(ApiResponseError, match=re.escape("no more watch-only accounts can be added")):
            self.account.accounts_service.remaining_watch_only_account_capacity()
