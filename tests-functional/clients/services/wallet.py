import logging
from typing import Optional

from clients.rpc import RpcClient
from clients.services.service import Service

logger = logging.getLogger(__name__)


class WalletService(Service):
    def __init__(self, client: RpcClient):
        super().__init__(client, "wallet")

    def get_balances_at_by_chain(self, chains: list, addresses: list, tokens: list):
        # If tokens is empty, use native tokens for the specified chains
        if not tokens:
            tokens = [f"{chain}-0x0000000000000000000000000000000000000000" for chain in chains]
        params = [addresses, tokens]
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
        return self.rpc_request("getSuggestedRoutesAsync", [params])

    def build_transactions_from_route(self, uuid: str):
        params = [uuid]
        return self.rpc_request("buildTransactionsFromRoute", params)

    def sign_message(self, address: str, password: str, message: str):
        params = [address, password, message]
        return self.rpc_request("signMessage", params)

    def get_ethereum_chain(
        self,
    ):
        return self.rpc_request("getEthereumChains")

    def get_all_token_lists(
        self,
    ):
        return self.rpc_request("getAllTokenLists")

    def get_crypto_on_ramps(
        self,
    ):
        return self.rpc_request("getCryptoOnRamps")

    def get_cached_currency_formats(
        self,
    ):
        return self.rpc_request("getCachedCurrencyFormats")

    def fetch_prices(self, tokens_keys: list, currencies: list):
        params = [tokens_keys, currencies]
        return self.rpc_request("fetchPrices", params)

    def fetch_market_values(self, tokens_keys: list, currency: str):
        params = [tokens_keys, currency]
        return self.rpc_request("fetchMarketValues", params)

    def fetch_token_details(self, tokens_keys: list):
        params = [tokens_keys]
        return self.rpc_request("fetchTokenDetails", params)

    def get_wallet_connect_active_sessions(self, timestamp: int):
        params = [timestamp]
        return self.rpc_request("getWalletConnectActiveSessions", params)

    def stop_suggested_routes_async_calculation(self):
        return self.rpc_request("stopSuggestedRoutesAsyncCalculation")

    def fetch_or_get_cached_wallet_balances(self, addresses: list, force_refresh: bool = False):
        params = [addresses, force_refresh]
        return self.rpc_request("fetchOrGetCachedWalletBalances", params)

    def restart_wallet_reload_timer(self):
        return self.rpc_request("restartWalletReloadTimer")

    def build_transaction(self, chain_id: int, send_tx_args_json: str):
        params = [chain_id, send_tx_args_json]
        return self.rpc_request("buildTransaction", params)

    def send_transaction_with_signature(self, chain_id: int, tx_type: int, send_tx_args_json: str, signature: str):
        params = [chain_id, tx_type, send_tx_args_json, signature]
        return self.rpc_request("sendTransactionWithSignature", params)

    def suggested_community_routes(
        self,
        uuid: str,
        send_type: int,
        chain_id: int,
        address_from: str,
        community_id: str,
        signer_pub_key: str,
        token_ids: list,
        wallet_addresses: list,
        transfer_details: list,
        addr_to: Optional[str] = None,
        signature: str = "",
        owner_token_parameters: Optional[dict] = None,
        master_token_parameters: Optional[dict] = None,
    ):
        community_params = {
            "communityID": community_id,
            "signerPubKey": signer_pub_key,
            "tokenIds": token_ids,
            "walletAddresses": wallet_addresses,
            "transferDetails": transfer_details,
            "tokenDeploymentSignature": signature,
        }
        if owner_token_parameters:
            community_params["ownerTokenParameters"] = owner_token_parameters
        if master_token_parameters:
            community_params["masterTokenParameters"] = master_token_parameters

        native_address = "0x0000000000000000000000000000000000000000"
        addr_to = addr_to if addr_to else native_address
        params = {
            "uuid": uuid,
            "sendType": send_type,
            "addrFrom": address_from,
            "addrTo": addr_to,
            "amountIn": "0x0",
            "tokenKey": f"{chain_id}-{native_address}",
            "toTokenKey": f"{chain_id}-{native_address}",
            "fromChainID": chain_id,
            "toChainID": chain_id,
            "gasFeeMode": 0,
            "communityRouteInputParams": community_params,
        }

        logger.info(f"Params to suggested_community_routes {params}")

        return self.rpc_request("getSuggestedRoutes", [params])
