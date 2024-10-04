import pytest
from conftest import option
from test_cases import EthApiTestCase

def validateHeader(header, block_number, block_hash):
    assert header["number"] == block_number
    assert header["hash"] == block_hash

def validateBlock(block, block_number, block_hash, expected_tx_hash):
    validateHeader(block["header"], block_number, block_hash)
    tx_hashes = [tx["hash"] for tx in block["transactions"]]
    assert expected_tx_hash in tx_hashes

def validateTransaction(tx, tx_hash):
    assert tx["tx"]["hash"] == tx_hash

def validateReceipt(receipt, tx_hash, block_number, block_hash):
    assert receipt["transactionHash"] == tx_hash
    assert receipt["blockNumber"] == block_number
    assert receipt["blockHash"] == block_hash

@pytest.mark.rpc
@pytest.mark.ethclient
class TestRpc(EthApiTestCase):
    def test_block_number(self):
        self.rpc_valid_request("ethclient_blockNumber", [self.network_id])

    def test_suggest_gas_price(self):
        self.rpc_valid_request("ethclient_suggestGasPrice", [self.network_id])

    def test_header_by_number(self, tx_data):
        response = self.rpc_valid_request("ethclient_headerByNumber", [self.network_id, tx_data.block_number])
        validateHeader(response.json()["result"], tx_data.block_number, tx_data.block_hash)

    def test_block_by_number(self, tx_data):
        response = self.rpc_valid_request("ethclient_blockByNumber", [self.network_id, tx_data.block_number])
        validateBlock(response.json()["result"], tx_data.block_number, tx_data.block_hash, tx_data.tx_hash)

    def test_header_by_hash(self, tx_data):
        response = self.rpc_valid_request("ethclient_headerByHash", [self.network_id, tx_data.block_hash])
        validateHeader(response.json()["result"], tx_data.block_number, tx_data.block_hash)

    def test_block_by_hash(self, tx_data):
        response = self.rpc_valid_request("ethclient_blockByHash", [self.network_id, tx_data.block_hash])
        validateBlock(response.json()["result"], tx_data.block_number, tx_data.block_hash, tx_data.tx_hash)

    def test_transaction_by_hash(self, tx_data):
        response = self.rpc_valid_request("ethclient_transactionByHash", [self.network_id, tx_data.tx_hash])
        validateTransaction(response.json()["result"], tx_data.tx_hash)

    def test_transaction_receipt(self, tx_data):
        response = self.rpc_valid_request("ethclient_transactionReceipt", [self.network_id, tx_data.tx_hash])
        validateReceipt(response.json()["result"], tx_data.tx_hash, tx_data.block_number, tx_data.block_hash)
