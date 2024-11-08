import random

import pytest

from test_cases import StatusBackendTestCase


@pytest.mark.accounts
@pytest.mark.rpc
class TestAppGeneral(StatusBackendTestCase):

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
