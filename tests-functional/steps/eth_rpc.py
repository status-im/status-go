import time


class EthRpcSteps:

    def get_block_header(self, rpc_client, network_id, block_number):
        method = "ethclient_headerByNumber"
        params = [network_id, block_number]
        return rpc_client.rpc_valid_request(method, params)

    def get_transaction_receipt(self, rpc_client, network_id, tx_hash):
        method = "ethclient_transactionReceipt"
        params = [network_id, tx_hash]
        return rpc_client.rpc_valid_request(method, params)

    def wait_until_tx_not_pending(self, rpc_client, network_id, tx_hash, timeout=10):
        method = "ethclient_transactionByHash"
        params = [network_id, tx_hash]
        response = rpc_client.rpc_valid_request(method, params)

        start_time = time.time()
        while response.json()["result"]["isPending"] is True:
            time_passed = time.time() - start_time
            if time_passed >= timeout:
                raise TimeoutError(f"Tx {tx_hash} is still pending after {timeout} seconds")
            time.sleep(0.5)
            response = rpc_client.rpc_valid_request(method, params)
        return response.json()["result"]["tx"]
