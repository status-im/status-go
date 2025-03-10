from steps.wallet import WalletSteps


class TransactionSteps(WalletSteps):

    def setup_method(self):
        self.tx_hash = self.send_valid_multi_transaction()
