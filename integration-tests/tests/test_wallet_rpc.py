import random
import pytest
import jsonschema
import json
from conftest import option, user_1, user_2
from test_cases import RpcTestCase, TransactionTestCase


@pytest.mark.wallet
@pytest.mark.tx
@pytest.mark.rpc
class TestTransactionRpc(TransactionTestCase):

    @pytest.mark.parametrize(
        "method, params",
        [
            (
                "wallet_checkRecentHistoryForChainIDs",
                [[31337], ["0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"]],
            ),
            (
                "wallet_getPendingTransactionsForIdentities",
                [[{"chainId": None, "hash": None}]],
            ),
        ],
    )
    def test_tx_(self, method, params):

        _id = str(random.randint(1, 9999))

        if method in ["wallet_getPendingTransactionsForIdentities"]:
            params[0][0]["chainId"] = self.network_id
            params[0][0]["hash"] = self.tx_hash

        response = self.rpc_request(method, params, _id)
        self.verify_is_valid_json_rpc_response(response)
        with open(f"{option.base_dir}/schemas/{method}", "r") as schema:
            jsonschema.validate(instance=response.json(), schema=json.load(schema))

    def test_create_multi_transaction(self):
        response = self.wallet_create_multi_transaction()
        
        # how to create schema:
        # from schema_builder import CustomSchemaBuilder
        # CustomSchemaBuilder(method).create_schema(response.json())
        
        with open(f"{option.base_dir}/schemas/wallet_createMultiTransaction", "r") as schema:
            jsonschema.validate(instance=response.json(), schema=json.load(schema))


@pytest.mark.wallet
@pytest.mark.rpc
class TestRpc(RpcTestCase):

    @pytest.mark.parametrize(
        "method, params",
        [   
            ("wallet_startWallet", []),
            ("wallet_getEthereumChains", []),
            ("wallet_startWallet", []),
            ("wallet_getTokenList", []),
            ("wallet_getCryptoOnRamps", []),
        ],
    )
    def test_(self, method, params):
        _id = str(random.randint(1, 8888))

        response = self.rpc_request(method, params, _id)
        self.verify_is_valid_json_rpc_response(response)
        with open(f"{option.base_dir}/schemas/{method}", "r") as schema:
            jsonschema.validate(instance=response.json(), schema=json.load(schema))
