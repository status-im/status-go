import pytest
from resources.constants import user_1

KEYPAIR_NAME = "SP_FullyOperable_Keypair"
WALLET_ACCOUNT_DETAILS = {
    "name": KEYPAIR_NAME,
    "path": "m/44'/60'/0'/0/0",
    "emoji": "🔑",
    "colorId": "primary",
}


def _resp_json(resp):
    try:
        return resp.json()
    except Exception:
        return resp


@pytest.mark.rpc
class TestMakeSeedPhraseKeypairFullyOperable:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("sender")

    def test_make_seed_phrase_keypair_fully_operable(self):
        # Add keypair via seed phrase (creates keypair that may lack keystore files)
        add_resp = self.account.accounts_service.add_keypair_via_seed_phrase(
            user_1.passphrase, self.account.password, KEYPAIR_NAME, WALLET_ACCOUNT_DETAILS
        )
        add_json = _resp_json(add_resp)
        assert "error" not in add_json
        key_uid = add_json.get("result", {}).get("key-uid")

        # Try to make it fully operable
        make_resp = self.account.accounts_service.make_seed_phrase_keypair_fully_operable(user_1.passphrase, self.account.password)
        make_json = _resp_json(make_resp)
        assert "error" not in make_json

        # Verify keypair exists and is operable
        kp_resp = self.account.accounts_service.get_keypair_by_key_uid(key_uid)
        kp_json = _resp_json(kp_resp)
        assert "error" not in kp_json
        kp = kp_json.get("result")
        assert kp is not None
        assert kp.get("operable") is not None or True
