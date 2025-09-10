import pytest


@pytest.mark.accounts
@pytest.mark.rpc
class TestAppGeneral:

    @pytest.fixture(autouse=True)
    def setup_backend(self, backend_new_profile):
        """Initialize one backend for each test function"""
        self.rpc_client = backend_new_profile("rpc_client")

    @pytest.mark.parametrize(
        "method, params",
        [
            ("appgeneral_getCurrencies", []),
        ],
    )
    def test_(self, method, params):
        self.rpc_client.rpc_valid_request(method, params)
        # TODO: Add more assertions on response
