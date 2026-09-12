from clients.rpc import RpcClient
from clients.services.service import Service


class AccountService(Service):
    def __init__(self, client: RpcClient):
        super().__init__(client, "accounts")

    def get_accounts(self):
        response = self.rpc_request("getAccounts")
        return response

    def get_account_keypairs(self):
        response = self.rpc_request("getKeypairs")
        return response

    def add_account(self, password, account_data):
        params = [password, account_data]
        response = self.rpc_request("addAccount", params)
        return response

    def delete_account(self, account_address, password):
        params = [account_address, password]
        response = self.rpc_request("deleteAccount", params)
        return response

    def import_mnemonic(self, mnemonic, password):
        params = [mnemonic, password]
        response = self.rpc_request("importMnemonic", params)
        return response

    def add_keypair_via_seed_phrase(self, mnemonic, password, name, cold_wallet, wallet_account):
        params = [mnemonic, password, name, cold_wallet, wallet_account]
        response = self.rpc_request("addKeypairViaSeedPhrase", params)
        return response

    def add_keypair_via_private_key(self, private_key, password, name, wallet_account):
        params = [private_key, password, name, wallet_account]
        response = self.rpc_request("addKeypairViaPrivateKey", params)
        return response

    def verify_password(self, password):
        params = [password]
        response = self.rpc_request("verifyPassword", params)
        return response

    def resolve_suggested_path_for_keypair(self, key_uid):
        params = [key_uid]
        response = self.rpc_request("resolveSuggestedPathForKeypair", params)
        return response

    def has_paired_devices(self):
        response = self.rpc_request("hasPairedDevices", [])
        return response

    def update_keypair_name(self, key_uid, name):
        params = [key_uid, name]
        response = self.rpc_request("updateKeypairName", params)
        return response

    def move_wallet_account(self, from_position, to_position):
        params = [from_position, to_position]
        response = self.rpc_request("moveWalletAccount", params)
        return response

    def update_token_preferences(self, preferences):
        params = [preferences]
        response = self.rpc_request("updateTokenPreferences", params)
        return response

    def get_token_preferences(self):
        response = self.rpc_request("getTokenPreferences", [])
        return response

    def update_collectible_preferences(self, preferences):
        params = [preferences]
        response = self.rpc_request("updateCollectiblePreferences", params)
        return response

    def get_collectible_preferences(self):
        response = self.rpc_request("getCollectiblePreferences", [])
        return response

    def get_account_by_address(self, address):
        params = [address]
        response = self.rpc_request("getAccountByAddress", params)
        return response

    def get_keypair_by_key_uid(self, key_uid):
        params = [key_uid]
        response = self.rpc_request("getKeypairByKeyUID", params)
        return response

    def update_account(self, account):
        params = [account]
        response = self.rpc_request("updateAccount", params)
        return response

    def migrate_non_profile_keypair_to_cold_wallet(self, key_uid, password, cold_wallet):
        params = [key_uid, password, cold_wallet]
        response = self.rpc_request("migrateNonProfileKeypairToColdWallet", params)
        return response

    def migrate_non_profile_cold_wallet_keypair_to_app(self, mnemonic, password):
        params = [mnemonic, password]
        response = self.rpc_request("migrateNonProfileColdWalletKeypairToApp", params)
        return response

    def get_random_mnemonic(self):
        response = self.rpc_request("getRandomMnemonic", [])
        return response

    def add_keypair_stored_to_cold_wallet(self, key_uid, master_address, name, wallet_xpub, cold_wallet, wallet_accounts):
        params = [
            key_uid,
            master_address,
            name,
            wallet_xpub,
            cold_wallet,
            wallet_accounts,
        ]
        response = self.rpc_request("addKeypairStoredToColdWallet", params)
        return response

    def update_keypair(self, keypair):
        params = [keypair]
        response = self.rpc_request("updateKeypair", params)
        return response

    def get_watch_only_accounts(self):
        response = self.rpc_request("getWatchOnlyAccounts", [])
        return response

    def delete_keypair(self, key_uid, password):
        params = [key_uid, password]
        response = self.rpc_request("deleteKeypair", params)
        return response

    def remaining_account_capacity(self):
        response = self.rpc_request("remainingAccountCapacity", [])
        return response

    def remaining_keypair_capacity(self):
        response = self.rpc_request("remainingKeypairCapacity", [])
        return response

    def remaining_watch_only_account_capacity(self):
        response = self.rpc_request("remainingWatchOnlyAccountCapacity", [])
        return response

    def get_num_of_addresses_to_generate_for_keypair(self, key_uid):
        params = [key_uid]
        response = self.rpc_request("getNumOfAddressesToGenerateForKeypair", params)
        return response

    def verify_keystore_file_for_account(self, address, password):
        params = [address, password]
        response = self.rpc_request("verifyKeystoreFileForAccount", params)
        return response
