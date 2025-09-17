import copy
import pytest
from resources.constants import new_account_data_1


@pytest.mark.rpc
class TestGetWatchOnlyAccounts:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("sender")
        self.account_data = copy.deepcopy(new_account_data_1)

    def test_get_watch_only_accounts_returns_all_watch_accounts(self):
        self.account_data["type"] = "watch"
        self.account.accounts_service.add_account(self.account.password, self.account_data)

        watch_accounts = self.account.accounts_service.get_watch_only_accounts()
        assert len(watch_accounts) == 1
        assert watch_accounts[0].get("address") == self.account_data.get("address")

    def test_get_watch_only_with_no_watch_accounts(self):
        watch_accounts = self.account.accounts_service.get_watch_only_accounts()
        assert watch_accounts is None
