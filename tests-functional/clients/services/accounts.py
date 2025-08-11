from clients.rpc import RpcClient
from clients.services.service import Service


class AccountService(Service):
    def __init__(self, client: RpcClient):
        super().__init__(client, "accounts")

    def get_accounts(self):
        response = self.rpc_request("getAccounts")
        return response.json()

    def get_account_keypairs(self):
        response = self.rpc_request("getKeypairs")
        return response.json()

    def add_account(self, account_data, **kwargs):
        params = ["", account_data]
        response = self.rpc_request("addAccount", params, **kwargs)
        return response.json()

    def delete_account(self, account_address):
        params = [account_address]
        response = self.rpc_request("deleteAccount", params)
        return response.json()

    def import_mnemonic(self, mnemonic, password):
        params = [mnemonic, password]
        response = self.rpc_request("importMnemonic", params)
        return response.json()

    def add_keypair(self, password, keypair):
        params = [password, keypair]
        response = self.rpc_request("addKeypair", params)
        return response.json()
