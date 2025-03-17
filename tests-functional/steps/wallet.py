from steps.status_backend import StatusBackendSteps
from clients.contract_deployers.snt import SNTV2_ABI, SNT_TOKEN_CONTROLLER_ABI


class WalletSteps(StatusBackendSteps):
    @classmethod
    def setup_class(self):
        super().setup_class()

    def get_snt_contract(self):
        return self.anvil_client.eth.contract(address=self.snt_deployer.snt_contract_address, abi=SNTV2_ABI)

    def get_snt_token_controller(self):
        return self.anvil_client.eth.contract(address=self.snt_deployer.snt_token_controller_address, abi=SNT_TOKEN_CONTROLLER_ABI)

    def mint_snt(self, address, amount):
        snt_controller = self.get_snt_token_controller()
        tx_hash = snt_controller.functions.generateTokens(address, amount).transact()
        self.anvil_client.eth.wait_for_transaction_receipt(tx_hash)
