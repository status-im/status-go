import copy
import pytest
from resources.constants import new_account_data_1


@pytest.mark.rpc
class TestMoveWalletAccount:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("sender")
        self.account_data = copy.deepcopy(new_account_data_1)

    def test_move_non_profile_wallet_accounts(self):
        # Add new account
        path = "m/44'/60'/0'/0/1"
        derived_addresses_response = self.account.wallet_service.get_derived_addresses_for_mnemonic(self.account.mnemonic, [path])
        derived_addresses = derived_addresses_response.json()["result"]
        self.account_data["key-uid"] = self.account.key_uid  # keyuid of profile keypair
        self.account_data["path"] = path
        self.account_data["address"] = derived_addresses[0].get("address")
        self.account_data["public-key"] = derived_addresses[0].get("public-key")
        self.account.accounts_service.add_account(self.account.password, self.account_data)

        # Get all accounts to determine positions
        accounts_response = self.account.accounts_service.get_accounts()
        accounts_before = [acc for acc in accounts_response.get("result", [])]
        assert len(accounts_before) >= 3, "Need at least two accounts to test move"

        # Move wallet account from position 1 to position 2
        move_accounts_response = self.account.accounts_service.move_wallet_account(
            accounts_before[1].get("position"), accounts_before[2].get("position")
        )
        assert "result" in move_accounts_response
        assert move_accounts_response.get("result") is None

        # Fetch the accounts again
        accounts_response_after = self.account.accounts_service.get_accounts()
        accounts_after = [acc for acc in accounts_response_after.get("result", [])]

        # Make the positions equal so we can verify that accounts match after move
        # Ex Account at position 1 becomes position 2
        accounts_before[1]["position"] = accounts_before[2]["position"] = accounts_after[1]["position"] = accounts_after[2]["position"] = 0
        assert accounts_before[1] == accounts_after[2]
        assert accounts_before[2] == accounts_after[1]

    def test_move_profile_account_isnt_allowed(self):
        # Get all accounts to determine positions
        accounts_response = self.account.accounts_service.get_accounts()
        accounts_before = [acc for acc in accounts_response.get("result", [])]
        assert len(accounts_before) >= 2, "Need at least two accounts to test move"

        # Move wallet account from position 0 to position 1
        move_accounts_response = self.account.accounts_service.move_wallet_account(
            accounts_before[0].get("position"), accounts_before[1].get("position")
        )
        assert move_accounts_response.get("error").get("message") == "accounts: trying to move account to a wrong position"
