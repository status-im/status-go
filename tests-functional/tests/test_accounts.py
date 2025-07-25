import random
import pytest
from resources.constants import user_1


@pytest.mark.accounts
@pytest.mark.rpc
class TestAccounts:

    @pytest.fixture(autouse=True)
    def setup_backend(self, backend_factory):
        """Initialize one backend for each test function"""
        self.rpc_client = backend_factory("rpc_client")

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
