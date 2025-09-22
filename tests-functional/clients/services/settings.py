from clients.rpc import RpcClient
from clients.services.service import Service


class SettingsService(Service):
    def __init__(self, client: RpcClient):
        super().__init__(client, "settings")

    def get_settings(self):
        response = self.rpc_request("getSettings")
        return response

    def save_setting(self, key, value):
        params = [key, value]
        response = self.rpc_request("saveSetting", params)
        return response
