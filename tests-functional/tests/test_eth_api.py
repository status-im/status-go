import pytest
from collections import namedtuple

from clients.anvil import Anvil
from clients.signals import SignalType
from resources.constants import user_1, user_2
from uuid import uuid4
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
        self.rpc_client = backend_recovered_profile(name="rpc_client", user=user_1)
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
        anvil = Anvil()

        anvil.eth.wait_for_transaction_receipt(tx_hash)
        receipt = anvil.transaction_receipt(tx_hash)
        block_number = receipt["blockNumber"]
        block_hash = receipt["blockHash"]
        TxData = namedtuple("TxData", ["tx_hash", "block_number", "block_hash"])
        self.tx_data = TxData(tx_hash, block_number, block_hash)
