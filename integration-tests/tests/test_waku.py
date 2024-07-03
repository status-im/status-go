import random
import pytest
import jsonschema
import json
from conftest import option
from test_cases import RpcTestCase


@pytest.mark.waku
class TestWakuRpc(RpcTestCase):

    @pytest.mark.parametrize(
        "method, params",
        [   
            ("wakuext_peers", []),
        ],
    )
    def test_(self, method, params):
        _id = str(random.randint(1, 8888))
        response = self.rpc_request(method, params, _id)

        # how to create schema:
        # from schema_builder import CustomSchemaBuilder
        # CustomSchemaBuilder("wakuext_peers").create_schema(response.json())
        
        self.verify_is_valid_json_rpc_response(response)
        with open(f"{option.base_dir}/schemas/{method}", "r") as schema:
            jsonschema.validate(instance=response.json(), schema=json.load(schema))
