from clients.rpc import RpcClient
from clients.services.service import Service


class EthService(Service):
    def __init__(self, client: RpcClient):
        super().__init__(client, "eth")

    def estimate_gas(self, id: int, to: str, value: int):
        params = params = [id, {"to": to, "value": value}]
        response = self.rpc_request("estimateGas", params)
        return response
