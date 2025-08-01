import pytest

# from clients.status_backend import StatusBackend
from resources.constants import new_account_data_1, new_account_data_2


@pytest.mark.rpc
class TestAddAccount:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        """Initialize two unprivileged backends (sender and receiver) for each test function"""
        self.account = backend_new_profile("sender")

    def test_add_account_for_valid_key_uid(self):
        # Get keypairs to find the key-uid
        keypairs_response = self.account.accounts_service.get_account_keypairs()
        keypairs = keypairs_response.get("result", [])
        assert len(keypairs) > 0
        key_uid = keypairs[0]["key-uid"]

        # Add new account for the existing key-uid
        account_data = new_account_data_1
        account_data["key-uid"] = key_uid
        add_account_response = self.account.accounts_service.add_account(account_data)
        assert "error" not in add_account_response

        # After adding the account check that the get accounts will retrieve the created account
        accounts_response = self.account.accounts_service.get_accounts()
        accounts = accounts_response.get("result", [])

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

    def test_add_account_for_unknown_key_uid(self):
        account_data = new_account_data_1
        account_data["key-uid"] = "0x3231d92c94548d14f097173765a50bebe28fbad8f2267c9e08cc4433a6f219a4"
        add_account_response = self.account.accounts_service.add_account(account_data, skip_validation=True)
        assert add_account_response.get("error", {}).get("message", "") == "cannot add an account for an unknown keypair"

    def test_add_account_for_empty_key_uid(self):
        account_data = new_account_data_1
        account_data["key-uid"] = ""
        add_account_response = self.account.accounts_service.add_account(account_data, skip_validation=True)
        assert add_account_response.get("error", {}).get("message", "") == "`KeyUID` field of an account must be set"

    def test_add_seconds_account_for_same_keypair(self):
        # Get keypairs to find the key-uid
        keypairs_response = self.account.accounts_service.get_account_keypairs()
        keypairs = keypairs_response.get("result", [])
        assert len(keypairs) > 0
        key_uid = keypairs[0]["key-uid"]

        # Add new account for the existing key-uid
        account_data = new_account_data_1
        account_data["key-uid"] = key_uid
        add_account_response = self.account.accounts_service.add_account(account_data)
        assert "error" not in add_account_response

        # Add a second account
        account_data = new_account_data_2
        account_data["key-uid"] = key_uid
        add_second_account_response = self.account.accounts_service.add_account(account_data)
        assert "error" not in add_second_account_response

    def test_add_duplicate_account(self):
        # Get keypairs to find the key-uid
        keypairs_response = self.account.accounts_service.get_account_keypairs()
        keypairs = keypairs_response.get("result", [])
        assert len(keypairs) > 0
        key_uid = keypairs[0]["key-uid"]

        # Add new account for the existing key-uid
        account_data = new_account_data_1
        account_data["key-uid"] = key_uid
        add_account_response = self.account.accounts_service.add_account(account_data)
        assert "error" not in add_account_response

        # Add same account again
        add_account_response = self.account.accounts_service.add_account(account_data, skip_validation=True)
        assert add_account_response.get("error", {}).get("message", "") == "account already exists"

    def test_add_new_keypair_and_add_account_for_it(self):
        # Import mnemonic to create keypair
        mnemonic = "test test test test test test test test test test test junk"
        password = "testpassword"

        response = self.account.accounts_service.import_mnemonic(mnemonic, password)
        assert "error" not in response

        # Get keypairs to find the key-uid and full keypair data
        response = self.account.accounts_service.get_account_keypairs()
        keypairs = response.get("result", [])
        assert len(keypairs) > 0
        keypair = keypairs[0]

        account_data = new_account_data_1
        account_data["key-uid"] = keypair["key-uid"]

        # Filter out default wallet and chat accounts before adding a new keypair
        keypair["accounts"] = [account_data]

        # Call accounts_addKeypair with the retrieved keypair data
        response = self.account.accounts_service.add_keypair(password, keypair)
