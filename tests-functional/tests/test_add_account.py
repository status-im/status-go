import pytest

# from clients.status_backend import StatusBackend
from resources.constants import new_account_data_1


@pytest.mark.rpc
class TestAddAccount:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        """Initialize two unprivileged backends (sender and receiver) for each test function"""
        self.account = backend_new_profile("sender")

    def test_add_valid_account(self):
        # Get keypairs to find the key-uid
        keypairs_response = self.account.accounts_service.get_account_keypairs()
        keypairs = keypairs_response.get("result", [])
        assert len(keypairs) > 0
        key_uid = keypairs[0]["key-uid"]

        self.account.accounts_service.get_accounts()

        account_data = new_account_data_1
        account_data["key-uid"] = key_uid
        add_account_response = self.account.accounts_service.add_account(account_data)
        assert "error" not in add_account_response

        self.account.accounts_service.get_accounts()

    # def test_add_account_for_unknown_key(self):
    #     backend_client = StatusBackend(self.await_signals)
    #     backend_client.init_status_backend()
    #     backend_client.create_account_and_login()
    #     backend_client.wait_for_login()

    #     # Add a account associated with no keypair
    #     account_data = {
    #         "address": "0x1234567890abcdef1234567890abcdef12345678",  # Use an address derived from the mnemonic
    #         "key-uid": "0x50061d0119f732400c8c11b760d5a1cf472fa3baaaae88b4bc238b110b2939da",
    #         "wallet": False,
    #         "chat": False,
    #         "type": "generated",
    #         "path": "m/44'/60'/0'/0/0",
    #         "public-key": "0xabcdef",
    #         "name": "generated-account",
    #         "emoji": "🔑",
    #         "colorId": "blue",
    #         "hidden": False,
    #     }

    #     response = backend_client.rpc_valid_request("accounts_addAccount", ["", account_data])
    #     assert response.status_code == 200
    #     result = response.json()
    #     assert "error" not in result

    # def test_add_duplicate_account22(self):
    #     backend_client = StatusBackend(self.await_signals)
    #     backend_client.init_status_backend()
    #     backend_client.create_account_and_login()
    #     backend_client.wait_for_login()

    #     # Get keypairs to find the key-uid
    #     response = backend_client.rpc_valid_request("accounts_getKeypairs", [])
    #     keypairs = response.json().get("result", [])
    #     assert len(keypairs) > 0
    #     key_uid = keypairs[0]["key-uid"]

    #     account_data = {
    #         "address": "0x1234567890abcdef1234567890abcdef12345678",
    #         "key-uid": key_uid,
    #         "wallet": False,
    #         "chat": False,
    #         "type": "generated",
    #         "path": "m/44'/60'/0'/0/0",
    #         "public-key": "0xabcdef",
    #         "name": "generated-account",
    #         "emoji": "🔑",
    #         "colorId": "blue",
    #         "hidden": False,
    #     }

    #     # Add first time
    #     response1 = backend_client.rpc_valid_request("accounts_addAccount", ["", account_data])
    #     assert response1.status_code == 200
    #     result1 = response1.json()
    #     assert "error" not in result1

    #     # Add duplicate
    #     response2 = backend_client.rpc_valid_request("accounts_addAccount", ["", account_data])
    #     assert response2.status_code != 200 or "error" in response2.text

    # def test_add_new_keypair_and_add_account_for_it(self):
    #     backend_client = StatusBackend(self.await_signals)
    #     backend_client.init_status_backend()
    #     backend_client.create_account_and_login()
    #     backend_client.wait_for_login()

    #     # Import mnemonic to create keypair
    #     mnemonic = "test test test test test test test test test test test junk"
    #     password = "testpassword"

    #     response = backend_client.rpc_valid_request("accounts_importMnemonic", [mnemonic, password])
    #     assert response.status_code == 200
    #     result = response.json()
    #     assert "error" not in result

    #     # Get keypairs to find the key-uid and full keypair data
    #     response = backend_client.rpc_valid_request("accounts_getKeypairs", [])
    #     keypairs = response.json().get("result", [])
    #     assert len(keypairs) > 0
    #     keypair = keypairs[0]

    #     # Filter out default wallet and chat accounts before calling accounts_addKeypair
    #     filtered_accounts = [acc for acc in keypair["accounts"] if not acc.get("wallet", False) and not acc.get("chat", False)]
    #     keypair["accounts"] = filtered_accounts

    #     # Call accounts_addKeypair with the retrieved keypair data
    #     response = backend_client.rpc_valid_request("accounts_addKeypair", [password, keypair])
    #     assert response.status_code == 200
    #     result = response.json()
    #     assert "error" not in result

    #     # Now add an account associated with the imported keypair
    #     account_data = {
    #         "address": "0x1234567890abcdef1234567890abcdef12345678",  # Use an address derived from the mnemonic
    #         "key-uid": keypair["key-uid"],  # Use the key-uid from the keypair
    #         "wallet": False,
    #         "chat": False,
    #         "type": "generated",
    #         "path": "m/44'/60'/0'/0/0",
    #         "public-key": "0xabcdef",
    #         "name": "generated-account",
    #         "emoji": "🔑",
    #         "colorId": "blue",
    #         "hidden": False,
    #     }

    #     response = backend_client.rpc_valid_request("accounts_addAccount", ["", account_data])
    #     assert response.status_code == 200
    #     result = response.json()
    #     assert "error" not in result

    #     response = backend_client.rpc_valid_request("accounts_getKeypairs", [])
    #     response = backend_client.rpc_valid_request("accounts_getAccounts", [])
