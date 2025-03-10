import json
import time
from collections import namedtuple
import pytest
from steps.wallet import WalletSteps


class EthRpcSteps(WalletSteps):

    @pytest.fixture(autouse=True, scope="class")
    def tx_data(self):
        tx_hash = self.send_valid_multi_transaction()
        self.wait_until_tx_not_pending(tx_hash)

        receipt = self.get_transaction_receipt(tx_hash)
        try:
            block_number = receipt.json()["result"]["blockNumber"]
            block_hash = receipt.json()["result"]["blockHash"]
        except (KeyError, json.JSONDecodeError):
            raise Exception(receipt.content)

        tx_data = namedtuple("TxData", ["tx_hash", "block_number", "block_hash"])
        return tx_data(tx_hash, block_number, block_hash)

    def get_block_header(self, block_number):
        method = "ethclient_headerByNumber"
        params = [self.network_id, block_number]
        return self.rpc_client.rpc_valid_request(method, params)

    def get_transaction_receipt(self, tx_hash):
        method = "ethclient_transactionReceipt"
        params = [self.network_id, tx_hash]
        return self.rpc_client.rpc_valid_request(method, params)

    def wait_until_tx_not_pending(self, tx_hash, timeout=10):
        method = "ethclient_transactionByHash"
        params = [self.network_id, tx_hash]
        response = self.rpc_client.rpc_valid_request(method, params)

        start_time = time.time()
        while response.json()["result"]["isPending"] is True:
            time_passed = time.time() - start_time
            if time_passed >= timeout:
                raise TimeoutError(f"Tx {tx_hash} is still pending after {timeout} seconds")
            time.sleep(0.5)
            response = self.rpc_client.rpc_valid_request(method, params)
        return response.json()["result"]["tx"]
