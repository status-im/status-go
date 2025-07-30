import os
import random
import pytest
from resources.constants import user_1


@pytest.mark.accounts
@pytest.mark.rpc
class TestAccounts:

    @pytest.fixture(autouse=True)
    def setup_backend(self, backend_new_profile):
        """Initialize one backend for each test function"""
        self.rpc_client = backend_new_profile("rpc_client")

    @pytest.mark.parametrize(
        "method, params",
        [
            ("accounts_getAccounts", []),
            ("accounts_getKeypairs", []),
            # ("accounts_hasPairedDevices", []), # randomly crashes app, to be reworked/fixed
            # ("accounts_remainingAccountCapacity", []), # randomly crashes app, to be reworked/fixed
            ("multiaccounts_getIdentityImages", [user_1.private_key]),
        ],
    )
    def test_(self, method, params):
        _id = str(random.randint(1, 8888))

        response = self.rpc_client.rpc_valid_request(method, params, _id)
        self.rpc_client.verify_json_schema(response.json(), method)

    def test_store_identity_image(self):
        self.rpc_client.import_data(
            os.path.abspath(os.path.join(os.path.dirname(__file__), "../resources/images/test-image-200x200.jpg")), "/tmp/images/"
        )
        response = self.rpc_client.rpc_valid_request(
            "multiaccounts_storeIdentityImage", [self.rpc_client.key_uid, "/tmp/images/test-image-200x200.jpg", 0, 0, 200, 200]
        )

        jsonResponse = response.json()
        result = jsonResponse.get("result", {})
        assert result is not None, "Identity images were not stored (no result)"
        assert len(result) == 2, "Identity images were not stored (wrong count)"
        assert result[0]["localUrl"] is not None, "Local URL for identity image 1 is not set"
        assert result[1]["localUrl"] is not None, "Local URL for identity image 2 is not set"

        response = self.rpc_client.rpc_valid_request("multiaccounts_getIdentityImages", [self.rpc_client.key_uid])
        jsonResponse = response.json()
        result = jsonResponse.get("result", {})
        assert result is not None, "Identity images were not retrieved (no result)"
        assert len(result) == 2, "Identity images were not retrieved (wrong count)"
        assert result[0]["localUrl"] is not None, "Local URL for identity image 1 is not set"
        assert result[1]["localUrl"] is not None, "Local URL for identity image 2 is not set"
