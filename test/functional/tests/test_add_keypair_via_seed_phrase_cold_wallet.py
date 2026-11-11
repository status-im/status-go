import pytest
from resources.constants import COLD_WALLET_TYPE_STATUS_KEYCARD, user_1, wallet_account_details_derivation, keypair_name


@pytest.mark.rpc
class TestAddKeypairViaSeedPhraseColdWallet:

    @pytest.fixture()
    def backend(self, backend_new_profile):
        return backend_new_profile("account-backend")

    def test_add_keypair_via_seed_phrase_as_cold_wallet_writes_no_keystore(self, backend):
        add_resp = backend.accounts_service.add_keypair_via_seed_phrase(
            user_1.passphrase,
            backend.password,
            keypair_name,
            COLD_WALLET_TYPE_STATUS_KEYCARD,
            wallet_account_details_derivation,
        )
        key_uid = add_resp.get("key-uid")
        assert key_uid is not None, "Expected addKeypairViaSeedPhrase to return the created keypair's key-uid"
        addresses = [a.get("address") for a in add_resp.get("accounts", [])] + [add_resp.get("derived-from")]

        kp = backend.accounts_service.get_keypair_by_key_uid(key_uid)
        assert (
            kp.get("cold-wallet") == COLD_WALLET_TYPE_STATUS_KEYCARD
        ), "Expected the coldWallet argument to survive the RPC->messenger->manager pass-through unchanged"
        for address in addresses:
            resp = backend.accounts_service.verify_keystore_file_for_account(address, backend.password)
            assert resp is False, f"Expected no keystore file for {address} because a keycard-created keypair must keep key material off disk"
