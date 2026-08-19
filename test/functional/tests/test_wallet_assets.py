import uuid as uuid_lib
import json
import pytest
from resources.constants import BURN_ADDRESS
import resources.constants as constants

from clients.signals import SignalType, WalletEventType
from utils import wallet_utils
from clients.services.wallet import WalletService


@pytest.mark.rpc
@pytest.mark.assets
@pytest.mark.wallet
class TestWalletAssets:

    @pytest.fixture(autouse=True)
    def setup_backend(self, funded_new_profile, multicall3_deployer):
        self.rpc_client, self.wallet_address = funded_new_profile(name="rpc_client", multicall_contract_address=multicall3_deployer.contract_address)
        self.wallet_service = WalletService(self.rpc_client)

    def test_balance_refresh_ticker_after_sending_transaction(self):
        uuid = str(uuid_lib.uuid4())
        gas_fee_mode = constants.gas_fee_mode_medium
        amount_in = "0xde0b6b3a7640000"
        native_token_key = wallet_utils.get_token_key(constants.ANVIL_NETWORK_ID, constants.NATIVE_TOKEN_ADDRESS)
        input_params = {
            "uuid": uuid,
            "sendType": 0,
            "addrFrom": self.wallet_address,
            "addrTo": BURN_ADDRESS,
            "amountIn": amount_in,
            "amountOut": "0x0",
            "tokenKey": native_token_key,
            "tokenIDIsOwnerToken": False,
            "toTokenKey": native_token_key,
            "fromChainID": constants.ANVIL_NETWORK_ID,
            "toChainID": constants.ANVIL_NETWORK_ID,
            "gasFeeMode": gas_fee_mode,
            # params for building tx from route
            "slippagePercentage": 0,
        }

        # Prepare to send tx
        routes = wallet_utils.get_suggested_routes(self.rpc_client, **input_params)
        assert "Route" in routes, f"No route found: {routes}"
        build_tx = wallet_utils.build_transactions_from_route(self.rpc_client, input_params.get("uuid"))
        tx_signatures = wallet_utils.sign_messages(self.rpc_client, build_tx["signingDetails"]["hashes"], input_params.get("addrFrom"))

        # Send tx, listen to reload tick signal
        def accept_fn(signal):
            match signal["event"]["type"]:
                case WalletEventType.TRANSACTIONS_PENDING_TRANSACTION_STATUS_CHANGED.value:
                    tx_status = json.loads(signal["event"]["message"].replace("'", '"'))
                    return tx_status["status"] == "Success"
                case WalletEventType.WALLET_TICK_RELOAD.value:
                    return True
                case _:
                    return False

        with self.rpc_client.expect_signal(
            SignalType.WALLET,
            count=2,
            accept_fn=accept_fn,
        ) as exp:
            _ = self.rpc_client.wallet_service.send_router_transactions_with_signatures(uuid, tx_signatures)
        signals = exp.result

        # Verify
        assert len(signals) == 2
