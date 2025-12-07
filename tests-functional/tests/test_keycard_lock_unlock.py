import copy
import re
import pytest
from resources.constants import keycard_1
from clients.api import ApiResponseError


@pytest.mark.rpc
class TestKeycardLockUnlock:

    @pytest.fixture()
    def backend(self, backend_new_profile):
        return backend_new_profile("account-backend")

    @pytest.fixture()
    def keycard(self, backend):
        """Fresh keycard data bound to the current account's key-uid."""
        kc = copy.deepcopy(keycard_1)
        kc["key-uid"] = backend.key_uid
        return kc

    def test_keycard_lock_unlock_flow(self, backend, keycard):
        backend.accounts_service.save_or_update_keycard(keycard, backend.password)

        # Lock the keycard
        resp_lock = backend.accounts_service.keycard_locked(keycard["keycard-uid"])
        assert resp_lock is None

        # Verify the keycard is locked
        keycards_after_lock = backend.accounts_service.get_all_known_keycards()
        assert keycards_after_lock[0].get("keycard-locked") is True

        # Unlock the keycard
        resp_unlock = backend.accounts_service.keycard_unlocked(keycard["keycard-uid"])
        assert resp_unlock is None

        # Verify the keycard is unlocked
        keycards_after_unlock = backend.accounts_service.get_all_known_keycards()
        assert keycards_after_unlock[0].get("keycard-locked") is False

    def test_lock_nonexistent_keycard(self, backend):
        with pytest.raises(ApiResponseError, match=re.escape("keycard: no keycard for the passed keycard uid")):
            backend.accounts_service.keycard_locked("non-existent-keycard-uid-1234")

    def test_unlock_nonexistent_keycard(self, backend):
        with pytest.raises(ApiResponseError, match=re.escape("keycard: no keycard for the passed keycard uid")):
            backend.accounts_service.keycard_unlocked("non-existent-keycard-uid-1234")
