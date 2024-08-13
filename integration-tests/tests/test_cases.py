import json
import jsonschema
import requests
from conftest import option, user_1, user_2


class RpcTestCase:

    def setup_method(self):
        self.network_id = 31337

    def verify_is_valid_json_rpc_response(self, response, _id=None):
        assert response.status_code == 200
        assert response.content

        try:
            response.json()["result"]
        except json.JSONDecodeError:
            raise AssertionError(f"invalid JSON in {response.content}")
        except KeyError:
            raise AssertionError(f"no 'result' in {response.json()}")
        if _id:
            try:
                if _id != response.json()["id"]:
                    raise AssertionError(
                        f"got id: {response.json()['id']} instead of expected id: {_id}"
                    )
            except KeyError:
                raise AssertionError(f"no id in response {response.json()}")
        return response

    def rpc_request(self, method, params=[], _id=None, client=None, url=None):
        client = client if client else requests.Session()
        url = url if url else option.rpc_url

        data = {"jsonrpc": "2.0", "method": method}
        if params:
            data["params"] = params
        data["id"] = _id if _id else 13

        response = client.post(url, json=data)

        return response

    def verify_json_schema(self, response, method):
        with open(f"{option.base_dir}/schemas/{method}", "r") as schema:
            jsonschema.validate(instance=response.json(), schema=json.load(schema))

class TransactionTestCase(RpcTestCase):

    def wallet_create_multi_transaction(self):

        method = "wallet_createMultiTransaction"
        params = [
            {
                "fromAddress": user_1.address,
                "fromAmount": "0x5af3107a4000",
                "fromAsset": "ETH",
                "multiTxType": "MultiTransactionSend",
                "toAddress": user_2.address,
                "toAsset": "ETH",
            },
            [
                {
                    "bridgeName": "Transfer",
                    "chainID": 31337,
                    "transferTx": {
                        "data": "",
                        "from": user_1.address,
                        "gas": "0x5BBF",
                        "input": "",
                        "maxFeePerGas": "0xbcc0f04fd",
                        "maxPriorityFeePerGas": "0x3b9aca00",
                        "to": user_2.address,
                        "type": "0x02",
                        "value": "0x5af3107a4000",
                    },
                }
            ],
            f"{option.password}",
        ]

        response = self.rpc_request(method, params, 13)
        self.verify_is_valid_json_rpc_response(response)
        return response

    def setup_method(self):
        super().setup_method()
        
        response = self.wallet_create_multi_transaction()
        try:
            self.tx_hash = response.json()["result"]["hashes"][str(self.network_id)][0]
        except (KeyError, json.JSONDecodeError):
            raise Exception(response.content)

