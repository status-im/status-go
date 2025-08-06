from clients.rpc import RpcClient
from clients.services.service import Service


class AccountService(Service):
    def __init__(self, client: RpcClient):
        super().__init__(client, "accounts")

    def get_accounts(self, **kwargs):
        response = self.rpc_request("getAccounts", **kwargs)
        return response.json()

    def get_account_keypairs(self, **kwargs):
        response = self.rpc_request("getKeypairs", **kwargs)
        return response.json()

    def add_account(self, account_data, **kwargs):
        params = ["", account_data]
        response = self.rpc_request("addAccount", params, **kwargs)
        return response.json()

    def delete_account(self, account_address, **kwargs):
        params = [account_address]
        response = self.rpc_request("deleteAccount", params, **kwargs)
        return response.json()

    def import_mnemonic(self, mnemonic, password, **kwargs):
        params = [mnemonic, password]
        response = self.rpc_request("importMnemonic", params, **kwargs)
        return response.json()

    def add_keypair(self, password, keypair, **kwargs):
        params = [password, keypair]
        response = self.rpc_request("addKeypair", params, **kwargs)
        return response.json()
