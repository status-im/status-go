import copy
import re
import pytest
from resources.constants import keycard_1
from clients.api import ApiResponseError


@pytest.mark.rpc
class TestSetKeycardName:

    @pytest.fixture()
    def backend(self, backend_new_profile):
        return backend_new_profile("account-backend")

    @pytest.fixture()
    def keycard(self, backend):
        """Fresh keycard data bound to the current account's key-uid."""
        kc = copy.deepcopy(keycard_1)
        kc["key-uid"] = backend.key_uid
        return kc

    def test_set_keycard_name_success(self, backend, keycard):
        backend.accounts_service.save_or_update_keycard(keycard, backend.password)
        new_name = "UpdatedKeycardName"
        resp = backend.accounts_service.set_keycard_name(keycard["keycard-uid"], new_name)
        assert resp is None

        keycards_after = backend.accounts_service.get_all_known_keycards()
        keycards = keycards_after
        assert len(keycards) == 1
        assert keycards[0].get("keycard-name") == new_name

    def test_set_name_to_nonexistent_keycard(self, backend):
        with pytest.raises(ApiResponseError, match=re.escape("keycard: no keycard for the passed keycard uid")):
            backend.accounts_service.set_keycard_name("non-existent-keycard-uid-1234", "Name")
