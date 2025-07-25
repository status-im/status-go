import pytest
import json
from collections import namedtuple
from clients.signals import SignalType
from resources.constants import user_1, user_2, ANVIL_NETWORK_ID
from uuid import uuid4
from steps.eth_rpc import EthRpcSteps
from utils import wallet_utils


def validate_header(header, block_number, block_hash):
    assert header["number"] == block_number
    assert header["hash"] == block_hash


def validate_block(block, block_number, block_hash, expected_tx_hash):
    validate_header(block["header"], block_number, block_hash)
    tx_hashes = [tx["hash"] for tx in block["transactions"]]
    assert expected_tx_hash in tx_hashes


def validate_transaction(tx, tx_hash):
    assert tx["tx"]["hash"] == tx_hash


def validate_receipt(receipt, tx_hash, block_number, block_hash):
    assert receipt["transactionHash"] == tx_hash
    assert receipt["blockNumber"] == block_number
    assert receipt["blockHash"] == block_hash


@pytest.mark.rpc
@pytest.mark.ethclient
@pytest.mark.xdist_group(name="Eth")
class TestEth:
    await_signals = [
        SignalType.NODE_LOGIN.value,
        SignalType.WALLET.value,
        SignalType.WALLET_SUGGESTED_ROUTES.value,
        SignalType.WALLET_ROUTER_SIGN_TRANSACTIONS.value,
        SignalType.WALLET_ROUTER_SENDING_TRANSACTIONS_STARTED.value,
        SignalType.WALLET_ROUTER_TRANSACTIONS_SENT.value,
    ]

    @pytest.fixture(autouse=True)
    def setup_backend(self, backend_recovered_profile):
        self.rpc_client = backend_recovered_profile("rpc_client", user_1)
        self.network_id = ANVIL_NETWORK_ID
        """Create a transaction and assign tx_data to self."""
        uuid = str(uuid4())
        input_params = {
            "uuid": uuid,
            "sendType": 0,
            "addrFrom": user_1.address,
            "addrTo": user_2.address,
            "amountIn": "0xde0b6b3a7640000",
            "amountOut": "0x0",
            "tokenID": "ETH",
            "tokenIDIsOwnerToken": False,
            "toTokenID": "",
            "fromChainID": 31337,
            "toChainID": 31337,
            "gasFeeMode": 1,
            "slippagePercentage": 0,
        }
        tx_data = wallet_utils.send_router_transaction(self.rpc_client, **input_params)
        tx_hash = tx_data["tx_status"]["hash"]
        eth_helpers = EthRpcSteps()
        eth_helpers.wait_until_tx_not_pending(self.rpc_client, self.network_id, tx_hash)
        receipt = eth_helpers.get_transaction_receipt(self.rpc_client, self.network_id, tx_hash)
        try:
            block_number = receipt.json()["result"]["blockNumber"]
            block_hash = receipt.json()["result"]["blockHash"]
        except (KeyError, json.JSONDecodeError):
            raise Exception(receipt.content)
        TxData = namedtuple("TxData", ["tx_hash", "block_number", "block_hash"])
        self.tx_data = TxData(tx_hash, block_number, block_hash)

    def test_block_number(self):
        self.rpc_client.rpc_valid_request("ethclient_blockNumber", [self.network_id])

    def test_suggest_gas_price(self):
        self.rpc_client.rpc_valid_request("ethclient_suggestGasPrice", [self.network_id])

    def test_header_by_number(self):
        response = self.rpc_client.rpc_valid_request("ethclient_headerByNumber", [self.network_id, self.tx_data.block_number])
        validate_header(response.json()["result"], self.tx_data.block_number, self.tx_data.block_hash)

    def test_block_by_number(self):
        response = self.rpc_client.rpc_valid_request("ethclient_blockByNumber", [self.network_id, self.tx_data.block_number])
        validate_block(
            response.json()["result"],
            self.tx_data.block_number,
            self.tx_data.block_hash,
            self.tx_data.tx_hash,
        )

    def test_header_by_hash(self):
        response = self.rpc_client.rpc_valid_request("ethclient_headerByHash", [self.network_id, self.tx_data.block_hash])
        validate_header(response.json()["result"], self.tx_data.block_number, self.tx_data.block_hash)

    def test_block_by_hash(self):
        response = self.rpc_client.rpc_valid_request("ethclient_blockByHash", [self.network_id, self.tx_data.block_hash])
        validate_block(
            response.json()["result"],
            self.tx_data.block_number,
            self.tx_data.block_hash,
            self.tx_data.tx_hash,
        )

    def test_transaction_by_hash(self):
        response = self.rpc_client.rpc_valid_request("ethclient_transactionByHash", [self.network_id, self.tx_data.tx_hash])
        validate_transaction(response.json()["result"], self.tx_data.tx_hash)

    def test_transaction_receipt(self):
        response = self.rpc_client.rpc_valid_request("ethclient_transactionReceipt", [self.network_id, self.tx_data.tx_hash])
        validate_receipt(
            response.json()["result"],
            self.tx_data.tx_hash,
            self.tx_data.block_number,
            self.tx_data.block_hash,
        )
