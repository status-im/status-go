from clients.rpc import RpcClient
from clients.services.service import Service


class SettingsService(Service):
    def __init__(self, client: RpcClient):
        super().__init__(client, "settings")

    def get_accounts(self):
        response = self.rpc_request("getAccounts")
        return response.json()

    def get_account_keypairs(self):
        response = self.rpc_request("getKeypairs")
        return response.json()

    def get_pubkey(self, display_name):
        accounts = self.get_accounts().get("result", [])
        for account in accounts:
            if account.get("name") == display_name:
                return account.get("public-key")
        raise ValueError(f"Public key not found for display name: {display_name}")

    def get_settings(self):
        response = self.rpc_request("getSettings")
        return response.json()
