import copy
import re
import pytest
from resources.constants import keycard_1
from clients.api import ApiResponseError


@pytest.mark.rpc
class TestSaveOrUpdateKeycard:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("sender")
        self.keycard = keycard_1
        self.keycard["key-uid"] = self.account.key_uid

    def test_save_new_keycard(self):
        keycards_before = self.account.accounts_service.get_all_known_keycards()
        assert keycards_before["result"] == []

        resp = self.account.accounts_service.save_or_update_keycard(self.keycard, self.account.password)
        assert resp["result"] is None

        keycards_after = self.account.accounts_service.get_all_known_keycards()
        keycards = keycards_after["result"]
        assert len(keycards) == 1
        assert keycards[0].get("keycard-uid") == self.keycard.get("keycard-uid")
        assert keycards[0].get("keycard-name") == self.keycard.get("keycard-name")
        assert keycards[0].get("keycard-locked") is False
        assert keycards[0].get("accounts-addresses") == self.keycard.get("accounts-addresses")
        assert keycards[0].get("key-uid") == self.keycard.get("key-uid")
        assert keycards[0].get("Position") == 0

    def test_save_duplicate_keycard(self):
        self.account.accounts_service.save_or_update_keycard(self.keycard, self.account.password)
        self.account.accounts_service.save_or_update_keycard(self.keycard, self.account.password)
        keycards_after = self.account.accounts_service.get_all_known_keycards()
        keycards = keycards_after.get("result")
        assert len(keycards) == 1  # only one keycard is saved

    def test_save_multiple_keycards(self):
        self.account.accounts_service.save_or_update_keycard(self.keycard, self.account.password)
        self.second_keycard = copy.deepcopy(self.keycard)
        self.second_keycard["keycard-uid"] = "second_kc_uid"
        self.second_keycard["keycard-name"] = "second_kc_name"
        self.second_keycard["keycard-addresses"] = "0x2f49e5eff87892deb1fffeed666fa75ccb2dbbc2"
        self.account.accounts_service.save_or_update_keycard(self.second_keycard, self.account.password)

        keycards_after = self.account.accounts_service.get_all_known_keycards()
        keycards = keycards_after.get("result")
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

    def test_save_keycard_with_wrong_key_uid(self):
        self.keycard["key-uid"] = "0x91b0a565ccc994d62b4869984fd720c6f99827966f94cb6a9aff02bcbb86a069"
        with pytest.raises(ApiResponseError, match=re.escape("[validation] keycard does not relate to any keypair: keypair is not found")):
            self.account.accounts_service.save_or_update_keycard(self.keycard, self.account.password)

    def test_save_keycard_with_no_account_address(self):
        del self.keycard["accounts-addresses"]
        with pytest.raises(ApiResponseError, match=re.escape("[validation] keycard does not have any accounts")):
            self.account.accounts_service.save_or_update_keycard(self.keycard, self.account.password)

    def test_update_keycards(self):
        self.test_save_multiple_keycards()
        self.keycard["keycard-name"] = "UpdatedFirstKeycardName"
        self.second_keycard["keycard-name"] = "UpdatedSecondKeycardName"
        self.account.accounts_service.save_or_update_keycard(self.keycard, self.account.password)
        self.account.accounts_service.save_or_update_keycard(self.second_keycard, self.account.password)
        keycards_update = self.account.accounts_service.get_all_known_keycards()
        keycards = keycards_update.get("result")
        assert len(keycards) == 2
        assert keycards[0].get("keycard-name") == self.keycard.get("keycard-name")
        assert keycards[1].get("keycard-name") == self.second_keycard.get("keycard-name")
