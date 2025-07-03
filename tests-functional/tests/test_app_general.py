import random
import pytest


@pytest.mark.accounts
@pytest.mark.rpc
class TestAppGeneral:

    @pytest.fixture(autouse=True)
    def setup_backend(self, backend_factory):
        """Initialize one backend for each test function"""
        self.rpc_client = backend_factory("rpc_client")

    @pytest.mark.parametrize(
        "method, params",
        [
            ("appgeneral_getCurrencies", []),
        ],
    )
    def test_(self, method, params):
        _id = str(random.randint(1, 8888))

        response = self.rpc_client.rpc_valid_request(method, params, _id)
        self.rpc_client.verify_json_schema(response.json(), method)
