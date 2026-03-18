from clients.services.service import Service


class EnsService(Service):
    def __init__(self, client):
        super().__init__(client, "ens")

    def add(self, chain_id, username):
        return self.rpc_request("add", [chain_id, username])

    def remove(self, chain_id, username):
        return self.rpc_request("remove", [chain_id, username])

    def get_ens_usernames(self):
        return self.rpc_request("getEnsUsernames")

    def get_registrar_address(self, chain_id):
        return self.rpc_request("getRegistrarAddress", [chain_id])

    def resolver(self, chain_id, username):
        return self.rpc_request("resolver", [chain_id, username])

    def owner_of(self, chain_id, username):
        return self.rpc_request("ownerOf", [chain_id, username])

    def public_key_of(self, chain_id, username):
        return self.rpc_request("publicKeyOf", [chain_id, username])

    def address_of(self, chain_id, username):
        return self.rpc_request("addressOf", [chain_id, username])

    def expire_at(self, chain_id, username):
        return self.rpc_request("expireAt", [chain_id, username])

    def price(self, chain_id):
        return self.rpc_request("price", [chain_id])
