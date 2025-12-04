import copy
import re
import pytest
from resources.constants import keycard_1
from clients.api import ApiResponseError


@pytest.mark.rpc
class TestKeycardLockUnlock:

    @pytest.fixture()
    def account(self, backend_new_profile):
        return backend_new_profile("account-backend")

    @pytest.fixture()
    def keycard(self, account):
        """Fresh keycard data bound to the current account's key-uid."""
        kc = copy.deepcopy(keycard_1)
        kc["key-uid"] = account.key_uid
        return kc

    def test_keycard_lock_unlock_flow(self, account, keycard):
        account.accounts_service.save_or_update_keycard(keycard, account.password)

        # Lock the keycard
        resp_lock = account.accounts_service.keycard_locked(keycard["keycard-uid"])
        assert resp_lock is None

        # Verify the keycard is locked
        keycards_after_lock = account.accounts_service.get_all_known_keycards()
        assert keycards_after_lock[0].get("keycard-locked") is True

        # Unlock the keycard
        resp_unlock = account.accounts_service.keycard_unlocked(keycard["keycard-uid"])
        assert resp_unlock is None

        # Verify the keycard is unlocked
        keycards_after_unlock = account.accounts_service.get_all_known_keycards()
        assert keycards_after_unlock[0].get("keycard-locked") is False

    def test_lock_nonexistent_keycard(self, account):
        with pytest.raises(ApiResponseError, match=re.escape("keycard: no keycard for the passed keycard uid")):
            account.accounts_service.keycard_locked("non-existent-keycard-uid-1234")

    def test_unlock_nonexistent_keycard(self, account):
        with pytest.raises(ApiResponseError, match=re.escape("keycard: no keycard for the passed keycard uid")):
            account.accounts_service.keycard_unlocked("non-existent-keycard-uid-1234")
