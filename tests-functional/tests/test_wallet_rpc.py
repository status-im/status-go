import random

import pytest

from steps.status_backend import StatusBackendSteps


@pytest.mark.wallet
@pytest.mark.rpc
class TestRpc(StatusBackendSteps):

    @pytest.mark.parametrize(
        "method, params",
        [
            ("wallet_startWallet", []),
            ("wallet_getEthereumChains", []),
            ("wallet_getTokenList", []),
            ("wallet_getCryptoOnRamps", []),
            ("wallet_getCachedCurrencyFormats", []),
            (
                "wallet_fetchPrices",
                [
                    [
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
                    ],
                    ["usd"],
                ],
            ),
            (
                "wallet_fetchMarketValues",
                [
                    [
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
                    ],
                    "usd",
                ],
            ),
            (
                "wallet_fetchTokenDetails",
                [
                    [
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
                ],
            ),
            ("wallet_getWalletConnectActiveSessions", [1728995277]),
            ("wallet_stopSuggestedRoutesAsyncCalculation", []),
        ],
    )
    def test_(self, method, params):
        _id = str(random.randint(1, 8888))

        response = self.rpc_client.rpc_valid_request(method, params, _id)
        self.rpc_client.verify_json_schema(response.json(), method)
