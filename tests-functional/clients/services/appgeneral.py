from clients.rpc import RpcClient
from clients.services.service import Service


class AppgeneralService(Service):
    def __init__(self, client: RpcClient):
        super().__init__(client, "appgeneral")

    def get_currencies(self):
        params = []
        response = self.rpc_request("getCurrencies", params)
        return response

    def version(self):
        params = []
        response = self.rpc_request("version", params)
        return response
