from clients.rpc import RpcClient
from clients.services.service import Service


class WalletService(Service):
    def __init__(self, client: RpcClient):
        super().__init__(client, "wallet")

    def get_balances_at_by_chain(self, chains: list, addresses: list, tokens: list):
        params = [chains, addresses, tokens]
        return self.rpc_request("getBalancesByChain", params)
