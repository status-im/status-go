import pytest
from web3 import Web3

from clients.alchemy_stub import AlchemyStub, NFT, PEER, TOKEN
from clients.api import ApiResponseError
from resources.constants import NATIVE_TOKEN_ADDRESS
from steps import messenger
from utils.config import Config

STUB_CHAIN_ID = 31338


def stub_network(port):
    host = "127.0.0.1" if Config.status_backend_urls else "host.docker.internal"
    return {
        "chainID": STUB_CHAIN_ID,
        "chainName": "Alchemy stub",
        "rpcProviders": [
            {
                "chainId": STUB_CHAIN_ID,
                "name": "direct-alchemy",
                "url": f"http://{host}:{port}/",
                "enableRpsLimiter": False,
                "type": "embedded-direct",
                "enabled": True,
                "authType": "no-auth",
            }
        ],
        "shortName": "stub",
        "nativeCurrencyName": "Ether",
        "nativeCurrencySymbol": "ETH",
        "nativeCurrencyDecimals": 18,
        "isTest": False,
        "layer": 1,
        "enabled": True,
        "isActive": False,
        "isDeactivatable": True,
    }


@pytest.mark.wallet
@pytest.mark.rpc
class TestWalletGetTransfers:

    @pytest.fixture(autouse=True)
    def setup_backend(self, backend_new_profile):
        self.alchemy = AlchemyStub()
        self.backend = backend_new_profile("get_transfers", extra_networks_override=[stub_network(self.alchemy.port)])
        self.address = Web3.to_checksum_address(messenger.wallet_address(self.backend)).lower()
        yield
        self.alchemy.close()

    def get_transfers(self, *params):
        return self.backend.wallet_service.rpc_request("getTransfers", list(params))

    def test_get_transfers(self):
        result = self.get_transfers([self.address], [STUB_CHAIN_ID])

        assert result["coverage"] == [
            {"address": self.address, "chainId": STUB_CHAIN_ID, "complete": True, "lastFetchedBlock": 0x103},
        ]
        assert any(r["method"] == "alchemy_getAssetTransfers" for r in self.alchemy.requests)

        transfers = result["transfers"]
        assert [t["category"] for t in transfers] == ["erc20", "external"]

        erc20, native = transfers
        assert erc20["direction"] == "out"
        assert erc20["from"] == self.address
        assert erc20["to"] == PEER
        assert erc20["tokenAddress"] == TOKEN
        assert erc20["amount"] == "-123456"
        assert erc20["value"] == "-0.123456"
        assert erc20["decimals"] == 6

        assert native["direction"] == "in"
        assert native["tokenAddress"] == NATIVE_TOKEN_ADDRESS
        assert native["amount"] == "1000000000000000000"
        assert native["value"] == "1"
        assert native["symbol"] == "ETH"
        assert native["timestamp"] == 1704067200
        assert native["blockNumber"] == 0x101

        with_nfts = self.get_transfers([self.address], [STUB_CHAIN_ID], True)["transfers"]
        assert [t["category"] for t in with_nfts] == ["erc721", "erc20", "external"]
        assert with_nfts[0]["tokenAddress"] == NFT
        assert with_nfts[0]["tokenId"] == "51"
        assert with_nfts[0]["amount"] == "1"

        page = self.get_transfers([self.address], [STUB_CHAIN_ID], True, 1, 1)["transfers"]
        assert [t["category"] for t in page] == ["erc20"]

    def test_get_transfers_without_provider(self):
        with pytest.raises(ApiResponseError, match="no history provider for chain 1"):
            self.get_transfers([self.address], [1])
