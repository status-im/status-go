from clients.rpc import RpcClient
from clients.services.service import Service


class WalletService(Service):
    def __init__(self, client: RpcClient):
        super().__init__(client, "wallet")

    def get_balances_at_by_chain(self, chains: list, addresses: list, tokens: list):
        params = [chains, addresses, tokens]
        return self.rpc_request("getBalancesByChain", params)

    def start_wallet(self):
        return self.rpc_request("startWallet")

    def get_derived_addresses_for_mnemonic(self, mnemonic: str, paths: list):
        params = [mnemonic, paths]
        return self.rpc_request("getDerivedAddressesForMnemonic", params)

    def send_router_transactions_with_signatures(self, uuid: str, tx_signatures: dict):
        params = [{"uuid": uuid, "Signatures": tx_signatures}]
        return self.rpc_request("sendRouterTransactionsWithSignatures", params)

    def get_owned_collectibles_async(self, params: dict):
        return self.rpc_request("getOwnedCollectiblesAsync", params)

    def start_activity_filter_session_v2(self, params: dict):
        return self.rpc_request("startActivityFilterSessionV2", params)

    def reset_activity_filter_session(self, session_id: int):
        params = [session_id]
        return self.rpc_request("resetActivityFilterSession", params)

    def set_fee_mode(self, path_tx_identity: dict, gas_fee_mode: int):
        params = [path_tx_identity, gas_fee_mode]
        return self.rpc_request("setFeeMode", params)

    def set_custom_tx_details(self, tx_identity_params: dict, tx_custom_params: dict):
        params = [tx_identity_params, tx_custom_params]
        return self.rpc_request("setCustomTxDetails", params)

    def get_suggested_routes_async(self, params: dict):
        return self.rpc_request("getSuggestedRoutesAsync", params)

    def build_transactions_from_route(self, uuid: str):
        params = [uuid]
        return self.rpc_request("buildTransactionsFromRoute", params)

    def sign_message(self, hash: str, address: str, password: str):
        params = [hash, address, password]
        return self.rpc_request("signMessage", params)

    def get_ethereum_chain(
        self,
    ):
        return self.rpc_request("getEthereumChains")

    def get_token_list(
        self,
    ):
        return self.rpc_request("getTokenList")

    def get_crypto_on_ramps(
        self,
    ):
        return self.rpc_request("getCryptoOnRamps")

    def get_cached_currency_formats(
        self,
    ):
        return self.rpc_request("getCachedCurrencyFormats")

    def fetch_prices(self, symbols: list, currencies: list):
        params = [symbols, currencies]
        return self.rpc_request("fetchPrices", params)

    def fetch_market_values(self, symbols: list, currency: str):
        params = [symbols, currency]
        return self.rpc_request("fetchMarketValues", params)

    def fetch_token_details(self, symbols: list):
        params = [symbols]
        return self.rpc_request("fetchTokenDetails", params)

    def get_wallet_connect_active_sessions(self, timestamp: int):
        params = [timestamp]
        return self.rpc_request("getWalletConnectActiveSessions", params)

    def stop_suggested_routes_async_calculation(self):
        return self.rpc_request("stopSuggestedRoutesAsyncCalculation")

    def fetch_or_get_cached_wallet_balances(self, addresses: list, force_refresh: bool = False):
        params = [addresses, force_refresh]
        return self.rpc_request("fetchOrGetCachedWalletBalances", params)
