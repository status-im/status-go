from clients.rpc import RpcClient
from clients.services.service import Service


class SettingsService(Service):
    def __init__(self, client: RpcClient):
        super().__init__(client, "settings")

    def get_settings(self, **kwargs):
        response = self.rpc_request("getSettings", **kwargs)
        return response.json()
