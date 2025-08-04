import copy
import pytest
from resources.constants import new_account_data_1, new_account_data_2


@pytest.mark.rpc
class TestAddAccount:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("sender")
        self.account_data = copy.deepcopy(new_account_data_1)

    def test_add_account_for_valid_key_uid(self):
        self.account_data["key-uid"] = self.get_keypair_key_uid()
        add_account_response = self.account.accounts_service.add_account(self.account_data)
        self.account.verify_json_schema(add_account_response, method="accounts_addAccount")
        assert "error" not in add_account_response

        # After adding the account check that the get accounts will retrieve the new account
        accounts_response = self.account.accounts_service.get_accounts()
        accounts = accounts_response.get("result", [])
        self.check_new_account_was_created(accounts, self.account_data)

    def test_add_seconds_account_for_same_keypair(self):
        key_uid = self.get_keypair_key_uid()

        self.account_data["key-uid"] = key_uid
        add_account_response = self.account.accounts_service.add_account(self.account_data)
        assert "error" not in add_account_response

        # Add a second account
        self.account_data = new_account_data_2
        self.account_data["key-uid"] = key_uid
        add_second_account_response = self.account.accounts_service.add_account(self.account_data)
        assert "error" not in add_second_account_response

    def test_add_duplicate_account(self):
        self.account_data["key-uid"] = self.get_keypair_key_uid()
        add_account_response = self.account.accounts_service.add_account(self.account_data)
        assert "error" not in add_account_response

        # Add same account again
        add_account_response = self.account.accounts_service.add_account(self.account_data, skip_validation=True)
        assert add_account_response.get("error", {}).get("message", "") == "account already exists"

    def test_add_new_account_via_add_keypair(self):
        keypairs = self.get_account_keypairs()
        keypair = keypairs[0]
        self.account_data["key-uid"] = self.get_keypair_key_uid()

        # Add the new account to the keypair accounts
        keypair["accounts"] = [self.account_data]

        # Call accounts_addKeypair with the new account
        password = self.account.created_account.get("password")
        add_keypair_response = self.account.accounts_service.add_keypair(password, keypair)
        self.account.verify_json_schema(add_keypair_response, method="accounts_addKeypair")

        # After adding the account check that the get accounts will retrieve the new account
        accounts_response = self.account.accounts_service.get_accounts()
        accounts = accounts_response.get("result", [])
        self.check_new_account_was_created(accounts, self.account_data)

    def test_add_account_for_unknown_key_uid(self):
        self.account_data["key-uid"] = "0x3231d92c94548d14f097173765a50bebe28fbad8f2267c9e08cc4433a6f219a4"
        add_account_response = self.account.accounts_service.add_account(self.account_data, skip_validation=True)
        assert add_account_response.get("error", {}).get("message", "") == "cannot add an account for an unknown keypair"

    def test_add_account_for_empty_key_uid(self):
        self.account_data["key-uid"] = ""
        add_account_response = self.account.accounts_service.add_account(self.account_data, skip_validation=True)
        assert add_account_response.get("error", {}).get("message", "") == "`KeyUID` field of an account must be set"

    @pytest.mark.parametrize("key", ["wallet", "chat"])
    def test_add_account_with_key_set_on_true__(self, key):
        self.account_data["key-uid"] = self.get_keypair_key_uid()
        self.account_data[key] = True
        add_account_response = self.account.accounts_service.add_account(self.account_data, skip_validation=True)
        assert add_account_response.get("error", {}).get("message", "") == "default wallet and chat account cannot be added this way"

    def test_add_watch_account(self):
        self.account_data["type"] = "watch"
        add_account_response = self.account.accounts_service.add_account(self.account_data)
        self.account.verify_json_schema(add_account_response, method="accounts_addAccount")
        assert "error" not in add_account_response

        # After adding the account check that the get accounts will retrieve the new account
        accounts_response = self.account.accounts_service.get_accounts()
        accounts = accounts_response.get("result", [])
        self.check_new_account_was_created(accounts, self.account_data)

    def test_add_seed_account(self):
        self.account_data["key-uid"] = self.get_keypair_key_uid()
        self.account_data["type"] = "seed"
        add_account_response = self.account.accounts_service.add_account(self.account_data)
        self.account.verify_json_schema(add_account_response, method="accounts_addAccount")
        assert "error" not in add_account_response

    def test_delete_account(self):
        self.account_data["type"] = "watch"
        add_account_response = self.account.accounts_service.add_account(self.account_data)
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

    def get_keypair_key_uid(self):
        keypairs = self.get_account_keypairs()
        first_keypair = keypairs[0]
        return first_keypair.get("key-uid")

    def check_new_account_was_created(self, accounts, account_data):
        mismatch_details = ""
        for acc in accounts:
            mismatches = []
            for key, value in account_data.items():
                if key not in acc:
                    mismatches.append(f"Missing key: {key}")
                    continue
                actual_value = acc[key]
                if key == "address":
                    if value.lower() != actual_value.lower():
                        mismatches.append(f"Mismatch in key '{key}': expected '{value}', got '{actual_value}'")
                else:
                    if value != actual_value:
                        mismatches.append(f"Mismatch in key '{key}': expected '{value}', got '{actual_value}'")
            if not mismatches:
                break
            else:
                mismatch_details = "\\n".join(mismatches)
        else:
            assert False, f"Added account not found or mismatched in accounts list. Details:\\n{mismatch_details}"
