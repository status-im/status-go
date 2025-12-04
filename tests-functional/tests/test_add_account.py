import copy
import re
import pytest
from resources.constants import new_account_data_1, user_1, wallet_account_details_derivation, keypair_name
from clients.api import ApiResponseError


@pytest.mark.rpc
class TestAddAccount:

    @pytest.fixture()
    def account(self, backend_new_profile):
        return backend_new_profile("account-backend")

    @pytest.fixture()
    def account_data(self):
        return copy.deepcopy(new_account_data_1)

    def test_add_second_wallet_account_to_profile_keypair(self, account, account_data):
        # add account on path m/44'/60'/0'/0/1 to profile keypair
        path = "m/44'/60'/0'/0/1"
        derived_addresses_response = account.wallet_service.get_derived_addresses_for_mnemonic(account.mnemonic, [path])

        # update account being added with necessary details
        account_data["key-uid"] = account.key_uid  # keyuid of profile keypair
        account_data["path"] = path
        account_data["address"] = derived_addresses_response[0].get("address")
        account_data["public-key"] = derived_addresses_response[0].get("public-key")

        account.accounts_service.add_account(account.password, account_data)
        # TODO: Add more assertions on response

        # After adding the account check that the get accounts will retrieve the new account
        accounts_response = account.accounts_service.get_accounts()
        self.check_new_account_was_created(accounts_response, account_data)

    def test_add_duplicate_account(self, account):
        # since default wallet account is already added by creating the backend profile, we can try just adding the same account again
        accounts_response = account.accounts_service.get_accounts()
        assert len(accounts_response) == 2
        default_account = accounts_response[0]
        if not default_account["wallet"]:
            default_account = accounts_response[1]
        assert default_account["wallet"] is True

        # Add the same account second time
        with pytest.raises(ApiResponseError, match=re.escape("account already exists")):
            account.accounts_service.add_account(account.password, default_account)

    def test_add_account_for_unknown_key_uid(self, account, account_data):
        account_data["key-uid"] = "0x3231d92c94548d14f097173765a50bebe28fbad8f2267c9e08cc4433a6f219a4"
        with pytest.raises(ApiResponseError, match=re.escape("keypair is not found")):
            account.accounts_service.add_account(account.password, account_data)

    def test_add_account_for_empty_address(self, account, account_data):
        account_data["address"] = ""
        with pytest.raises(ApiResponseError, match=re.escape("invalid argument 1: hex string has length 0, want 40 for types.Address")):
            account.accounts_service.add_account(account.password, account_data)

    def test_add_account_for_empty_path(self, account, account_data):
        account_data["path"] = ""
        with pytest.raises(ApiResponseError, match=re.escape("[account] account mismatch")):
            account.accounts_service.add_account(account.password, account_data)

    @pytest.mark.parametrize(
        "key, error", [("wallet", "[database] cannot add default wallet account"), ("chat", "[database] cannot add default chat account")]
    )
    def test_add_account_with_key_set_on_true__(self, account, account_data, key, error):
        account_data["key-uid"] = account.key_uid
        account_data[key] = True
        with pytest.raises(ApiResponseError, match=re.escape(error)):
            account.accounts_service.add_account(account.password, account_data)

    def test_add_watch_account(self, account, account_data):
        account_data["type"] = "watch"
        account.accounts_service.add_account(account.password, account_data)
        # TODO: Add more assertions on response

        # After adding the account check that the get accounts will retrieve the new account
        accounts_response = account.accounts_service.get_accounts()
        accounts = accounts_response
        self.check_new_account_was_created(accounts, account_data)

    def test_add_account_to_seed_imported_keypair(self, account, account_data):
        used_mnemonic = user_1.passphrase
        profile_password = account.password
        add_keypair_response = account.accounts_service.add_keypair_via_seed_phrase(
            used_mnemonic, profile_password, keypair_name, wallet_account_details_derivation
        )

        path = "m/44'/60'/0'/0/1"
        derived_addresses_response = account.wallet_service.get_derived_addresses_for_mnemonic(used_mnemonic, [path])
        derived_addresses = derived_addresses_response

        # update account being added with necessary details
        account_data["key-uid"] = add_keypair_response["key-uid"]
        account_data["path"] = path
        account_data["address"] = derived_addresses[0].get("address")
        account_data["public-key"] = derived_addresses[0].get("public-key")

        account.accounts_service.add_account(profile_password, account_data)
        # TODO: Add more assertions on response

    @staticmethod
    def get_account_keypairs(account):
        keypairs_response = account.accounts_service.get_account_keypairs()
        keypairs = keypairs_response
        assert len(keypairs) > 0
        return keypairs

    @staticmethod
    def check_new_account_was_created(accounts, account_data):
        mismatch_details = ""
        for acc in accounts:
            mismatches = []
            for key, value in account_data.items():
                if key not in acc:
                    mismatches.append(f"Missing key: {key}")
                    continue
                actual_value = acc[key]
                if key == "address" and value.lower() == actual_value.lower():
                    continue
                if value == actual_value:
                    continue
                mismatches.append(f"Mismatch in key '{key}': expected '{value}', got '{actual_value}'")
            if not mismatches:
                break
            else:
                mismatch_details = "\\n".join(mismatches)
        else:
            assert False, f"Added account not found or mismatched in accounts list. Details:\\n{mismatch_details}"
