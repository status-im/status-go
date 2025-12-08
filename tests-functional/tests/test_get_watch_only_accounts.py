import copy
import pytest
from resources.constants import new_account_data_1


@pytest.mark.rpc
class TestGetWatchOnlyAccounts:

    @pytest.fixture()
    def backend(self, backend_new_profile):
        return backend_new_profile("account-backend")

    @pytest.fixture()
    def account_data(self):
        return copy.deepcopy(new_account_data_1)

    def test_get_watch_only_accounts_returns_all_watch_accounts(self, backend, account_data):
        account_data["type"] = "watch"
        backend.accounts_service.add_account(backend.password, account_data)

        watch_accounts = backend.accounts_service.get_watch_only_accounts()
        assert len(watch_accounts) == 1
        assert watch_accounts[0].get("address") == account_data.get("address")

    def test_get_watch_only_with_no_watch_accounts(self, backend):
        watch_accounts = backend.accounts_service.get_watch_only_accounts()
        assert watch_accounts is None
