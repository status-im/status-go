import pytest

from resources.constants import ANVIL_NETWORK_ID, user_1

TOKENS = [
    "WETH9",
    "USDC",
    "ZEENUS",
    "EUROC",
    "WEENUS",
    "XEENUS",
    "WETH",
    "ETH",
    "STT",
    "UNI",
    "YEENUS",
    "DAI",
]


@pytest.mark.wallet
@pytest.mark.rpc
class TestRpc:
    @pytest.fixture(autouse=True)
    def setup_backend(self, backend_recovered_profile):
        self.rpc_client = backend_recovered_profile(name="main_user", user=user_1)

    def test_start_wallet(self):
        result = self.rpc_client.wallet_service.start_wallet()
        assert result is None

    def test_get_ethereum_chain(self):
        result = self.rpc_client.wallet_service.get_ethereum_chain()
        assert result[0].get("Prod").get("chainId") == ANVIL_NETWORK_ID
        assert result[0].get("Prod").get("chainName") == "Anvil"
        assert result[0].get("Prod").get("nativeCurrencyName") == "Ether"
        assert result[0].get("Test") is None

    def test_get_token_list(self):
        result = self.rpc_client.wallet_service.get_token_list()
        assert result.get("data")[0].get("name") == "native"
        assert result.get("data")[0].get("tokens")[0].get("name") == "Ether"

    def test_get_crypto_on_ramps(self):
        result = self.rpc_client.wallet_service.get_crypto_on_ramps()
        assert result[0].get("description") == "The new standard for fiat to crypto"
        assert result[0].get("name") == "MoonPay"

    def test_get_cached_currency_formats(self):
        result = self.rpc_client.wallet_service.get_cached_currency_formats()
        assert result.get("AED").get("symbol") == "AED"

    def test_fetch_prices(self):
        result = self.rpc_client.wallet_service.fetch_prices(TOKENS, ["usd"])
        for symbol in TOKENS:
            assert symbol in result

    def test_fetch_market_values(self):
        result = self.rpc_client.wallet_service.fetch_market_values(TOKENS, "usd")
        for symbol in TOKENS:
            assert symbol in result

    def test_fetch_token_details(self):
        result = self.rpc_client.wallet_service.fetch_token_details(TOKENS)
        assert "USDC" in result
        assert "ETH" in result

    def test_get_wallet_connect_active_sessions(self):
        result = self.rpc_client.wallet_service.get_wallet_connect_active_sessions(1728995277)
        assert result is None

    def test_stop_suggested_routes_async_calculation(self):
        result = self.rpc_client.wallet_service.stop_suggested_routes_async_calculation()
        assert result is None
