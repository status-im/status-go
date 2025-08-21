import pytest
from resources.constants import keycard_1


@pytest.mark.rpc
class TestDeleteKeycard:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("sender")
        self.keycard = keycard_1
        self.keycard["key-uid"] = self.account.key_uid

    def test_delete_existing_keycard(self):
        self.account.accounts_service.save_or_update_keycard(self.keycard)
        del_resp = self.account.accounts_service.delete_keycard(self.keycard["keycard-uid"])
        assert del_resp["result"] is None
        keycards_after = self.account.accounts_service.get_all_known_keycards()
        assert keycards_after["result"] == []

    def test_delete_nonexistent_keycard(self):
        resp = self.account.accounts_service.delete_keycard("non-existent-keycard-uid-1234")
        assert resp.get("error").get("message") == "keycard: no keycard for the passed keycard uid"
