from clients.anvil import Anvil
from clients.smart_contract_runner import SmartContractRunner
from clients.contract_deployers.snt import SNTDeployer, SNTV2_ABI, SNT_TOKEN_CONTROLLER_ABI
from clients.contract_deployers.communities import CommunitiesDeployer
from resources.constants import DEPLOYER_ACCOUNT
from steps.status_backend import StatusBackendSteps
from web3 import Web3


class WalletSteps(StatusBackendSteps):
    """
    WalletSteps is a test utility class for managing wallet-related operations
    in functional tests. All tests requiring Smart Contracts
    deployment need to run in the same worker to avoid deployment conflicts,
    which can cause failures. To achieve this, tests should be marked with
    @pytest.mark.xdist_group(name="WalletSteps").
    """

    erc20_token_list = {}

    @classmethod
    def setup_class(cls, skip_login=False):
        super().setup_class(skip_login=True)

        cls.anvil_client = Anvil()
        cls.anvil_client.eth.default_account = Web3.to_checksum_address(DEPLOYER_ACCOUNT.address)

        cls.smart_contract_runner = SmartContractRunner()
        cls.snt_deployer = SNTDeployer(cls.smart_contract_runner)
        cls.communities_deployer = CommunitiesDeployer(cls.smart_contract_runner)

        cls.erc20_token_list["SNT"] = cls.snt_deployer.snt_contract_address
        token_overrides = cls._token_list_to_token_overrides(cls.erc20_token_list)
        cls.rpc_client.restore_account_and_login(token_overrides=token_overrides)
        cls.rpc_client.wait_for_login()

    def get_snt_contract(self):
        return self.anvil_client.eth.contract(address=self.snt_deployer.snt_contract_address, abi=SNTV2_ABI)

    def get_snt_token_controller(self):
        return self.anvil_client.eth.contract(address=self.snt_deployer.snt_token_controller_address, abi=SNT_TOKEN_CONTROLLER_ABI)

    def mint_snt(self, address, amount):
        snt_controller = self.get_snt_token_controller()
        tx_hash = snt_controller.functions.generateTokens(address, amount).transact()
        self.anvil_client.eth.wait_for_transaction_receipt(tx_hash)

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
