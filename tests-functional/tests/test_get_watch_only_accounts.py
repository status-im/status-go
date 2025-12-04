import copy
import pytest
from resources.constants import new_account_data_1


@pytest.mark.rpc
class TestGetWatchOnlyAccounts:

    @pytest.fixture()
    def account(self, backend_new_profile):
        return backend_new_profile("account-backend")

    @pytest.fixture()
    def account_data(self):
        return copy.deepcopy(new_account_data_1)

    def test_get_watch_only_accounts_returns_all_watch_accounts(self, account, account_data):
        account_data["type"] = "watch"
        account.accounts_service.add_account(account.password, account_data)

        watch_accounts = account.accounts_service.get_watch_only_accounts()
        assert len(watch_accounts) == 1
        assert watch_accounts[0].get("address") == account_data.get("address")

    def test_get_watch_only_with_no_watch_accounts(self, account):
        watch_accounts = account.accounts_service.get_watch_only_accounts()
        assert watch_accounts is None
