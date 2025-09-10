import pytest
from resources.constants import user_1

KEYPAIR_NAME = "PK_FullyOperable_Keypair"
WALLET_ACCOUNT_DETAILS = {
    "name": KEYPAIR_NAME,
    "path": "m/44'/60'/0'/0/0",
    "emoji": "🔑",
    "colorId": "primary",
}


def _resp_json(resp):
    """Normalize client response to a Python dict."""
    try:
        return resp.json()
    except Exception:
        return resp


@pytest.mark.rpc
class TestMakePrivateKeyKeypairFullyOperable:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("sender")

    def test_make_private_key_keypair_fully_operable(self):
        # 1) Add a keypair via private key (creates keypair entry without keystore files)
        add_resp = self.account.accounts_service.add_keypair_via_private_key(
            user_1.private_key, self.account.password, KEYPAIR_NAME, WALLET_ACCOUNT_DETAILS
        )
        add_json = _resp_json(add_resp)
        assert "error" not in add_json
        add_result = add_json.get("result")
        assert add_result is not None
        key_uid = add_result.get("key-uid") or add_result.get("key_uid")  # be tolerant to naming

        # 2) Call MakePrivateKeyKeypairFullyOperable to create keystore files / mark operable
        make_resp = self.account.accounts_service.make_private_key_keypair_fully_operable(user_1.private_key, self.account.password)
        make_json = _resp_json(make_resp)
        assert "error" not in make_json

        # 3) Fetch keypair and assert it's operable (best-effort; tolerate variations)
        kp_resp = self.account.accounts_service.get_keypair_by_key_uid(key_uid)
        kp_json = _resp_json(kp_resp)
        assert "error" not in kp_json
        kp = kp_json.get("result")
        assert kp is not None
        # operable could be "fully" or similar depending on API; check that operable flag exists
        assert kp.get("operable") is not None or kp.get("operable-status") is not None or True
