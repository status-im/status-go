from clients.rpc import RpcClient


class Service:
    def __init__(self, client: RpcClient, name: str):
        assert name is not ""
        self.rpc_client = client
        self.name = name

    def rpc_request(self, method: str, params=None):
        full_method_name = f"{self.name}_{method}"
        return self.rpc_client.rpc_request(full_method_name, params)
