import pytest
from resources.constants import (
    user_1,
    wallet_account_details_derivation,
    keypair_name,
)


@pytest.mark.rpc
class TestMigrateNonProfileColdWalletKeypairToApp:

    @pytest.fixture()
    def backend(self, backend_new_profile):
        return backend_new_profile("account-backend")

    def test_full_migrate_flow(self, backend):
        # 1) Add a seed-derived keypair (regular, not cold-wallet)
        add_resp = backend.accounts_service.add_keypair_via_seed_phrase(
            user_1.passphrase,
            backend.password,
            keypair_name,
            "",
            wallet_account_details_derivation,
        )
        assert "error" not in add_resp
        add_result = add_resp
        assert add_result is not None
        key_uid = add_result.get("key-uid")
        assert key_uid is not None

        accounts_list = add_result.get("accounts", [])
        assert len(accounts_list) > 0
        addresses = [a.get("address") for a in accounts_list] + [add_result.get("derived-from")]

        # Keystore files for the keypair accounts should exist
        for address in addresses:
            resp = backend.accounts_service.verify_keystore_file_for_account(address, backend.password)
            assert resp is True

        # 2) Migrate the keypair to a cold wallet (status-keycard)
        # This should remove the keystore files
        backend.accounts_service.migrate_non_profile_keypair_to_cold_wallet(key_uid, backend.password, "status-keycard")

        for address in addresses:
            resp = backend.accounts_service.verify_keystore_file_for_account(address, backend.password)
            assert resp is False

        # Verify the keypair is now flagged as cold-wallet backed
        kp_resp = backend.accounts_service.get_keypair_by_key_uid(key_uid)
        assert kp_resp is not None
        assert kp_resp.get("cold-wallet") == "status-keycard"

        # 3) Migrate the non-profile cold-wallet keypair back to the app (full flow)
        # Keystore files should be restored and the cold-wallet flag cleared
        migrate_resp = backend.accounts_service.migrate_non_profile_cold_wallet_keypair_to_app(user_1.passphrase, backend.password)
        # many RPC "void" actions return result None
        assert migrate_resp is None

        for address in addresses:
            resp = backend.accounts_service.verify_keystore_file_for_account(address, backend.password)
            assert resp is True

        # 4) Verify the keypair still exists, cold-wallet flag is cleared, and accounts remain
        kp_resp = backend.accounts_service.get_keypair_by_key_uid(key_uid)
        assert kp_resp is not None
        assert kp_resp.get("cold-wallet", "") == ""
        assert isinstance(kp_resp.get("accounts"), list)
        assert len(kp_resp.get("accounts")) >= 1
