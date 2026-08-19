from clients.rpc import RpcClient
from clients.services.service import Service


class MultiAccountsService(Service):
    def __init__(self, client: RpcClient):
        super().__init__(client, "multiaccounts")

    def store_identity_image(self, key_uid: str, path: str, ax: int, ay: int, bx: int, by: int):
        params = [key_uid, path, ax, ay, bx, by]
        response = self.rpc_request("storeIdentityImage", params)
        return response

    def get_identity_images(self, key_uid: str):
        params = [key_uid]
        response = self.rpc_request("getIdentityImages", params)
        return response
