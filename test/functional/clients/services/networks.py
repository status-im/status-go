from clients.rpc import RpcClient
from clients.services.service import Service


class NetworksService(Service):
    def __init__(self, client: RpcClient):
        super().__init__(client, "networks")

    def add_ethereum_chain(self, network: dict):
        return self.rpc_request("addEthereumChain", [network])

    def delete_ethereum_chain(self, chain_id: int):
        return self.rpc_request("deleteEthereumChain", [chain_id])

    def get_ethereum_chains(self):
        return self.rpc_request("getEthereumChains")

    def get_flat_ethereum_chains(self):
        return self.rpc_request("getFlatEthereumChains")

    def set_chain_active(self, chain_id: int, active: bool):
        return self.rpc_request("setChainActive", [chain_id, active])

    def set_chain_enabled(self, chain_id: int, enabled: bool):
        return self.rpc_request("setChainEnabled", [chain_id, enabled])

    def set_chain_user_rpc_providers(self, chain_id: int, rpc_providers: list):
        return self.rpc_request("setChainUserRpcProviders", [chain_id, rpc_providers])
