import copy
import re
import pytest
from resources.constants import new_account_data_1
from clients.api import ApiResponseError


@pytest.mark.rpc
class TestMoveWalletAccount:

    @pytest.fixture()
    def backend(self, backend_new_profile):
        return backend_new_profile("account-backend")

    @pytest.fixture()
    def account_data(self):
        """Fresh copy of base account data for each test."""
        return copy.deepcopy(new_account_data_1)

    def test_move_non_profile_wallet_accounts(self, backend, account_data):
        # Add new account
        path = "m/44'/60'/0'/0/1"
        derived_addresses_response = backend.wallet_service.get_derived_addresses_for_mnemonic(backend.mnemonic, [path])
        derived_addresses = derived_addresses_response
        account_data["key-uid"] = backend.key_uid  # keyuid of profile keypair
        account_data["path"] = path
        account_data["address"] = derived_addresses[0].get("address")
        account_data["public-key"] = derived_addresses[0].get("public-key")
        backend.accounts_service.add_account(backend.password, account_data)

        # Get all accounts to determine positions
        accounts_response = backend.accounts_service.get_accounts()
        accounts_before = [acc for acc in accounts_response]
        assert len(accounts_before) >= 3, "Need at least two accounts to test move"

        # Move wallet account from position 1 to position 2
        move_accounts_response = backend.accounts_service.move_wallet_account(accounts_before[1].get("position"), accounts_before[2].get("position"))
        assert move_accounts_response is None

        # Fetch the accounts again
        accounts_response_after = backend.accounts_service.get_accounts()
        accounts_after = [acc for acc in accounts_response_after]

        # Checking the positions are still incremental after move
        assert accounts_before[0]["position"] == -1
        assert accounts_before[1]["position"] == 0
        assert accounts_before[2]["position"] == 1
        assert accounts_after[0]["position"] == -1
        assert accounts_after[1]["position"] == 0
        assert accounts_after[2]["position"] == 1

        # Make the positions equal so we can verify that entire account objects match after move
        # Ex Account at position 1 becomes position 2
        accounts_before[1]["position"] = accounts_before[2]["position"] = accounts_after[1]["position"] = accounts_after[2]["position"] = 0

        assert accounts_before[1] == accounts_after[2]
        assert accounts_before[2] == accounts_after[1]

    def test_move_profile_account_isnt_allowed(self, backend):
        # Get all accounts to determine positions
        accounts_response = backend.accounts_service.get_accounts()
        accounts_before = [acc for acc in accounts_response]
        assert len(accounts_before) >= 2, "Need at least two accounts to test move"

        # Move wallet account from position 0 to position 1
        with pytest.raises(ApiResponseError, match=re.escape("accounts: trying to move account to a wrong position")):
            backend.accounts_service.move_wallet_account(accounts_before[0].get("position"), accounts_before[1].get("position"))
