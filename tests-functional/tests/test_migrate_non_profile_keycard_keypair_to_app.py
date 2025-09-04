import copy
import pytest
from resources.constants import keycard_1, user_1

KEYPAIR_NAME = "SeedPhraseImportedKeypairForMigrate"
WALLET_ACCOUNT_DETAILS = {
    "name": KEYPAIR_NAME,
    "path": "m/44'/60'/0'/0/0",
    "emoji": "🔑",
    "colorId": "primary",
}


@pytest.mark.rpc
class TestMigrateNonProfileKeycardKeypairToApp:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("sender")

    def test_full_migrate_flow(self):
        # 1) Add a seed-derived keypair (simulates a keypair that will later be converted to a keycard)
        add_resp = self.account.accounts_service.add_keypair_via_seed_phrase(
            user_1.passphrase, self.account.password, KEYPAIR_NAME, WALLET_ACCOUNT_DETAILS
        )
        assert "error" not in add_resp
        add_result = add_resp.get("result")
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
            resp = self.account.accounts_service.verify_keystore_file_for_account(address, self.account.password)
            assert resp.get("result") is True

        save_resp = self.account.accounts_service.save_or_update_keycard(kc)
        assert "error" not in save_resp

        for address in addresses:
            resp = self.account.accounts_service.verify_keystore_file_for_account(address, self.account.password)
            assert resp.get("result") is True

        # verify keycard present
        keycards_before = self.account.accounts_service.get_all_known_keycards()
        kcs = keycards_before.get("result", [])
        matching = [x for x in kcs if x.get("keycard-uid") == kc["keycard-uid"]]
        assert len(matching) == 1

        kp_resp = self.account.accounts_service.get_keypair_by_key_uid(key_uid)

        # 3) Migrate the non-profile keycard keypair to app (full flow)
        migrate_resp = self.account.accounts_service.migrate_non_profile_keycard_keypair_to_app(user_1.passphrase, self.account.password)
        # RPC returns a JSON object; migration should not return an error
        assert "error" not in migrate_resp
        # many RPC "void" actions return result None
        assert migrate_resp.get("result") is None

        for address in addresses:
            resp = self.account.accounts_service.verify_keystore_file_for_account(address, self.account.password)
            assert resp.get("result") is True

        # 4) Verify the keypair still exists and that its keycards list is empty (i.e. migrated off keycard)
        kp_resp = self.account.accounts_service.get_keypair_by_key_uid(key_uid)
        assert "error" not in kp_resp
        kp = kp_resp.get("result")
        assert kp is not None
        # after migration the keypair should no longer have keycards linked
        assert isinstance(kp.get("keycards"), list)
        assert len(kp.get("keycards")) == 0

        # accounts should still be present on the keypair
        assert isinstance(kp.get("accounts"), list)
        assert len(kp.get("accounts")) >= 1
