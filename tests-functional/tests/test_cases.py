import json
import websocket
import threading
import logging
import jsonschema
import time
import requests
from conftest import option, user_1, user_2
import pytest
from collections import namedtuple

class RpcTestCase:
    network_id = 31337

    def setup_method(self):
        pass

    def _try_except_JSONDecodeError_KeyError(self, response, key: str):
        try:
            return response.json()[key]
        except json.JSONDecodeError:
            raise AssertionError(
                f"Invalid JSON in response: {response.content}")
        except KeyError:
            raise AssertionError(
                f"Key '{key}' not found in the JSON response: {response.content}")

    def verify_is_valid_json_rpc_response(self, response, _id=None):
        assert response.status_code == 200
        assert response.content
        self._try_except_JSONDecodeError_KeyError(response, "result")

        if _id:
            try:
                if _id != response.json()["id"]:
                    raise AssertionError(
                        f"got id: {response.json()['id']} instead of expected id: {_id}"
                    )
            except KeyError:
                raise AssertionError(f"no id in response {response.json()}")
        return response

    def verify_is_json_rpc_error(self, response):
        assert response.status_code == 200
        assert response.content
        self._try_except_JSONDecodeError_KeyError(response, "error")

    def rpc_request(self, method, params=[], _id=None, client=None, url=None):
        client = client if client else requests.Session()
        url = url if url else option.rpc_url

        data = {"jsonrpc": "2.0", "method": method}
        if params:
            data["params"] = params
        data["id"] = _id if _id else 13

        response = client.post(url, json=data)

        return response

    def rpc_valid_request(self, method, params=[], _id=None, client=None, url=None):
        response = self.rpc_request(method, params, _id, client, url)
        self.verify_is_valid_json_rpc_response(response, _id)
        return response

    def verify_json_schema(self, response, method):
        with open(f"{option.base_dir}/schemas/{method}", "r") as schema:
            jsonschema.validate(instance=response.json(),
                                schema=json.load(schema))

class WalletTestCase(RpcTestCase):
    def setup_method(self):
        super().setup_method()

    def wallet_create_multi_transaction(self, **kwargs):
        method = "wallet_createMultiTransaction"
        transferTx_data = {
            "data": "",
            "from": user_1.address,
            "gas": "0x5BBF",
            "input": "",
            "maxFeePerGas": "0xbcc0f04fd",
            "maxPriorityFeePerGas": "0xbcc0f04fd",
            "to": user_2.address,
            "type": "0x02",
            "value": "0x5af3107a4000",
        }
        for key, new_value in kwargs.items():
            if key in transferTx_data:
                transferTx_data[key] = new_value
            else:
                print(
                    f"Warning: The key '{key}' does not exist in the transferTx parameters and will be ignored.")
        params = [
            {
                "fromAddress": user_1.address,
                "fromAmount": "0x5af3107a4000",
                "fromAsset": "ETH",
                "type": 0, # MultiTransactionSend
                "toAddress": user_2.address,
                "toAsset": "ETH",
            },
            [
                {
                    "bridgeName": "Transfer",
                    "chainID": 31337,
                    "transferTx": transferTx_data
                }
            ],
            f"{option.password}",
        ]
        return self.rpc_request(method, params)

    def send_valid_multi_transaction(self, **kwargs):
        response = self.wallet_create_multi_transaction(**kwargs)

        tx_hash = None
        self.verify_is_valid_json_rpc_response(response)
        try:
            tx_hash = response.json(
        )["result"]["hashes"][str(self.network_id)][0]
        except (KeyError, json.JSONDecodeError):
            raise Exception(response.content)
        return tx_hash

class TransactionTestCase(WalletTestCase):
    def setup_method(self):
        super().setup_method()

        self.tx_hash = self.send_valid_multi_transaction()

class EthApiTestCase(WalletTestCase):
    @pytest.fixture(autouse=True, scope='class')
    def tx_data(self):
        tx_hash = self.send_valid_multi_transaction()
        self.wait_until_tx_not_pending(tx_hash)

        receipt = self.get_transaction_receipt(tx_hash)
        try:
            block_number = receipt.json()["result"]["blockNumber"]
            block_hash = receipt.json()["result"]["blockHash"]
        except (KeyError, json.JSONDecodeError):
            raise Exception(receipt.content)
        
        TxData = namedtuple("TxData", ["tx_hash", "block_number", "block_hash"])
        return TxData(tx_hash, block_number, block_hash)

    def get_block_header(self, block_number):
        method = "ethclient_headerByNumber"
        params = [self.network_id, block_number]
        return self.rpc_valid_request(method, params)

    def get_transaction_receipt(self, tx_hash):
        method = "ethclient_transactionReceipt"
        params = [self.network_id, tx_hash]
        return self.rpc_valid_request(method, params)

    def wait_until_tx_not_pending(self, tx_hash, timeout=10):
        method = "ethclient_transactionByHash"
        params = [self.network_id, tx_hash]
        response = self.rpc_valid_request(method, params)

        start_time = time.time()
        while response.json()["result"]["isPending"] == True:
            time_passed = time.time() - start_time
            if time_passed >= timeout:
                raise TimeoutError(
                    f"Tx {tx_hash} is still pending after {timeout} seconds")
            time.sleep(0.5)
            response = self.rpc_valid_request(method, params)
        return response.json()["result"]["tx"]

class SignalTestCase(RpcTestCase):

    await_signals = []
    received_signals = {}

    def on_message(self, ws, signal):
        signal = json.loads(signal)
        if signal.get("type") in self.await_signals:
            self.received_signals[signal["type"]] = signal

    def wait_for_signal(self, signal_type, timeout=10):
        start_time = time.time()
        while signal_type not in self.received_signals:
            time_passed = time.time() - start_time
            if time_passed >= timeout:
                raise TimeoutError(
                    f"Signal {signal_type} is not  received in {timeout} seconds")
            time.sleep(0.5)
        return self.received_signals[signal_type]

    def _on_error(self, ws, error):
        logging.info(f"Error: {error}")

    def _on_close(self, ws, close_status_code, close_msg):
        logging.info(f"Connection closed: {close_status_code}, {close_msg}")

    def _on_open(self, ws):
        logging.info("Connection opened")

    def _connect(self):
        self.url = f"{option.ws_url}/signals"

        ws = websocket.WebSocketApp(self.url,
                                    on_message=self.on_message,
                                    on_error=self._on_error,
                                    on_close=self._on_close)

        ws.on_open = self._on_open

        ws.run_forever()

    def setup_method(self):
        super().setup_method()

        websocket_thread = threading.Thread(target=self._connect)
        websocket_thread.daemon = True
        websocket_thread.start()
