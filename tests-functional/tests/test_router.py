import pytest
import time
import uuid
from conftest import user_1, user_2, option
from test_cases import SignalTestCase


@pytest.mark.rpc
@pytest.mark.transaction
@pytest.mark.wallet
class TestTransactionFromRoute(SignalTestCase):

    await_signals = [
        "wallet.suggested.routes",
        "wallet.router.sign-transactions",
        "wallet.router.sending-transactions-started",
        "wallet.transaction.status-changed",
        "wallet.router.transactions-sent"
    ]

    def test_tx_from_route(self):

        _uuid = str(uuid.uuid4())
        amount_in = "0xde0b6b3a7640000"

        method = "wallet_getSuggestedRoutesAsync"
        params = [
            {
                "uuid": _uuid,
                "sendType": 0,
                "addrFrom": user_1.address,
                "addrTo": user_2.address,
                "amountIn": amount_in,
                "amountOut": "0x0",
                "tokenID": "ETH",
                "tokenIDIsOwnerToken": False,
                "toTokenID": "",
                "disabledFromChainIDs": [10, 42161],
                "disabledToChainIDs": [10, 42161],
                "gasFeeMode": 1,
                "fromLockedAmount": {}
            }
        ]
        response = self.rpc_client.rpc_valid_request(method, params)

        routes = self.signal_client.wait_for_signal("wallet.suggested.routes")
        assert routes['event']['Uuid'] == _uuid

        method = "wallet_buildTransactionsFromRoute"
        params = [
            {
                "uuid": _uuid,
                "slippagePercentage": 0
            }
        ]
        response = self.rpc_client.rpc_valid_request(method, params)

        wallet_router_sign_transactions = self.signal_client.wait_for_signal(
            "wallet.router.sign-transactions")

        assert wallet_router_sign_transactions['event']['signingDetails']['signOnKeycard'] == False
        transaction_hashes = wallet_router_sign_transactions['event']['signingDetails']['hashes']

        assert transaction_hashes, "Transaction hashes are empty!"

        tx_signatures = {}

        for hash in transaction_hashes:

            method = "wallet_signMessage"
            params = [
                hash,
                user_1.address,
                option.password
            ]

            response = self.rpc_client.rpc_valid_request(method, params)

            if response.json()["result"].startswith("0x"):
                tx_signature = response.json()["result"][2:]

            signature = {
                "r": tx_signature[:64],
                "s": tx_signature[64:128],
                "v": tx_signature[128:]
            }

            tx_signatures[hash] = signature

        method = "wallet_sendRouterTransactionsWithSignatures"
        params = [
            {
                "uuid": _uuid,
                "Signatures": tx_signatures
            }
        ]
        response = self.rpc_client.rpc_valid_request(method, params)

        tx_status = self.signal_client.wait_for_signal(
            "wallet.transaction.status-changed")

        assert tx_status["event"]["chainId"] == 31337
        assert tx_status["event"]["status"] == "Success"
        tx_hash = tx_status["event"]["hash"]

        method = "eth_getTransactionByHash"
        params = [tx_hash]

        response = self.rpc_client.rpc_valid_request(method, params, url=option.anvil_url)
        tx_details = response.json()["result"]

        assert tx_details["value"] == amount_in
        assert tx_details["to"] == user_2.address
        assert tx_details["from"] == user_1.address
