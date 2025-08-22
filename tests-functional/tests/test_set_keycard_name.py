import copy
import pytest
from resources.constants import keycard_1


@pytest.mark.rpc
class TestSetKeycardName:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("sender")
        self.keycard = copy.deepcopy(keycard_1)
        self.keycard["key-uid"] = self.account.key_uid

    def test_set_keycard_name_success(self):
        self.account.accounts_service.save_or_update_keycard(self.keycard)
        new_name = "UpdatedKeycardName"
        resp = self.account.accounts_service.set_keycard_name(self.keycard["keycard-uid"], new_name)
        assert resp["result"] is None

        keycards_after = self.account.accounts_service.get_all_known_keycards()
        keycards = keycards_after.get("result")
        assert len(keycards) == 1
        assert keycards[0].get("keycard-name") == new_name

    def test_set_name_to_nonexistent_keycard(self):
        resp = self.account.accounts_service.set_keycard_name("non-existent-keycard-uid-1234", "Name")
        assert resp.get("error").get("message") == "keycard: no keycard for the passed keycard uid"
