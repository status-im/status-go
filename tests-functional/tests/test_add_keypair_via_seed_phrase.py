import re
import pytest
from resources.constants import user_1, user_2, wallet_account_details_derivation, wallet_account_details_root, keypair_name
from clients.api import ApiResponseError


@pytest.mark.rpc
class TestAddKeypairViaSeedPhrase:

    @pytest.fixture()
    def account(self, backend_new_profile):
        return backend_new_profile("account-backend")

    def test_add_valid_keypair_via_seed_phrase(self, account):
        add_keypair_response = account.accounts_service.add_keypair_via_seed_phrase(
            user_1.passphrase, account.password, keypair_name, wallet_account_details_derivation
        )
        add_keypair_result = add_keypair_response
        accounts = add_keypair_result.get("accounts")
        assert len(accounts) == 1
        new_keypair = accounts[0]
        assert new_keypair.get("address").lower() == user_1.address.lower()
        assert new_keypair.get("chat") is False
        assert new_keypair.get("clock") == 0
        assert new_keypair.get("colorId") == wallet_account_details_derivation.get("colorId")
        assert new_keypair.get("createdAt") == 0
        assert new_keypair.get("emoji") == wallet_account_details_derivation.get("emoji")
        assert new_keypair.get("hidden") is False
        assert new_keypair.get("name") == keypair_name
        assert new_keypair.get("operable") == "fully"
        assert new_keypair.get("path") == wallet_account_details_derivation.get("path")
        assert new_keypair.get("position") == 1
        assert new_keypair.get("removed") is False
        assert new_keypair.get("type") == ""
        assert add_keypair_result.get("type") == "seed"
        assert new_keypair.get("wallet") is False

        # Fetch keypairs and ensure the imported one is present
        get_keypairs_response = account.accounts_service.get_account_keypairs()
        imported_keypairs = [keypair for keypair in get_keypairs_response if keypair.get("name") == keypair_name]
        assert len(imported_keypairs) == 1
        assert add_keypair_result.get("key-uid") == imported_keypairs[0].get("key-uid")
        assert add_keypair_result.get("type") == imported_keypairs[0].get("type")
        assert add_keypair_result.get("derived-from") == imported_keypairs[0].get("derived-from")

    def test_add_a_second_keypair_via_sp_with_same_details(self, account):
        account.accounts_service.add_keypair_via_seed_phrase(user_1.passphrase, account.password, keypair_name, wallet_account_details_derivation)

        # different private key but same details
        account.accounts_service.add_keypair_via_seed_phrase(user_2.private_key, account.password, keypair_name, wallet_account_details_derivation)

        keypairs_response = account.accounts_service.get_account_keypairs()
        imported_keypairs = [keypair for keypair in keypairs_response if keypair.get("name") == keypair_name]
        assert len(imported_keypairs) == 2, "2 keypairs with the same name should be saved"

    def test_add_duplicate_keypair_via_sp(self, account):
        resp1 = account.accounts_service.add_keypair_via_seed_phrase(
            user_1.passphrase, account.password, keypair_name, wallet_account_details_derivation
        )

        # same private key
        with pytest.raises(ApiResponseError, match=re.escape(f'[validation] keypair already added -  keyuid: {resp1.get("key-uid")}')):
            account.accounts_service.add_keypair_via_seed_phrase(user_1.passphrase, account.password, keypair_name, wallet_account_details_derivation)

    def test_add_keypair_via_sp_with_wrong_path(self, account):
        with pytest.raises(
            ApiResponseError,
            match=re.escape("[validation] unsupported profile or seed imported key pair wallet account"),
        ):
            account.accounts_service.add_keypair_via_seed_phrase(user_1.passphrase, account.password, keypair_name, wallet_account_details_root)

    def test_add_keypair_via_sp_with_empty_password(self, account):
        account.accounts_service.add_keypair_via_seed_phrase(user_1.passphrase, "", keypair_name, wallet_account_details_derivation)
