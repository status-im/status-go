import copy
import pytest
from resources.constants import new_account_data_1, user_1


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
        derived_addresses = derived_addresses_response.json()["result"]

        # update account being added with necessary details
        self.account_data["key-uid"] = self.account.key_uid  # keyuid of profile keypair
        self.account_data["path"] = path
        self.account_data["address"] = derived_addresses[0].get("address")
        self.account_data["public-key"] = derived_addresses[0].get("public-key")

        add_account_response = self.account.accounts_service.add_account(self.account.password, self.account_data)
        self.account.verify_json_schema(add_account_response, method="accounts_addAccount")
        assert "error" not in add_account_response

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
        add_account_response = self.account.accounts_service.add_account(self.account.password, defaultAccount, skip_validation=True)
        assert add_account_response.get("error", {}).get("message", "") == "account already exists"

    def test_add_new_keypair_via_seed_phrase(self):
        keypair_name = "SeedImportedKeypair"
        wallet_account_details = {
            "name": "SeedImportedAccount",
            "path": "m/44'/60'/0'/0/0",
            "emoji": "🔑",
            "colorId": "primary",
        }
        add_keypair_response = self.account.accounts_service.add_keypair_via_seed_phrase(
            user_1.passphrase, self.account.password, keypair_name, wallet_account_details
        )
        self.account.verify_json_schema(add_keypair_response, method="accounts_addKeypairViaSeedPhrase")

        keypairs = self.get_account_keypairs()
        assert len(keypairs) == 2
        imported_keypair = keypairs[0]
        if imported_keypair["name"] != keypair_name:
            imported_keypair = keypairs[1]
        assert imported_keypair["name"] == keypair_name

        assert imported_keypair["key-uid"] == add_keypair_response["result"]["key-uid"]
        assert imported_keypair["type"] == "seed"
        assert add_keypair_response["result"]["type"] == "seed"
        assert add_keypair_response["result"]["key-uid"] == imported_keypair["key-uid"]
        assert add_keypair_response["result"]["derived-from"] == imported_keypair["derived-from"]
        assert len(add_keypair_response["result"]["accounts"]) == 1
        assert len(imported_keypair["accounts"]) == 1

    def test_add_account_for_unknown_key_uid(self):
        self.account_data["key-uid"] = "0x3231d92c94548d14f097173765a50bebe28fbad8f2267c9e08cc4433a6f219a4"
        add_account_response = self.account.accounts_service.add_account(self.account.password, self.account_data, skip_validation=True)
        assert add_account_response.get("error", {}).get("message", "") == "keypair is not found"

    def test_add_account_for_empty_address(self):
        self.account_data["address"] = ""
        add_account_response = self.account.accounts_service.add_account(self.account.password, self.account_data, skip_validation=True)
        assert add_account_response.get("error", {}).get("message", "") == "invalid argument 1: hex string has length 0, want 40 for types.Address"

    def test_add_account_for_empty_path(self):
        self.account_data["path"] = ""
        add_account_response = self.account.accounts_service.add_account(self.account.password, self.account_data, skip_validation=True)
        expected_error = "[account] account mismatch"
        expected_error_context = "address: " + self.account_data["address"]
        error_message = add_account_response.get("error", {}).get("message", "")
        assert expected_error in error_message
        assert expected_error_context.lower() in error_message.lower()

    @pytest.mark.parametrize("key", ["wallet", "chat"])
    def test_add_account_with_key_set_on_true__(self, key):
        self.account_data["key-uid"] = self.account.key_uid
        self.account_data[key] = True
        add_account_response = self.account.accounts_service.add_account(self.account.password, self.account_data, skip_validation=True)
        if key == "wallet":
            assert add_account_response.get("error", {}).get("message", "") == "[database] cannot add default wallet account"
        else:
            assert add_account_response.get("error", {}).get("message", "") == "[database] cannot add default chat account"

    def test_add_watch_account(self):
        self.account_data["type"] = "watch"
        add_account_response = self.account.accounts_service.add_account(self.account.password, self.account_data)
        self.account.verify_json_schema(add_account_response, method="accounts_addAccount")
        assert "error" not in add_account_response

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
        derived_addresses = derived_addresses_response.json()["result"]

        # update account being added with necessary details
        self.account_data["key-uid"] = add_keypair_response["result"]["key-uid"]
        self.account_data["path"] = path
        self.account_data["address"] = derived_addresses[0].get("address")
        self.account_data["public-key"] = derived_addresses[0].get("public-key")

        add_account_response = self.account.accounts_service.add_account(profile_password, self.account_data)
        self.account.verify_json_schema(add_account_response, method="accounts_addAccount")
        assert "error" not in add_account_response

    def test_delete_account(self):
        self.account_data["type"] = "watch"
        add_account_response = self.account.accounts_service.add_account(self.account.password, self.account_data)
        self.account.verify_json_schema(add_account_response, method="accounts_addAccount")
        assert "error" not in add_account_response

        accounts_response = self.account.accounts_service.get_accounts()
        assert len(accounts_response.get("result", [])) == 3

        # Delete the account
        delete_response = self.account.accounts_service.delete_account(self.account_data["address"])
        assert "error" not in delete_response
        self.account.verify_json_schema(delete_response, method="accounts_deleteAccount")

        accounts_response = self.account.accounts_service.get_accounts()
        assert len(accounts_response.get("result", [])) == 2

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
