import copy
import re
import pytest
from resources.constants import keycard_1
from clients.api import ApiResponseError


@pytest.mark.rpc
class TestGetKeycards:

    # getAllKnownKeycards is covered inside TestSaveOrUpdateKeycard suite

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("sender")
        self.keycard = copy.deepcopy(keycard_1)
        self.keycard["key-uid"] = self.account.key_uid
        self.account.accounts_service.save_or_update_keycard(self.keycard, self.account.password)
        self.second_keycard = copy.deepcopy(self.keycard)
        self.second_keycard["keycard-uid"] = "second_kc_uid"
        self.second_keycard["keycard-name"] = "second_kc_name"
        self.second_keycard["keycard-addresses"] = "0x2f49e5eff87892deb1fffeed666fa75ccb2dbbc2"
        self.account.accounts_service.save_or_update_keycard(self.second_keycard, self.account.password)

    def test_get_keycard_by_keycard_uid(self):
        first_keycard = self.account.accounts_service.get_keycard_by_keycard_uid(self.keycard.get("keycard-uid"))
        assert first_keycard.get("keycard-name") == self.keycard.get("keycard-name")
        second_keycard = self.account.accounts_service.get_keycard_by_keycard_uid(self.second_keycard.get("keycard-uid"))
        assert second_keycard.get("keycard-name") == self.second_keycard.get("keycard-name")

    def test_get_keycard_by_nonexistent_keycard_uid(self):
        with pytest.raises(ApiResponseError, match=re.escape("keycard: no keycard for the passed keycard uid")):
            self.account.accounts_service.get_keycard_by_keycard_uid("0x0000000000000000000000000000000000000000000000000000000000000000")

    def test_get_keycards_with_same_keyuid(self):
        resp = self.account.accounts_service.get_keycards_with_same_key_uid(self.account.key_uid)
        keycards = resp
        assert len(keycards) == 2
        assert keycards[0].get("keycard-uid") == self.keycard.get("keycard-uid")
        assert keycards[0].get("keycard-name") == self.keycard.get("keycard-name")
        assert keycards[0].get("keycard-locked") is False
        assert keycards[0].get("accounts-addresses") == self.keycard.get("accounts-addresses")
        assert keycards[0].get("key-uid") == self.keycard.get("key-uid")
        assert keycards[0].get("Position") == 0
        assert keycards[1].get("keycard-uid") == self.second_keycard.get("keycard-uid")
        assert keycards[1].get("keycard-name") == self.second_keycard.get("keycard-name")
        assert keycards[1].get("keycard-locked") is False
        assert keycards[1].get("accounts-addresses") == self.second_keycard.get("accounts-addresses")
        assert keycards[1].get("key-uid") == self.second_keycard.get("key-uid")
        assert keycards[1].get("Position") == 1

    def test_get_keycards_with_same_nonexistent_keyuid(self):
        resp = self.account.accounts_service.get_keycards_with_same_key_uid("0x0000000000000000000000000000000000000000000000000000000000000000")
        assert resp == []
