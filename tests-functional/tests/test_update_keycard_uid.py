import copy
import re
from time import sleep
import pytest
from resources.constants import keycard_1
from clients.api import ApiResponseError


@pytest.mark.rpc
class TestUpdateKeycardUID:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("sender")
        self.keycard = copy.deepcopy(keycard_1)
        self.keycard["key-uid"] = self.account.key_uid

    def test_update_keycard_uid_success(self):
        # Save keycard first
        self.account.accounts_service.save_or_update_keycard(self.keycard, self.account.password)

        old_uid = self.keycard["keycard-uid"]
        new_uid = "updated-kc-uid-1234"

        sleep(1)

        # Update keycard uid
        resp = self.account.accounts_service.update_keycard_uid(old_uid, new_uid)
        assert resp["result"] is None

        # Verify new uid resolves to the keycard
        keycard_new = self.account.accounts_service.get_keycard_by_keycard_uid(new_uid)
        assert keycard_new.get("result") is not None
        assert keycard_new.get("result").get("keycard-uid") == new_uid

        # Verify old uid no longer exists
        with pytest.raises(ApiResponseError, match=re.escape("keycard: no keycard for the passed keycard uid")):
            self.account.accounts_service.get_keycard_by_keycard_uid(old_uid)

    def test_update_keycard_uid_for_nonexistent_keycard(self):
        with pytest.raises(ApiResponseError, match=re.escape("keycard: no keycard for the passed keycard uid")):
            self.account.accounts_service.update_keycard_uid("non-existent-old-uid", "some-new-uid")
