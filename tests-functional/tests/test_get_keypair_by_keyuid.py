import pytest
from resources.constants import user_1


@pytest.mark.rpc
class TestGetKeypairByKeyUID:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("sender")

    def test_get_keypair_by_existing_keyuid(self):
        get_account_keypairs_resp = self.account.accounts_service.get_account_keypairs()
        get_keypair_by_key_uid_resp = self.account.accounts_service.get_keypair_by_key_uid(self.account.key_uid)
        keypair = get_keypair_by_key_uid_resp.get("result")
        assert keypair == get_account_keypairs_resp.get("result")[0]
        assert keypair.get("name") == self.account.display_name
        accounts = keypair.get("accounts")
        assert accounts[0].get("path") == "m/43'/60'/1581'/0'/0"
        assert accounts[0].get("name") == ""
        assert accounts[0].get("chat") is True
        assert accounts[0].get("wallet") is False
        assert accounts[0].get("prodPreferredChainIds") == "1:10:42161:8453"
        assert accounts[0].get("type") == "generated"
        assert accounts[1].get("path") == "m/44'/60'/0'/0/0"
        assert accounts[1].get("name") == "Account 1"
        assert accounts[1].get("chat") is False
        assert accounts[1].get("wallet") is True
        assert accounts[0].get("address") != accounts[1].get("address")
        assert accounts[0].get("key-uid") == accounts[1].get("key-uid")

    def test_get_keypair_by_nonexistent_keyuid(self):
        resp = self.account.accounts_service.get_keypair_by_key_uid("0x6d462df35b97fabb8f792eac01240556e26fd2600753e5bbffa4713a9c95abc7")
        assert "keypair is not found" in resp.get("error").get("message")

    def test_get_newly_imported_keypair(self):
        keypair_name = "SeedImportedKeypair"
        wallet_account_details = {
            "name": "SeedImportedAccount",
            "path": "m/44'/60'/0'/0/0",
            "emoji": "🔑",
            "colorId": "primary",
        }
        self.account.accounts_service.add_keypair_via_seed_phrase(user_1.passphrase, self.account.password, keypair_name, wallet_account_details)
        keypairs_response = self.account.accounts_service.get_account_keypairs()
        imported_keypair = [keypair for keypair in keypairs_response.get("result", []) if keypair.get("name") == keypair_name][0]
        imported_keypair_key_uid = imported_keypair.get("key-uid")
        resp = self.account.accounts_service.get_keypair_by_key_uid(imported_keypair_key_uid)
        assert imported_keypair == resp.get("result")
