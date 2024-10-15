import json
import random

import jsonschema
import pytest

from conftest import option
from test_cases import StatusBackendTestCase, TransactionTestCase


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

        response = self.rpc_client.rpc_valid_request(method, params, _id)
        self.rpc_client.verify_json_schema(response.json(), method)

    def test_create_multi_transaction(self):
        response = self.wallet_create_multi_transaction()
        self.rpc_client.verify_is_valid_json_rpc_response(response)

        # how to create schema:
        # from schema_builder import CustomSchemaBuilder
        # CustomSchemaBuilder(method).create_schema(response.json())

        with open(f"{option.base_dir}/schemas/wallet_createMultiTransaction/transferTx_positive", "r") as schema:
            jsonschema.validate(instance=response.json(), schema=json.load(schema))

    @pytest.mark.parametrize(
        "method, changed_values, expected_error_code, expected_error_text",
        [
            (
                    "transferTx_value_not_enough_balance",
                    {'value': '0x21e438ea8139cd35004'}, -32000, "Insufficient funds for gas",
            ),
            (
                    "transferTx_from_from_invalid_string",
                    {'from': 'some_invalid_address'}, -32602, "cannot unmarshal hex string without 0x prefix",
            ),
        ],
    )
    def test_create_multi_transaction_validation(self, method,
                                                 changed_values,
                                                 expected_error_code, expected_error_text):
        response = self.wallet_create_multi_transaction(**changed_values)
        self.rpc_client.verify_is_json_rpc_error(response)
        actual_error_code, actual_error_text = response.json()['error']['code'], response.json()['error']['message']
        assert expected_error_code == actual_error_code, \
            f"got code: {actual_error_code} instead of expected: {expected_error_code}"
        assert expected_error_text in actual_error_text, \
            f"got error: {actual_error_text} that does not include: {expected_error_text}"

        self.rpc_client.verify_json_schema(response.json(), "wallet_createMultiTransaction/transferTx_error")


@pytest.mark.wallet
@pytest.mark.rpc
class TestRpc(StatusBackendTestCase):

    @pytest.mark.parametrize(
        "method, params",
        [
            ("wallet_startWallet", []),
            ("wallet_getEthereumChains", []),
            ("wallet_getTokenList", []),
            ("wallet_getCryptoOnRamps", []),
            ("wallet_getCachedCurrencyFormats", [])
        ],
    )
    def test_(self, method, params):
        _id = str(random.randint(1, 8888))

        response = self.rpc_client.rpc_valid_request(method, params, _id)
        self.rpc_client.verify_json_schema(response.json(), method)
