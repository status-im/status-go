import uuid as uuid_lib
import json
import pytest
import resources.constants as constants

from clients.signals import SignalType, WalletEventType
from utils import wallet_utils
from clients.services.wallet import WalletService


@pytest.mark.rpc
@pytest.mark.assets
@pytest.mark.wallet
class TestWalletAssets:
    await_signals = [
        SignalType.NODE_LOGIN.value,
        SignalType.WALLET.value,
        SignalType.WALLET_SUGGESTED_ROUTES.value,
        SignalType.WALLET_ROUTER_SIGN_TRANSACTIONS.value,
        SignalType.WALLET_ROUTER_SENDING_TRANSACTIONS_STARTED.value,
        SignalType.WALLET_ROUTER_TRANSACTIONS_SENT.value,
    ]

    @pytest.fixture(autouse=True)
    def setup_backend(self, backend_factory_class):
        self.rpc_client = backend_factory_class(name="rpc_client", user=constants.user_1)
        self.wallet_service = WalletService(self.rpc_client)
        self.wallet_service.start_wallet()

    def test_balance_refresh_ticker_after_sending_transaction(self):
        uuid = str(uuid_lib.uuid4())
        amount_in = "0xde0b6b3a7640000"

        input_params = {
            "uuid": uuid,
            "sendType": 0,
            "addrFrom": constants.user_1.address,
            "addrTo": constants.user_2.address,
            "amountIn": amount_in,
            "amountOut": "0x0",
            "tokenID": "ETH",
            "tokenIDIsOwnerToken": False,
            "toTokenID": "",
            "fromChainID": 31337,
            "toChainID": 31337,
            "gasFeeMode": 1,
            # params for building tx from route
            "slippagePercentage": 0,
        }

        # Prepare to send tx
        routes = wallet_utils.get_suggested_routes(self.rpc_client, **input_params)
        assert "Route" in routes, f"No route found: {routes}"
        build_tx = wallet_utils.build_transactions_from_route(self.rpc_client, input_params.get("uuid"))
        tx_signatures = wallet_utils.sign_messages(self.rpc_client, build_tx["signingDetails"]["hashes"], input_params.get("addrFrom"))

        # Send tx, listen to reload tick signal
        method = "wallet_sendRouterTransactionsWithSignatures"
        params = [{"uuid": uuid, "Signatures": tx_signatures}]

        def accept_fn(signal):
            match signal["event"]["type"]:
                case WalletEventType.TRANSACTIONS_PENDING_TRANSACTION_STATUS_CHANGED.value:
                    tx_status = json.loads(signal["event"]["message"].replace("'", '"'))
                    return tx_status["status"] == "Success"
                case WalletEventType.WALLET_TICK_RELOAD.value:
                    return True
                case _:
                    return False

        self.rpc_client.prepare_wait_for_signal(
            SignalType.WALLET.value,
            2,
            accept_fn,
        )
        _ = self.rpc_client.rpc_valid_request(method, params)
        signals = self.rpc_client.wait_for_signal(SignalType.WALLET.value)

        # Verify
        assert len(signals) == 2
