import pytest
from resources.constants import user_1, user_2

KEYPAIR_NAME = "PrivateKeyImportedKeypair"
WALLET_ACCOUNT_DETAILS = {
    "name": KEYPAIR_NAME,
    "path": "m",
    "emoji": "🔑",
    "colorId": "primary",
}


@pytest.mark.rpc
class TestAddKeypairViaPrivateKey:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("sender")

    def test_add_valid_keypair_via_private_key(self):
        add_keypair_response = self.account.accounts_service.add_keypair_via_private_key(
            user_1.private_key, self.account.password, KEYPAIR_NAME, WALLET_ACCOUNT_DETAILS
        )
        new_keypair = add_keypair_response.get("result").get("accounts")[0]
        assert new_keypair.get("address").lower() == user_1.address.lower()
        assert new_keypair.get("chat") is False
        assert new_keypair.get("clock") == 0
        assert new_keypair.get("colorId") == WALLET_ACCOUNT_DETAILS.get("colorId")
        assert new_keypair.get("createdAt") == 0
        assert new_keypair.get("emoji") == WALLET_ACCOUNT_DETAILS.get("emoji")
        assert new_keypair.get("hidden") is False
        assert new_keypair.get("name") == KEYPAIR_NAME
        assert new_keypair.get("operable") == "fully"
        assert new_keypair.get("path") == WALLET_ACCOUNT_DETAILS.get("path")
        assert new_keypair.get("position") == 1
        assert new_keypair.get("removed") is False
        assert new_keypair.get("type") == ""
        assert new_keypair.get("wallet") is False

        # Fetch keypairs and ensure the imported one is present
        keypairs_response = self.account.accounts_service.get_account_keypairs()
        imported_keypair = [keypair for keypair in keypairs_response.get("result", []) if keypair.get("name") == KEYPAIR_NAME][0]
        assert add_keypair_response.get("result").get("key-uid") == imported_keypair.get("key-uid")

    def test_add_a_second_keypair_via_pk_with_same_details(self):
        self.account.accounts_service.add_keypair_via_private_key(user_1.private_key, self.account.password, KEYPAIR_NAME, WALLET_ACCOUNT_DETAILS)

        # different private key but same details
        self.account.accounts_service.add_keypair_via_private_key(user_2.private_key, self.account.password, KEYPAIR_NAME, WALLET_ACCOUNT_DETAILS)

        keypairs_response = self.account.accounts_service.get_account_keypairs()
        imported_keypairs = [keypair for keypair in keypairs_response.get("result", []) if keypair.get("name") == KEYPAIR_NAME]
        assert len(imported_keypairs) == 2, "2 keypairs with the same name should be saved"

    def test_add_duplicate_keypair_via_pk(self):
        resp1 = self.account.accounts_service.add_keypair_via_private_key(
            user_1.private_key, self.account.password, KEYPAIR_NAME, WALLET_ACCOUNT_DETAILS
        )

        # same private key
        resp2 = self.account.accounts_service.add_keypair_via_private_key(
            user_1.private_key, self.account.password, KEYPAIR_NAME, WALLET_ACCOUNT_DETAILS
        )
        assert resp2.get("error").get("message") == f'[validation] keypair already added -  keyuid: {resp1.get("result").get("key-uid")}'

    def test_add_keypair_via_pk_with_wrong_path(self):
        details = {
            "name": "name",
            "path": "m/44'/60'/0'/0/0",
            "emoji": "🔑",
            "colorId": "primary",
        }
        resp = self.account.accounts_service.add_keypair_via_private_key(user_1.private_key, self.account.password, KEYPAIR_NAME, details)
        assert (
            resp.get("error").get("message")
            == f'[validation] unsupported profile or seed imported key pair wallet account -  path: {details["path"]} expected path: m'
        )

    def test_add_keypair_via_pk_with_wrong_password(self):
        resp = self.account.accounts_service.add_keypair_via_private_key(user_1.private_key, "wrongpass", KEYPAIR_NAME, WALLET_ACCOUNT_DETAILS)
        assert "error" in resp
