import copy
import re
import pytest
from resources.constants import keycard_1
from clients.api import ApiResponseError


@pytest.mark.rpc
class TestGetKeycards:

    # getAllKnownKeycards is covered inside TestSaveOrUpdateKeycard suite

    @pytest.fixture()
    def backend(self, backend_new_profile):
        return backend_new_profile("account-backend")

    @pytest.fixture()
    def keycards(self, backend):
        """Prepare two keycards bound to the current account's key-uid and persist them."""
        first = copy.deepcopy(keycard_1)
        first["key-uid"] = backend.key_uid
        backend.accounts_service.save_or_update_keycard(first, backend.password)

        second = copy.deepcopy(first)
        second["keycard-uid"] = "second_kc_uid"
        second["keycard-name"] = "second_kc_name"
        second["keycard-addresses"] = "0x2f49e5eff87892deb1fffeed666fa75ccb2dbbc2"
        backend.accounts_service.save_or_update_keycard(second, backend.password)

        return first, second

    def test_get_keycard_by_keycard_uid(self, backend, keycards):
        first_keycard_data, second_keycard_data = keycards
        first_keycard = backend.accounts_service.get_keycard_by_keycard_uid(first_keycard_data.get("keycard-uid"))
        assert first_keycard.get("keycard-name") == first_keycard_data.get("keycard-name")
        second_keycard = backend.accounts_service.get_keycard_by_keycard_uid(second_keycard_data.get("keycard-uid"))
        assert second_keycard.get("keycard-name") == second_keycard_data.get("keycard-name")

    def test_get_keycard_by_nonexistent_keycard_uid(self, backend):
        with pytest.raises(ApiResponseError, match=re.escape("keycard: no keycard for the passed keycard uid")):
            backend.accounts_service.get_keycard_by_keycard_uid("0x0000000000000000000000000000000000000000000000000000000000000000")

    def test_get_keycards_with_same_keyuid(self, backend, keycards):
        first_keycard_data, second_keycard_data = keycards
        resp = backend.accounts_service.get_keycards_with_same_key_uid(backend.key_uid)
        keycards_list = resp
        assert len(keycards_list) == 2
        assert keycards_list[0].get("keycard-uid") == first_keycard_data.get("keycard-uid")
        assert keycards_list[0].get("keycard-name") == first_keycard_data.get("keycard-name")
        assert keycards_list[0].get("keycard-locked") is False
        assert keycards_list[0].get("accounts-addresses") == first_keycard_data.get("accounts-addresses")
        assert keycards_list[0].get("key-uid") == first_keycard_data.get("key-uid")
        assert keycards_list[0].get("Position") == 0
        assert keycards_list[1].get("keycard-uid") == second_keycard_data.get("keycard-uid")
        assert keycards_list[1].get("keycard-name") == second_keycard_data.get("keycard-name")
        assert keycards_list[1].get("keycard-locked") is False
        assert keycards_list[1].get("accounts-addresses") == second_keycard_data.get("accounts-addresses")
        assert keycards_list[1].get("key-uid") == second_keycard_data.get("key-uid")
        assert keycards_list[1].get("Position") == 1

    def test_get_keycards_with_same_nonexistent_keyuid(self, backend):
        resp = backend.accounts_service.get_keycards_with_same_key_uid("0x0000000000000000000000000000000000000000000000000000000000000000")
        assert resp == []
