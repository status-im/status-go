import random
import pytest


class TestRpc:

    @pytest.fixture(autouse=True)
    def setup_backend(self, backend_new_profile):
        """Initialize one backend for each test function"""
        self.rpc_client = backend_new_profile("rpc_client")

    @pytest.mark.parametrize(
        "method, params",
        [
            ("wakuext_peers", []),
        ],
    )
    def test_valid_rpc_requests(self, method, params):
        _id = str(random.randint(1, 8888))
        self.rpc_client.rpc_valid_request(method, params, _id, schema_check=True)
