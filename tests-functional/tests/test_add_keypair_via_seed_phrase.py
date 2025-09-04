import re
import pytest
from resources.constants import user_1, user_2
from clients.api import ApiResponseError

KEYPAIR_NAME = "SeedPhraseImportedKeypair"
WALLET_ACCOUNT_DETAILS = {
    "name": KEYPAIR_NAME,
    "path": "m/44'/60'/0'/0/0",
    "emoji": "🔑",
    "colorId": "primary",
}


@pytest.mark.rpc
class TestAddKeypairViaSeedPhrase:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("sender")

    def test_add_valid_keypair_via_seed_phrase(self):
        add_keypair_response = self.account.accounts_service.add_keypair_via_seed_phrase(
            user_1.passphrase, self.account.password, KEYPAIR_NAME, WALLET_ACCOUNT_DETAILS
        )
        add_keypair_result = add_keypair_response.get("result")
        accounts = add_keypair_result.get("accounts")
        assert len(accounts) == 1
        new_keypair = accounts[0]
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
        assert add_keypair_result.get("type") == "seed"
        assert new_keypair.get("wallet") is False

        # Fetch keypairs and ensure the imported one is present
        get_keypairs_response = self.account.accounts_service.get_account_keypairs()
        imported_keypairs = [keypair for keypair in get_keypairs_response.get("result", []) if keypair.get("name") == KEYPAIR_NAME]
        assert len(imported_keypairs) == 1
        assert add_keypair_result.get("key-uid") == imported_keypairs[0].get("key-uid")
        assert add_keypair_result.get("type") == imported_keypairs[0].get("type")
        assert add_keypair_result.get("derived-from") == imported_keypairs[0].get("derived-from")

    def test_add_a_second_keypair_via_sp_with_same_details(self):
        self.account.accounts_service.add_keypair_via_seed_phrase(user_1.passphrase, self.account.password, KEYPAIR_NAME, WALLET_ACCOUNT_DETAILS)

        # different private key but same details
        self.account.accounts_service.add_keypair_via_seed_phrase(user_2.private_key, self.account.password, KEYPAIR_NAME, WALLET_ACCOUNT_DETAILS)

        keypairs_response = self.account.accounts_service.get_account_keypairs()
        imported_keypairs = [keypair for keypair in keypairs_response.get("result", []) if keypair.get("name") == KEYPAIR_NAME]
        assert len(imported_keypairs) == 2, "2 keypairs with the same name should be saved"

    def test_add_duplicate_keypair_via_sp(self):
        resp1 = self.account.accounts_service.add_keypair_via_seed_phrase(
            user_1.passphrase, self.account.password, KEYPAIR_NAME, WALLET_ACCOUNT_DETAILS
        )

        # same private key
        with pytest.raises(ApiResponseError, match=re.escape(f'[validation] keypair already added -  keyuid: {resp1.get("result").get("key-uid")}')):
            self.account.accounts_service.add_keypair_via_seed_phrase(user_1.passphrase, self.account.password, KEYPAIR_NAME, WALLET_ACCOUNT_DETAILS)

    def test_add_keypair_via_sp_with_wrong_path(self):
        details = {
            "name": "name",
            "path": "m",
            "emoji": "🔑",
            "colorId": "primary",
        }
        with pytest.raises(
            ApiResponseError,
            match=re.escape(
                f'[validation] unsupported profile or seed imported key pair wallet account -  path: {details["path"]} expected path: m/44\''
            ),
        ):
            self.account.accounts_service.add_keypair_via_seed_phrase(user_1.passphrase, self.account.password, KEYPAIR_NAME, details)

    def test_add_keypair_via_sp_with_empty_password(self):
        self.account.accounts_service.add_keypair_via_seed_phrase(user_1.passphrase, "", KEYPAIR_NAME, WALLET_ACCOUNT_DETAILS)
