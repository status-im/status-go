import pytest

from resources.constants import ANVIL_NETWORK_ID, user_1

TOKENS_KEYS = [
    "1-0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",  # WETH
    "1-0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",  # USDC
    "1-0x0000000000000000000000000000000000000000",  # ETH
    "1-0x744d70fdbe2ba4cf95131626614a1763df805b9e",  # SNT
]

# Live CoinGecko / market.status.im — https://github.com/status-im/status-app/issues/21135
_SKIP_LIVE_MARKET_RPC = pytest.mark.skip(
    reason="https://github.com/status-im/status-app/issues/21135 — re-enable when market proxy puzzle auth is wired in functional tests"
)


@pytest.mark.wallet
@pytest.mark.rpc
class TestRpc:
    @pytest.fixture(autouse=True)
    def setup_backend(self, request, backend_recovered_profile):
        # Add Ethereum mainnet alongside Anvil so token keys like `1-0x...` can be resolved by the token manager.
        extra_kwargs = {}
        if request.node.name in (
            "test_fetch_prices",
            "test_fetch_market_values",
            "test_fetch_token_details",
        ):
            extra_kwargs["extra_networks_override"] = [
                {
                    "chainID": 1,
                    "chainName": "Ethereum Mainnet",
                    "rpcProviders": [
                        {
                            "chainId": 1,
                            "name": "Mainnet Dummy",
                            "url": "http://localhost:0",
                            "enableRpsLimiter": False,
                            "type": "embedded-direct",
                            "enabled": True,
                            "authType": "no-auth",
                        }
                    ],
                    "shortName": "eth",
                    "nativeCurrencyName": "Ether",
                    "nativeCurrencySymbol": "ETH",
                    "nativeCurrencyDecimals": 18,
                    "isTest": False,
                    "layer": 1,
                    "enabled": True,
                    "isActive": True,
                    "isDeactivatable": True,
                }
            ]

        self.rpc_client = backend_recovered_profile(name="main_user", user=user_1, **extra_kwargs)

    def test_start_wallet(self):
        result = self.rpc_client.wallet_service.start_wallet()
        assert result is None

    def test_get_ethereum_chain(self):
        result = self.rpc_client.networks_service.get_ethereum_chains()
        assert result[0].get("Prod").get("chainId") == ANVIL_NETWORK_ID
        assert result[0].get("Prod").get("chainName") == "Anvil"
        assert result[0].get("Prod").get("nativeCurrencyName") == "Ether"
        assert result[0].get("Test") is None

    def test_get_all_token_lists(self):
        result = self.rpc_client.wallet_service.get_all_token_lists()
        for token_list in result:
            if token_list.get("name") != "Native tokens":
                continue
            tokens = token_list.get("tokens")
            # search for native token on Anvil chain
            cross_chain_id = "eth-native"
            native_token = next(
                (token for token in tokens if token.get("crossChainId") == cross_chain_id and token.get("chainId") == ANVIL_NETWORK_ID), None
            )
            assert native_token is not None
            assert native_token.get("name") == "Ethereum"
            assert native_token.get("symbol") == "ETH"
            assert native_token.get("decimals") == 18
            expected_logo_uri = (
                "https://raw.githubusercontent.com/trustwallet/assets/master/blockchains/ethereum/assets/"
                "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2/logo.png"
            )
            assert native_token.get("logoUri") == expected_logo_uri
            assert native_token.get("custom") is False

    def test_get_crypto_on_ramps(self):
        result = self.rpc_client.wallet_service.get_crypto_on_ramps()
        assert result[0].get("description") == "The new standard for fiat to crypto"
        assert result[0].get("name") == "MoonPay"

    def test_get_cached_currency_formats(self):
        result = self.rpc_client.wallet_service.get_cached_currency_formats()
        assert result.get("AED").get("key") == "AED"

    @_SKIP_LIVE_MARKET_RPC
    def test_fetch_prices(self):
        result = self.rpc_client.wallet_service.fetch_prices(TOKENS_KEYS, ["usd"])
        for key in TOKENS_KEYS:
            assert key.lower() in result

    @_SKIP_LIVE_MARKET_RPC
    def test_fetch_market_values(self):
        result = self.rpc_client.wallet_service.fetch_market_values(TOKENS_KEYS, "usd")
        for key in TOKENS_KEYS:
            assert key.lower() in result

    @_SKIP_LIVE_MARKET_RPC
    def test_fetch_token_details(self):
        result = self.rpc_client.wallet_service.fetch_token_details(TOKENS_KEYS)
        for key in TOKENS_KEYS:
            assert key.lower() in result

    def test_stop_suggested_routes_async_calculation(self):
        result = self.rpc_client.wallet_service.stop_suggested_routes_async_calculation()
        assert result is None
