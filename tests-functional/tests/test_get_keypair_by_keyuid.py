import re
import pytest
from resources.constants import user_1, keypair_name, wallet_account_details_derivation
from clients.api import ApiResponseError


@pytest.mark.rpc
class TestGetKeypairByKeyUID:

    @pytest.fixture()
    def backend(self, backend_new_profile):
        return backend_new_profile("account-backend")

    def test_get_keypair_by_existing_keyuid(self, backend):
        get_account_keypairs_resp = backend.accounts_service.get_account_keypairs()
        get_keypair_by_key_uid_resp = backend.accounts_service.get_keypair_by_key_uid(backend.key_uid)
        keypair = get_keypair_by_key_uid_resp
        assert keypair == get_account_keypairs_resp[0]
        assert keypair.get("name") == backend.display_name
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

    def test_get_keypair_by_nonexistent_keyuid(self, backend):
        with pytest.raises(ApiResponseError, match=re.escape("keypair is not found")):
            backend.accounts_service.get_keypair_by_key_uid("0x6d462df35b97fabb8f792eac01240556e26fd2600753e5bbffa4713a9c95abc7")

    def test_get_newly_imported_keypair(self, backend):
        backend.accounts_service.add_keypair_via_seed_phrase(user_1.passphrase, backend.password, keypair_name, wallet_account_details_derivation)
        keypairs_response = backend.accounts_service.get_account_keypairs()
        imported_keypair = [keypair for keypair in keypairs_response if keypair.get("name") == keypair_name][0]
        imported_keypair_key_uid = imported_keypair.get("key-uid")
        resp = backend.accounts_service.get_keypair_by_key_uid(imported_keypair_key_uid)
        assert imported_keypair == resp
