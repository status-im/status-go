import copy
import re
import pytest
from resources.constants import new_account_data_1, user_1
from clients.api import ApiResponseError


@pytest.mark.rpc
class TestAddAccount:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("sender")
        self.account_data = copy.deepcopy(new_account_data_1)

    def test_add_second_wallet_account_to_profile_keypair(self):
        # add account on path m/44'/60'/0'/0/1 to profile keypair
        path = "m/44'/60'/0'/0/1"
        derived_addresses_response = self.account.wallet_service.get_derived_addresses_for_mnemonic(self.account.mnemonic, [path])
        derived_addresses = derived_addresses_response["result"]

        # update account being added with necessary details
        self.account_data["key-uid"] = self.account.key_uid  # keyuid of profile keypair
        self.account_data["path"] = path
        self.account_data["address"] = derived_addresses[0].get("address")
        self.account_data["public-key"] = derived_addresses[0].get("public-key")

        self.account.accounts_service.add_account(self.account.password, self.account_data)
        # TODO: Add more assertions on response

        # After adding the account check that the get accounts will retrieve the new account
        accounts_response = self.account.accounts_service.get_accounts()
        accounts = accounts_response.get("result", [])
        self.check_new_account_was_created(accounts, self.account_data)

    def test_add_duplicate_account(self):
        # since default wallet account is already added by creating the backend profile, we can try just adding the same account again
        accounts_response = self.account.accounts_service.get_accounts()
        assert len(accounts_response.get("result", [])) == 2
        defaultAccount = accounts_response.get("result", [])[0]
        if not defaultAccount["wallet"]:
            defaultAccount = accounts_response.get("result", [])[1]
        assert defaultAccount["wallet"] is True

        # Add the same account second time
        with pytest.raises(ApiResponseError, match=re.escape("account already exists")):
            self.account.accounts_service.add_account(self.account.password, defaultAccount)

    def test_add_account_for_unknown_key_uid(self):
        self.account_data["key-uid"] = "0x3231d92c94548d14f097173765a50bebe28fbad8f2267c9e08cc4433a6f219a4"
        with pytest.raises(ApiResponseError, match=re.escape("keypair is not found")):
            self.account.accounts_service.add_account(self.account.password, self.account_data)

    def test_add_account_for_empty_address(self):
        self.account_data["address"] = ""
        with pytest.raises(ApiResponseError, match=re.escape("invalid argument 1: hex string has length 0, want 40 for types.Address")):
            self.account.accounts_service.add_account(self.account.password, self.account_data)

    def test_add_account_for_empty_path(self):
        self.account_data["path"] = ""
        with pytest.raises(ApiResponseError, match=re.escape("[account] account mismatch")):
            self.account.accounts_service.add_account(self.account.password, self.account_data)

    @pytest.mark.parametrize(
        "key, error", [("wallet", "[database] cannot add default wallet account"), ("chat", "[database] cannot add default chat account")]
    )
    def test_add_account_with_key_set_on_true__(self, key, error):
        self.account_data["key-uid"] = self.account.key_uid
        self.account_data[key] = True
        with pytest.raises(ApiResponseError, match=re.escape(error)):
            self.account.accounts_service.add_account(self.account.password, self.account_data)

    def test_add_watch_account(self):
        self.account_data["type"] = "watch"
        self.account.accounts_service.add_account(self.account.password, self.account_data)
        # TODO: Add more assertions on response

        # After adding the account check that the get accounts will retrieve the new account
        accounts_response = self.account.accounts_service.get_accounts()
        accounts = accounts_response.get("result", [])
        self.check_new_account_was_created(accounts, self.account_data)

    def test_add_account_to_seed_imported_keypair(self):
        used_mnemonic = user_1.passphrase
        profile_password = self.account.password

        keypair_name = "SeedImportedKeypair"
        wallet_account_details = {
            "name": "SeedImportedAccount",
            "path": "m/44'/60'/0'/0/0",
            "emoji": "🔑",
            "colorId": "primary",
        }
        add_keypair_response = self.account.accounts_service.add_keypair_via_seed_phrase(
            used_mnemonic, profile_password, keypair_name, wallet_account_details
        )

        path = "m/44'/60'/0'/0/1"
        derived_addresses_response = self.account.wallet_service.get_derived_addresses_for_mnemonic(used_mnemonic, [path])
        derived_addresses = derived_addresses_response["result"]

        # update account being added with necessary details
        self.account_data["key-uid"] = add_keypair_response["result"]["key-uid"]
        self.account_data["path"] = path
        self.account_data["address"] = derived_addresses[0].get("address")
        self.account_data["public-key"] = derived_addresses[0].get("public-key")

        self.account.accounts_service.add_account(profile_password, self.account_data)
        # TODO: Add more assertions on response

    def get_account_keypairs(self):
        keypairs_response = self.account.accounts_service.get_account_keypairs()
        keypairs = keypairs_response.get("result", [])
        assert len(keypairs) > 0
        return keypairs

    def check_new_account_was_created(self, accounts, account_data):
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
