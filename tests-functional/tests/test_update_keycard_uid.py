import copy
import re
from time import sleep
import pytest
from resources.constants import keycard_1
from clients.api import ApiResponseError


@pytest.mark.rpc
class TestUpdateKeycardUID:

    @pytest.fixture()
    def account(self, backend_new_profile):
        return backend_new_profile("account-backend")

    @pytest.fixture()
    def keycard(self, account):
        """Fresh keycard data bound to the current account's key-uid."""
        kc = copy.deepcopy(keycard_1)
        kc["key-uid"] = account.key_uid
        return kc

    def test_update_keycard_uid_success(self, account, keycard):
        # Save keycard first
        account.accounts_service.save_or_update_keycard(keycard, account.password)

        old_uid = keycard["keycard-uid"]
        new_uid = "updated-kc-uid-1234"

        sleep(1)

        # Update keycard uid
        resp = account.accounts_service.update_keycard_uid(old_uid, new_uid)
        assert resp is None

        # Verify new uid resolves to the keycard
        keycard_new = account.accounts_service.get_keycard_by_keycard_uid(new_uid)
        assert keycard_new is not None
        assert keycard_new.get("keycard-uid") == new_uid

        # Verify old uid no longer exists
        with pytest.raises(ApiResponseError, match=re.escape("keycard: no keycard for the passed keycard uid")):
            account.accounts_service.get_keycard_by_keycard_uid(old_uid)

    def test_update_keycard_uid_for_nonexistent_keycard(self, account):
        with pytest.raises(ApiResponseError, match=re.escape("keycard: no keycard for the passed keycard uid")):
            account.accounts_service.update_keycard_uid("non-existent-old-uid", "some-new-uid")
