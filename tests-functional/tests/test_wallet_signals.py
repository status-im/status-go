import json

import pytest

from clients.signals import SignalType


@pytest.mark.wallet
@pytest.mark.rpc
class TestWalletSignals:

    @pytest.fixture(autouse=True)
    def setup_backend(self, funded_new_profile):
        self.rpc_client, self.wallet_address = funded_new_profile(name="rpc_client")

    @pytest.mark.skip  # TODO: returns empty response in most of the cases, so needs to be fixed with attention of required signals in signal_response
    def test_wallet_get_owned_collectibles_async(self):
        params = [
            0,
            [
                self.rpc_client.network_id,  # type: ignore
            ],
            [self.wallet_address],
            None,
            0,
            25,
            1,
            {"fetch-type": 2, "max-cache-age-seconds": 3600},
        ]
        with self.rpc_client.expect_signal(SignalType.WALLET, timeout=60) as exp:
            self.rpc_client.wallet_service.get_owned_collectibles_async(params)
        signal_response = exp.result
        # TODO: Add more assertions on response
        assert signal_response["event"]["type"] == "wallet-owned-collectibles-filtering-done"
        message = json.loads(signal_response["event"]["message"].replace("'", '"'))
        assert self.wallet_address in message["ownershipStatus"].keys()
