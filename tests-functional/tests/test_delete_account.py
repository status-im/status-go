import copy
import re
import pytest
from resources.constants import new_account_data_1
from clients.api import ApiResponseError


@pytest.mark.rpc
class TestDeleteAccount:

    @pytest.fixture()
    def account(self, backend_new_profile):
        """Backend for account deletion tests (profile owner)."""
        return backend_new_profile("account-backend")

    @pytest.fixture()
    def account_data(self):
        """Fresh copy of base account data for each test."""
        return copy.deepcopy(new_account_data_1)

    def test_delete_watch_account(self, account, account_data):
        account_data["type"] = "watch"
        account.accounts_service.add_account("", account_data)

        accounts_response = account.accounts_service.get_accounts()
        assert len(accounts_response) == 3

        resp = account.accounts_service.delete_account(account_data["address"], "")
        assert resp is None

        accounts_response = account.accounts_service.get_accounts()
        assert len(accounts_response) == 2

    def test_delete_default_chat_account(self, account):
        accounts_response = account.accounts_service.get_accounts()
        assert len(accounts_response) == 2

        with pytest.raises(ApiResponseError, match=re.escape("[database] cannot remove default chat account")):
            account.accounts_service.delete_account(accounts_response[0].get("address"), account.password)

        accounts_response = account.accounts_service.get_accounts()
        assert len(accounts_response) == 2

    def test_delete_default_wallet_account(self, account):
        accounts_response = account.accounts_service.get_accounts()
        assert len(accounts_response) == 2

        with pytest.raises(ApiResponseError, match=re.escape("[database] cannot remove default wallet account")):
            account.accounts_service.delete_account(accounts_response[1].get("address"), account.password)

        accounts_response = account.accounts_service.get_accounts()
        assert len(accounts_response) == 2

    def test_delete_nonexistent_account(self, account):
        with pytest.raises(ApiResponseError, match=re.escape("accounts: account is not found")):
            account.accounts_service.delete_account("0x0000000000000000000000000000000000000001", account.password)
