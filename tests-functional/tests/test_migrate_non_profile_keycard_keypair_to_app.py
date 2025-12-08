import copy
import pytest
from resources.constants import keycard_1, user_1, wallet_account_details_derivation, keypair_name


@pytest.mark.rpc
class TestMigrateNonProfileKeycardKeypairToApp:

    @pytest.fixture()
    def backend(self, backend_new_profile):
        return backend_new_profile("account-backend")

    def test_full_migrate_flow(self, backend):
        # 1) Add a seed-derived keypair (simulates a keypair that will later be converted to a keycard)
        add_resp = backend.accounts_service.add_keypair_via_seed_phrase(
            user_1.passphrase, backend.password, keypair_name, wallet_account_details_derivation
        )
        assert "error" not in add_resp
        add_result = add_resp
        assert add_result is not None
        key_uid = add_result.get("key-uid")
        assert key_uid is not None

        # 2) Create and save a keycard that references the keypair (so the keypair has a keycard)
        kc = copy.deepcopy(keycard_1)
        kc["key-uid"] = key_uid
        kc["keycard-uid"] = "kc-migrate-non-profile-1"
        # make the accounts-addresses reflect the accounts returned by add_keypair
        accounts_list = add_result.get("accounts", [])
        assert len(accounts_list) > 0
        kc["accounts-addresses"] = [a.get("address") for a in accounts_list]

        addresses = [kc["accounts-addresses"][0], add_result.get("derived-from")]

        for address in addresses:
            resp = backend.accounts_service.verify_keystore_file_for_account(address, backend.password)
            assert resp is True

        backend.accounts_service.save_or_update_keycard(kc, backend.password)

        for address in addresses:
            resp = backend.accounts_service.verify_keystore_file_for_account(address, backend.password)
            assert resp is False

        # verify keycard present
        keycards_before = backend.accounts_service.get_all_known_keycards()
        kcs = keycards_before
        matching = [x for x in kcs if x.get("keycard-uid") == kc["keycard-uid"]]
        assert len(matching) == 1

        kp_resp = backend.accounts_service.get_keypair_by_key_uid(key_uid)

        # 3) Migrate the non-profile keycard keypair to app (full flow)
        migrate_resp = backend.accounts_service.migrate_non_profile_keycard_keypair_to_app(user_1.passphrase, backend.password)
        # many RPC "void" actions return result None
        assert migrate_resp is None

        for address in addresses:
            resp = backend.accounts_service.verify_keystore_file_for_account(address, backend.password)
            assert resp is True

        # 4) Verify the keypair still exists and that its keycards list is empty (i.e. migrated off keycard)
        kp_resp = backend.accounts_service.get_keypair_by_key_uid(key_uid)
        assert kp_resp is not None
        # after migration the keypair should no longer have keycards linked
        assert isinstance(kp_resp.get("keycards"), list)
        assert len(kp_resp.get("keycards")) == 0

        # accounts should still be present on the keypair
        assert isinstance(kp_resp.get("accounts"), list)
        assert len(kp_resp.get("accounts")) >= 1
