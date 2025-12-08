import copy
import re
import pytest
from resources.constants import new_account_data_1
from clients.api import ApiResponseError


@pytest.mark.rpc
class TestUpdateAccount:

    @pytest.fixture()
    def backend(self, backend_new_profile):
        return backend_new_profile("account-backend")

    @pytest.fixture()
    def account_data(self):
        """Fresh copy of base account data for each test."""
        return copy.deepcopy(new_account_data_1)

    def test_update_editable_fields_to_all_accounts(self, backend, account_data):
        # create also a watch account
        account_data["type"] = "watch"
        backend.accounts_service.add_account(backend.password, account_data)

        # fetch all accounts(chat, wallet and watch) for update
        accounts_response_before = backend.accounts_service.get_accounts()
        accounts_before = accounts_response_before

        for before in accounts_before:
            # modify account fields and send update for all existing accounts
            before["colorId"] = "foo"
            before["hidden"] = True
            before["name"] = "UpdatedName"
            before["emoji"] = "✨"
            before["prodPreferredChainIds"] = "2:10:42161:8458"
            update_resp = backend.accounts_service.update_account(before)
            assert update_resp is None

        # verify update persisted
        accounts_response_after = backend.accounts_service.get_accounts()
        accounts_after = accounts_response_after
        assert len(accounts_before) == len(accounts_after)
        for before, after in zip(accounts_before, accounts_after):
            # making clocks equal so we can compare the entire account object
            before["clock"] = after["clock"] = 0
            assert after == before

    def test_try_to_update_non_editable_fields(self, backend):
        # fetch all accounts
        accounts_response_before = backend.accounts_service.get_accounts()
        accounts_before = accounts_response_before

        for before in accounts_before:
            before_copy = copy.deepcopy(before)
            before_copy["chat"] = not (before_copy["chat"])
            before_copy["mixedcase-address"] = "0xD0CDe0845A33d93e371fF272642026Ed945bcb0B"
            before_copy["operable"] = "partially"
            before_copy["path"] = "m/44'/60'/0'/0/1"
            before_copy["position"] = 5
            before_copy["public-key"] = (
                "0x04bf7c79e1c7375ac632d4b444ffee8e2d5a8158e70a4d54c50fb92047bc8b3d17329ad361b9b8e00c916e50617d72dc2bb42e37c3fc2c4ee3f367121ec397742d"
            )
            before_copy["type"] = "key"
            before_copy["wallet"] = not (before_copy["wallet"])
            backend.accounts_service.update_account(before_copy)

        # verify that no update was made to non editable fields
        accounts_response_after = backend.accounts_service.get_accounts()
        accounts_after = accounts_response_after
        assert len(accounts_before) == len(accounts_after)
        for before, after in zip(accounts_before, accounts_after):
            # making clocks equal so we can compare the entire account object
            before["clock"] = after["clock"] = 0
            assert after == before

    def test_update_nonexistent_account(self, backend):
        nonexisting = {"address": "0x0000000000000000000000000000000000000001", "name": "Nope"}
        with pytest.raises(ApiResponseError, match=re.escape("cannot update non-existing account: accounts: account is not found")):
            backend.accounts_service.update_account(nonexisting)
