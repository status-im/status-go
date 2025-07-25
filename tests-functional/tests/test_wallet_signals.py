import json
import random
import pytest

from resources.constants import user_1
from clients.signals import SignalType


@pytest.mark.wallet
@pytest.mark.rpc
class TestWalletSignals:
    await_signals = [SignalType.NODE_LOGIN.value, SignalType.WALLET.value]

    @pytest.fixture(autouse=True)
    def setup_backend(self, backend_factory_class):
        self.request_id = str(random.randint(1, 8888))
        self.rpc_client = backend_factory_class(name="rpc_client", user=user_1)

    @pytest.mark.skip  # TODO: returns empty response in most of the cases, so needs to be fixed with attention of required signals in signal_response
    def test_wallet_get_owned_collectibles_async(self):
        method = "wallet_getOwnedCollectiblesAsync"
        params = [
            0,
            [
                self.network_id,  # type: ignore
            ],
            [user_1.address],
            None,
            0,
            25,
            1,
            {"fetch-type": 2, "max-cache-age-seconds": 3600},
        ]
        self.rpc_client.rpc_valid_request(method, params, self.request_id)
        signal_response = self.rpc_client.wait_for_signal(SignalType.WALLET.value, timeout=60)
        self.rpc_client.verify_json_schema(signal_response, method)
        assert signal_response["event"]["type"] == "wallet-owned-collectibles-filtering-done"
        message = json.loads(signal_response["event"]["message"].replace("'", '"'))
        assert user_1.address in message["ownershipStatus"].keys()
