import random
import pytest
import jsonschema
import json
from conftest import option
from test_cases import RpcTestCase


@pytest.mark.accounts
@pytest.mark.rpc
class TestAccounts(RpcTestCase):

    @pytest.mark.parametrize(
        "method, params",
        [
            ("accounts_getKeypairs", []),
            ("accounts_hasPairedDevices", []),
            ("accounts_remainingAccountCapacity", [])

        ],
    )
    def test_(self, method, params):
        _id = str(random.randint(1, 8888))

        response = self.rpc_request(method, params, _id)
        self.verify_is_valid_json_rpc_response(response)
        with open(f"{option.base_dir}/schemas/{method}", "r") as schema:
            jsonschema.validate(instance=response.json(), schema=json.load(schema))
