import copy
import re
import pytest
from resources.constants import keycard_1
from clients.api import ApiResponseError


@pytest.mark.rpc
class TestDeleteKeycard:

    @pytest.fixture()
    def backend(self, backend_new_profile):
        return backend_new_profile("account-backend")

    @pytest.fixture()
    def keycard(self, backend):
        kc = copy.deepcopy(keycard_1)
        kc["key-uid"] = backend.key_uid
        return kc

    def test_delete_existing_keycard(self, backend, keycard):
        backend.accounts_service.save_or_update_keycard(keycard, backend.password)
        del_resp = backend.accounts_service.delete_keycard(keycard["keycard-uid"])
        assert del_resp is None
        keycards_after = backend.accounts_service.get_all_known_keycards()
        assert keycards_after == []

    def test_delete_nonexistent_keycard(self, backend):
        with pytest.raises(ApiResponseError, match=re.escape("keycard: no keycard for the passed keycard uid")):
            backend.accounts_service.delete_keycard("non-existent-keycard-uid-1234")

    def test_delete_keycard_accounts(self, backend, keycard):
        backend.accounts_service.save_or_update_keycard(keycard, backend.password)
        keycards_before = backend.accounts_service.get_all_known_keycards()
        assert keycards_before[0].get("accounts-addresses") == keycard["accounts-addresses"]
        del_resp = backend.accounts_service.delete_keycard_accounts(keycard["keycard-uid"], keycard["accounts-addresses"])
        assert del_resp is None
        keycards_after = backend.accounts_service.get_all_known_keycards()
        assert keycards_after[0].get("accounts-addresses") == ["0x0000000000000000000000000000000000000000"]

    def test_delete_all_keycards_with_key_uid(self, backend, keycard):
        backend.accounts_service.save_or_update_keycard(keycard, backend.password)
        second_keycard = copy.deepcopy(keycard)
        second_keycard["keycard-uid"] = "second_kc_uid_delete_all"
        second_keycard["keycard-name"] = "second_kc_name_delete_all"
        backend.accounts_service.save_or_update_keycard(second_keycard, backend.password)

        keycards_before = backend.accounts_service.get_all_known_keycards()
        assert len(keycards_before) == 2

        resp = backend.accounts_service.delete_all_keycards_with_key_uid(backend.key_uid)
        assert resp is None

        keycards_after = backend.accounts_service.get_all_known_keycards()
        assert keycards_after == []
