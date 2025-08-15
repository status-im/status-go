import copy
import pytest
from resources.constants import new_account_data_1


@pytest.mark.rpc
class TestDeleteAccount:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("sender")
        self.account_data = copy.deepcopy(new_account_data_1)

    def test_delete_watch_account(self):
        self.account_data["type"] = "watch"
        self.account.accounts_service.add_account("", self.account_data)

        accounts_response = self.account.accounts_service.get_accounts()
        assert len(accounts_response.get("result", [])) == 3

        resp = self.account.accounts_service.delete_account(self.account_data["address"], "")
        assert resp["result"] is None

        accounts_response = self.account.accounts_service.get_accounts()
        assert len(accounts_response.get("result", [])) == 2

    def test_delete_default_chat_account(self):
        accounts_response = self.account.accounts_service.get_accounts()
        assert len(accounts_response.get("result", [])) == 2

        resp = self.account.accounts_service.delete_account(accounts_response.get("result")[0].get("address"), self.account.password)
        assert resp.get("error").get("message") == "[database] cannot remove default chat account"

        accounts_response = self.account.accounts_service.get_accounts()
        assert len(accounts_response.get("result", [])) == 2

    def test_delete_default_wallet_account(self):
        accounts_response = self.account.accounts_service.get_accounts()
        assert len(accounts_response.get("result", [])) == 2

        resp = self.account.accounts_service.delete_account(accounts_response.get("result")[1].get("address"), self.account.password)
        assert resp.get("error").get("message") == "[database] cannot remove default wallet account"

        accounts_response = self.account.accounts_service.get_accounts()
        assert len(accounts_response.get("result", [])) == 2

    def test_delete_nonexistent_account(self):
        resp = self.account.accounts_service.delete_account("0x0000000000000000000000000000000000000001", self.account.password)
        assert resp.get("error").get("message") == "accounts: account is not found"
