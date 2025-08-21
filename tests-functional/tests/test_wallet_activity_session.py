import json
import random
from utils import wallet_utils
import uuid as uuid_lib
import pytest

from resources.constants import user_1, user_2
from clients.signals import SignalType
from clients.anvil import Anvil
from clients.smart_contract_runner import SmartContractRunner
from clients.contract_deployers.snt import SNTDeployer, SNTV2_ABI, SNT_TOKEN_CONTROLLER_ABI
from resources.constants import DEPLOYER_ACCOUNT
from clients.contract_deployers.communities import CommunitiesDeployer
from web3 import Web3

EventActivityFilteringDone = "wallet-activity-filtering-done"
EventActivityFilteringUpdate = "wallet-activity-filtering-entries-updated"
EventActivitySessionUpdated = "wallet-activity-session-updated"


def validate_entry(entry, tx_data):
    assert entry["transactions"][0]["chainId"] == tx_data["tx_status"]["chainId"]
    assert entry["transactions"][0]["hash"] == tx_data["tx_status"]["hash"]


@pytest.mark.wallet
@pytest.mark.rpc
@pytest.mark.activity
@pytest.mark.xdist_group(name="WalletSteps")
class TestWalletActivitySession:
    await_signals = [
        SignalType.NODE_LOGIN.value,
        SignalType.WALLET.value,
        SignalType.WALLET_SUGGESTED_ROUTES.value,
        SignalType.WALLET_ROUTER_SIGN_TRANSACTIONS.value,
        SignalType.WALLET_ROUTER_SENDING_TRANSACTIONS_STARTED.value,
        SignalType.WALLET_ROUTER_TRANSACTIONS_SENT.value,
    ]

    @staticmethod
    def _token_list_to_token_overrides(token_list):
        token_overrides = []
        for token_symbol, token_address in token_list.items():
            token_overrides.append(
                {
                    "symbol": token_symbol,
                    "address": token_address,
                }
            )
        return token_overrides

    def get_snt_contract(self):
        return self.anvil_client.eth.contract(address=self.snt_deployer.snt_contract_address, abi=SNTV2_ABI)

    def get_snt_token_controller(self):
        return self.anvil_client.eth.contract(address=self.snt_deployer.snt_token_controller_address, abi=SNT_TOKEN_CONTROLLER_ABI)

    def mint_snt(self, address, amount):
        snt_controller = self.get_snt_token_controller()
        tx_hash = snt_controller.functions.generateTokens(address, amount).transact()
        self.anvil_client.eth.wait_for_transaction_receipt(tx_hash)

    @pytest.fixture(autouse=True)
    def setup_backend(self, backend_recovered_profile):
        # Setup contracts and deployers
        self.anvil_client = Anvil()
        self.anvil_client.eth.default_account = Web3.to_checksum_address(DEPLOYER_ACCOUNT.address)
        self.smart_contract_runner = SmartContractRunner()
        self.snt_deployer = SNTDeployer(self.smart_contract_runner)
        self.communities_deployer = CommunitiesDeployer(self.smart_contract_runner)
        self.erc20_token_list = {"SNT": self.snt_deployer.snt_contract_address}
        token_overrides = self._token_list_to_token_overrides(self.erc20_token_list)

        # Create backend
        self.rpc_client = backend_recovered_profile(name="rpc_client", user=user_1, token_overrides=token_overrides)

        self.request_id = str(random.randint(1, 8888))
        self.mint_snt(user_1.address, 1000000000000000000000000)

    def test_wallet_start_activity_filter_session(self):
        uuid = str(uuid_lib.uuid4())
        amount_in = "0xde0b6b3a7640000"

        input_params = {
            "uuid": uuid,
            "sendType": 0,
            "addrFrom": user_1.address,
            "addrTo": user_2.address,
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

        tx_data = []  # (routes, build_tx, tx_signatures, tx_status)
        # Set up a transactions for account before starting session
        tx_data.append(wallet_utils.send_router_transaction(self.rpc_client, **input_params))

        # Start activity session
        method = "wallet_startActivityFilterSessionV2"
        params = [
            [user_1.address],
            [self.rpc_client.network_id],  # type: ignore
            {
                "period": {"startTimestamp": 0, "endTimestamp": 0},
                "types": [],
                "statuses": [],
                "counterpartyAddresses": [],
                "assets": [],
                "collectibles": [],
                "filterOutAssets": False,
                "filterOutCollectibles": False,
            },
            10,
        ]
        self.rpc_client.prepare_wait_for_signal(
            "wallet",
            1,
            lambda signal: signal["event"]["type"] == EventActivityFilteringDone,
        )
        response = self.rpc_client.rpc_valid_request(method, params, self.request_id)
        event_response = self.rpc_client.wait_for_signal("wallet", timeout=10)["event"]

        # Check response
        sessionID = int(response.json()["result"])
        assert sessionID > 0

        # Check response event
        assert int(event_response["requestId"]) == sessionID
        message = json.loads(event_response["message"].replace("'", '"'))
        assert int(message["errorCode"]) == 1
        assert len(message["activities"]) > 0  # Should have at least 1 entry
        # First activity entry should match last sent transaction
        validate_entry(message["activities"][0], tx_data[-1])

        # Trigger new ETH transfer
        uuid = str(uuid_lib.uuid4())
        input_params["uuid"] = uuid

        self.rpc_client.prepare_wait_for_signal(
            "wallet",
            1,
            lambda signal: signal["event"]["type"] == EventActivitySessionUpdated and signal["event"]["requestId"] == sessionID,
        )
        tx_data.append(wallet_utils.send_router_transaction(self.rpc_client, **input_params))
        event_response = self.rpc_client.wait_for_signal("wallet", timeout=10)["event"]

        # Check response event
        assert int(event_response["requestId"]) == sessionID
        message = json.loads(event_response["message"].replace("'", '"'))
        assert message["hasNewOnTop"]  # New entries reported

        # Trigger new STT transfer
        uuid = str(uuid_lib.uuid4())
        input_params["uuid"] = uuid
        input_params["tokenID"] = "SNT"
        self.rpc_client.prepare_wait_for_signal(
            "wallet",
            1,
            lambda signal: signal["event"]["type"] == EventActivitySessionUpdated and signal["event"]["requestId"] == sessionID,
        )
        tx_data.append(wallet_utils.send_router_transaction(self.rpc_client, **input_params))
        event_response = self.rpc_client.wait_for_signal("wallet", timeout=10)["event"]

        # Check response event
        assert int(event_response["requestId"]) == sessionID
        message = json.loads(event_response["message"].replace("'", '"'))
        assert message["hasNewOnTop"]  # New entries reported

        # Reset activity session
        method = "wallet_resetActivityFilterSession"
        params = [sessionID, 10]
        self.rpc_client.prepare_wait_for_signal(
            "wallet",
            1,
            lambda signal: signal["event"]["type"] == EventActivityFilteringDone and signal["event"]["requestId"] == sessionID,
        )
        response = self.rpc_client.rpc_valid_request(method, params, self.request_id)
        event_response = self.rpc_client.wait_for_signal("wallet", timeout=10)["event"]

        # Check response event
        assert int(event_response["requestId"]) == sessionID
        message = json.loads(event_response["message"].replace("'", '"'))
        assert int(message["errorCode"]) == 1
        assert len(message["activities"]) > 2  # Should have at least 3 entries

        # First activity entry should match last sent transaction
        validate_entry(message["activities"][0], tx_data[-1])

        # Second activity entry should match second to last sent transaction
        validate_entry(message["activities"][1], tx_data[-2])

        # Third activity entry should match third to last sent transaction
        validate_entry(message["activities"][2], tx_data[-3])
