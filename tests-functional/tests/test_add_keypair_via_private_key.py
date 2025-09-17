import re
import pytest
from resources.constants import user_1, user_2, wallet_account_details_root, wallet_account_details_derivation, keypair_name
from clients.api import ApiResponseError


@pytest.mark.rpc
class TestAddKeypairViaPrivateKey:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("sender")

    def test_add_valid_keypair_via_private_key(self):
        add_keypair_response = self.account.accounts_service.add_keypair_via_private_key(
            user_1.private_key, self.account.password, keypair_name, wallet_account_details_root
        )
        add_keypair_result = add_keypair_response
        accounts = add_keypair_result.get("accounts")
        assert len(accounts) == 1
        new_keypair = accounts[0]
        assert new_keypair.get("address").lower() == user_1.address.lower()
        assert new_keypair.get("chat") is False
        assert new_keypair.get("clock") == 0
        assert new_keypair.get("colorId") == wallet_account_details_root.get("colorId")
        assert new_keypair.get("createdAt") == 0
        assert new_keypair.get("emoji") == wallet_account_details_root.get("emoji")
        assert new_keypair.get("hidden") is False
        assert new_keypair.get("name") == keypair_name
        assert new_keypair.get("operable") == "fully"
        assert new_keypair.get("path") == wallet_account_details_root.get("path")
        assert new_keypair.get("position") == 1
        assert new_keypair.get("removed") is False
        assert new_keypair.get("type") == ""
        assert add_keypair_result.get("type") == "key"
        assert new_keypair.get("wallet") is False

        # Fetch keypairs and ensure the imported one is present
        get_keypairs_response = self.account.accounts_service.get_account_keypairs()
        imported_keypairs = [keypair for keypair in get_keypairs_response if keypair.get("name") == keypair_name]
        assert len(imported_keypairs) == 1
        assert add_keypair_result.get("key-uid") == imported_keypairs[0].get("key-uid")
        assert add_keypair_result.get("type") == imported_keypairs[0].get("type")
        assert add_keypair_result.get("derived-from") == imported_keypairs[0].get("derived-from")

    def test_add_a_second_keypair_via_pk_with_same_details(self):
        self.account.accounts_service.add_keypair_via_private_key(
            user_1.private_key, self.account.password, keypair_name, wallet_account_details_root
        )

        # different private key but same details
        self.account.accounts_service.add_keypair_via_private_key(
            user_2.private_key, self.account.password, keypair_name, wallet_account_details_root
        )

        keypairs_response = self.account.accounts_service.get_account_keypairs()
        imported_keypairs = [keypair for keypair in keypairs_response if keypair.get("name") == keypair_name]
        assert len(imported_keypairs) == 2, "2 keypairs with the same name should be saved"

    def test_add_duplicate_keypair_via_pk(self):
        resp1 = self.account.accounts_service.add_keypair_via_private_key(
            user_1.private_key, self.account.password, keypair_name, wallet_account_details_root
        )

        # same private key
        with pytest.raises(ApiResponseError, match=re.escape(f'[validation] keypair already added -  keyuid: {resp1.get("key-uid")}')):
            self.account.accounts_service.add_keypair_via_private_key(
                user_1.private_key, self.account.password, keypair_name, wallet_account_details_root
            )

    def test_add_keypair_via_pk_with_wrong_path(self):
        with pytest.raises(
            ApiResponseError,
            match=re.escape("[validation] unsupported profile or seed imported key pair wallet account"),
        ):
            self.account.accounts_service.add_keypair_via_private_key(
                user_1.private_key, self.account.password, keypair_name, wallet_account_details_derivation
            )

    def test_add_keypair_via_pk_with_empty_password(self):
        self.account.accounts_service.add_keypair_via_private_key(user_1.private_key, "", keypair_name, wallet_account_details_root)
