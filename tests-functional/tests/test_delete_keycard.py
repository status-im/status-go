import copy
import re
import pytest
from resources.constants import keycard_1
from clients.api import ApiResponseError


@pytest.mark.rpc
class TestDeleteKeycard:

    @pytest.fixture()
    def account(self, backend_new_profile):
        return backend_new_profile("account-backend")

    @pytest.fixture()
    def keycard(self, account):
        kc = copy.deepcopy(keycard_1)
        kc["key-uid"] = account.key_uid
        return kc

    def test_delete_existing_keycard(self, account, keycard):
        account.accounts_service.save_or_update_keycard(keycard, account.password)
        del_resp = account.accounts_service.delete_keycard(keycard["keycard-uid"])
        assert del_resp is None
        keycards_after = account.accounts_service.get_all_known_keycards()
        assert keycards_after == []

    def test_delete_nonexistent_keycard(self, account):
        with pytest.raises(ApiResponseError, match=re.escape("keycard: no keycard for the passed keycard uid")):
            account.accounts_service.delete_keycard("non-existent-keycard-uid-1234")

    def test_delete_keycard_accounts(self, account, keycard):
        account.accounts_service.save_or_update_keycard(keycard, account.password)
        keycards_before = account.accounts_service.get_all_known_keycards()
        assert keycards_before[0].get("accounts-addresses") == keycard["accounts-addresses"]
        del_resp = account.accounts_service.delete_keycard_accounts(keycard["keycard-uid"], keycard["accounts-addresses"])
        assert del_resp is None
        keycards_after = account.accounts_service.get_all_known_keycards()
        assert keycards_after[0].get("accounts-addresses") == ["0x0000000000000000000000000000000000000000"]

    def test_delete_all_keycards_with_key_uid(self, account, keycard):
        account.accounts_service.save_or_update_keycard(keycard, account.password)
        second_keycard = copy.deepcopy(keycard)
        second_keycard["keycard-uid"] = "second_kc_uid_delete_all"
        second_keycard["keycard-name"] = "second_kc_name_delete_all"
        account.accounts_service.save_or_update_keycard(second_keycard, account.password)

        keycards_before = account.accounts_service.get_all_known_keycards()
        assert len(keycards_before) == 2

        resp = account.accounts_service.delete_all_keycards_with_key_uid(account.key_uid)
        assert resp is None

        keycards_after = account.accounts_service.get_all_known_keycards()
        assert keycards_after == []
