import copy
import re
import pytest
from resources.constants import keycard_1
from clients.api import ApiResponseError


@pytest.mark.rpc
class TestSaveOrUpdateKeycard:

    @pytest.fixture()
    def account(self, backend_new_profile):
        return backend_new_profile("account-backend")

    @pytest.fixture()
    def keycard(self, account):
        """Fresh keycard data bound to the current account's key-uid."""
        kc = copy.deepcopy(keycard_1)
        kc["key-uid"] = account.key_uid
        return kc

    def test_save_new_keycard(self, account, keycard):
        keycards_before = account.accounts_service.get_all_known_keycards()
        assert keycards_before == []

        resp = account.accounts_service.save_or_update_keycard(keycard, account.password)
        assert resp is None

        keycards_after = account.accounts_service.get_all_known_keycards()
        keycards = keycards_after
        assert len(keycards) == 1
        assert keycards[0].get("keycard-uid") == keycard.get("keycard-uid")
        assert keycards[0].get("keycard-name") == keycard.get("keycard-name")
        assert keycards[0].get("keycard-locked") is False
        assert keycards[0].get("accounts-addresses") == keycard.get("accounts-addresses")
        assert keycards[0].get("key-uid") == keycard.get("key-uid")
        assert keycards[0].get("Position") == 0

    def test_save_duplicate_keycard(self, account, keycard):
        account.accounts_service.save_or_update_keycard(keycard, account.password)
        account.accounts_service.save_or_update_keycard(keycard, account.password)
        keycards_after = account.accounts_service.get_all_known_keycards()
        keycards = keycards_after
        assert len(keycards) == 1  # only one keycard is saved

    def test_save_multiple_keycards(self, account, keycard):
        account.accounts_service.save_or_update_keycard(keycard, account.password)
        second_keycard = copy.deepcopy(keycard)
        second_keycard["keycard-uid"] = "second_kc_uid"
        second_keycard["keycard-name"] = "second_kc_name"
        second_keycard["keycard-addresses"] = "0x2f49e5eff87892deb1fffeed666fa75ccb2dbbc2"
        account.accounts_service.save_or_update_keycard(second_keycard, account.password)

        keycards_after = account.accounts_service.get_all_known_keycards()
        keycards = keycards_after
        assert len(keycards) == 2
        assert keycards[0].get("keycard-uid") == keycard.get("keycard-uid")
        assert keycards[0].get("keycard-name") == keycard.get("keycard-name")
        assert keycards[0].get("keycard-locked") is False
        assert keycards[0].get("accounts-addresses") == keycard.get("accounts-addresses")
        assert keycards[0].get("key-uid") == keycard.get("key-uid")
        assert keycards[0].get("Position") == 0
        assert keycards[1].get("keycard-uid") == second_keycard.get("keycard-uid")
        assert keycards[1].get("keycard-name") == second_keycard.get("keycard-name")
        assert keycards[1].get("keycard-locked") is False
        assert keycards[1].get("accounts-addresses") == second_keycard.get("accounts-addresses")
        assert keycards[1].get("key-uid") == second_keycard.get("key-uid")
        assert keycards[1].get("Position") == 1

    def test_save_keycard_with_wrong_key_uid(self, account, keycard):
        keycard["key-uid"] = "0x91b0a565ccc994d62b4869984fd720c6f99827966f94cb6a9aff02bcbb86a069"
        with pytest.raises(ApiResponseError, match=re.escape("[validation] keycard does not relate to any keypair: keypair is not found")):
            account.accounts_service.save_or_update_keycard(keycard, account.password)

    def test_save_keycard_with_no_account_address(self, account, keycard):
        del keycard["accounts-addresses"]
        with pytest.raises(ApiResponseError, match=re.escape("[validation] keycard does not have any accounts")):
            account.accounts_service.save_or_update_keycard(keycard, account.password)

    def test_update_keycards(self, account, keycard):
        # First create two keycards
        account.accounts_service.save_or_update_keycard(keycard, account.password)
        second_keycard = copy.deepcopy(keycard)
        second_keycard["keycard-uid"] = "second_kc_uid"
        second_keycard["keycard-name"] = "second_kc_name"
        second_keycard["keycard-addresses"] = "0x2f49e5eff87892deb1fffeed666fa75ccb2dbbc2"
        account.accounts_service.save_or_update_keycard(second_keycard, account.password)

        # Now update their names
        keycard["keycard-name"] = "UpdatedFirstKeycardName"
        second_keycard["keycard-name"] = "UpdatedSecondKeycardName"
        account.accounts_service.save_or_update_keycard(keycard, account.password)
        account.accounts_service.save_or_update_keycard(second_keycard, account.password)

        keycards_update = account.accounts_service.get_all_known_keycards()
        keycards = keycards_update
        assert len(keycards) == 2
        assert keycards[0].get("keycard-name") == keycard.get("keycard-name")
        assert keycards[1].get("keycard-name") == second_keycard.get("keycard-name")
