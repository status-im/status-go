from clients.rpc import RpcClient
from clients.services.service import Service


class PreferencesService(Service):
    def __init__(self, client: RpcClient):
        super().__init__(client, "preferences")

    def get(self, category: str, key: str):
        return self.rpc_request("get", [category, key])

    def set(self, category: str, key: str, value: str):
        return self.rpc_request("set", [category, key, value])

    def get_all(self, category: str):
        return self.rpc_request("getAll", [category])

    def purge_unknown(self, category: str, valid_keys: list):
        return self.rpc_request("purgeUnknown", [category, valid_keys])

    def load_and_purge_unknown(self, category: str, valid_keys: list):
        return self.rpc_request("loadAndPurgeUnknown", [category, valid_keys])
