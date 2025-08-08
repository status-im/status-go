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

    def add_account(self, password, account_data):
        params = [password, account_data]
        response = self.rpc_request("addAccount", params)
        return response.json()

    def delete_account(self, account_address):
        params = [account_address]
        response = self.rpc_request("deleteAccount", params)
        return response.json()

    def import_mnemonic(self, mnemonic, password):
        params = [mnemonic, password]
        response = self.rpc_request("importMnemonic", params)
        return response.json()

    def add_keypair_via_seed_phrase(self, mnemonic, password, name, wallet_account):
        params = [mnemonic, password, name, wallet_account]
        response = self.rpc_request("addKeypairViaSeedPhrase", params)
        return response.json()

    def add_keypair_via_private_key(self, private_key, password, name, wallet_account):
        params = [private_key, password, name, wallet_account]
        response = self.rpc_request("addKeypairViaPrivateKey", params)
        return response.json()

    def verify_password(self, password):
        params = [password]
        response = self.rpc_request("verifyPassword", params)
        return response.json()

    def resolve_suggested_path_for_keypair(self, key_uid):
        params = [key_uid]
        response = self.rpc_request("resolveSuggestedPathForKeypair", params)
        return response.json()
