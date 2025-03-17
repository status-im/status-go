from clients.services.wallet import WalletService
from clients.signals import SignalType
from clients.anvil import Anvil
from clients.status_backend import StatusBackend
from clients.smart_contract_runner import SmartContractRunner
from clients.contract_deployers.snt import SNTDeployer
from clients.contract_deployers.communities import CommunitiesDeployer
from conftest import option
from resources.constants import ANVIL_NETWORK_ID, DEPLOYER_ACCOUNT
from web3 import Web3


class StatusBackendSteps:

    reuse_container = True  # Skip close_status_backend_containers cleanup
    await_signals = [SignalType.NODE_LOGIN.value]

    network_id = ANVIL_NETWORK_ID

    erc20_token_list = {}

    @classmethod
    def setup_class(self):
        self.anvil_client = Anvil()
        self.anvil_client.eth.default_account = Web3.to_checksum_address(DEPLOYER_ACCOUNT.address)

        self.smart_contract_runner = SmartContractRunner()

        self.snt_deployer = SNTDeployer(self.smart_contract_runner)
        self.erc20_token_list["SNT"] = self.snt_deployer.snt_contract_address

        self.communities_deployer = CommunitiesDeployer(self.smart_contract_runner)

        self.rpc_client = StatusBackend(await_signals=self.await_signals)
        self.wallet_service = WalletService(self.rpc_client)

        self.rpc_client.init_status_backend()

        token_overrides = self._token_list_to_token_overrides(self.erc20_token_list)
        self.rpc_client.restore_account_and_login(token_overrides=token_overrides)
        self.rpc_client.wait_for_login()

    def teardown_class(self):
        for status_backend in option.status_backend_containers:
            status_backend.container.stop(timeout=10)
            option.status_backend_containers.remove(status_backend)
            status_backend.container.remove()

    @classmethod
    def _token_list_to_token_overrides(cls, token_list):
        token_overrides = []
        for token_symbol, token_address in token_list.items():
            token_overrides.append(
                {
                    "symbol": token_symbol,
                    "address": token_address,
                }
            )
        return token_overrides
