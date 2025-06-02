import json
import random

import pytest

from resources.constants import user_1
from steps.status_backend import StatusBackendSteps


@pytest.mark.wallet
@pytest.mark.rpc
class TestWalletSignals(StatusBackendSteps):

    @classmethod
    def setup_class(cls, skip_login=False):
        cls.await_signals.append("wallet")
        super().setup_class()

    def setup_method(self):
        self.request_id = str(random.randint(1, 8888))

    @pytest.mark.skip
    def test_wallet_get_owned_collectibles_async(self):
        method = "wallet_getOwnedCollectiblesAsync"
        params = [
            0,
            [
                self.network_id,
            ],
            [user_1.address],
            None,
            0,
            25,
            1,
            {"fetch-type": 2, "max-cache-age-seconds": 3600},
        ]
        self.rpc_client.rpc_valid_request(method, params, self.request_id)
        signal_response = self.rpc_client.wait_for_signal("wallet", timeout=60)
        self.rpc_client.verify_json_schema(signal_response, method)
        assert signal_response["event"]["type"] == "wallet-owned-collectibles-filtering-done"
        message = json.loads(signal_response["event"]["message"].replace("'", '"'))
        assert user_1.address in message["ownershipStatus"].keys()
