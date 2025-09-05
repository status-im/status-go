import re
import pytest
from resources.constants import keycard_1
from clients.api import ApiResponseError


@pytest.mark.rpc
class TestKeycardLockUnlock:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("sender")
        self.keycard = keycard_1
        self.keycard["key-uid"] = self.account.key_uid

    def test_keycard_lock_unlock_flow(self):
        self.account.accounts_service.save_or_update_keycard(self.keycard, self.account.password)

        # Lock the keycard
        resp_lock = self.account.accounts_service.keycard_locked(self.keycard["keycard-uid"])
        assert resp_lock["result"] is None

        # Verify the keycard is locked
        keycards_after_lock = self.account.accounts_service.get_all_known_keycards()
        assert keycards_after_lock.get("result")[0].get("keycard-locked") is True

        # Unlock the keycard
        resp_unlock = self.account.accounts_service.keycard_unlocked(self.keycard["keycard-uid"])
        assert resp_unlock["result"] is None

        # Verify the keycard is unlocked
        keycards_after_unlock = self.account.accounts_service.get_all_known_keycards()
        assert keycards_after_unlock.get("result")[0].get("keycard-locked") is False

    def test_lock_nonexistent_keycard(self):
        with pytest.raises(ApiResponseError, match=re.escape("keycard: no keycard for the passed keycard uid")):
            self.account.accounts_service.keycard_locked("non-existent-keycard-uid-1234")

    def test_unlock_nonexistent_keycard(self):
        with pytest.raises(ApiResponseError, match=re.escape("keycard: no keycard for the passed keycard uid")):
            self.account.accounts_service.keycard_unlocked("non-existent-keycard-uid-1234")
