import json
import random

import pytest

from constants import user_1
from test_cases import SignalBackendTestCase


@pytest.mark.wallet
@pytest.mark.rpc
class TestWalletRpcSignal(SignalBackendTestCase):
    await_signals = ["wallet", ]

    def setup_method(self):
        super().setup_method()
        self.request_id = str(random.randint(1, 8888))

    def test_wallet_get_owned_collectibles_async(self):
        method = "wallet_getOwnedCollectiblesAsync"
        params = [0, [self.network_id, ], [user_1.address], None, 0, 25, 1,
                  {"fetch-type": 2, "max-cache-age-seconds": 3600}]
        self.rpc_client.rpc_valid_request(method, params, self.request_id)
        signal_response = self.rpc_client.wait_for_signal("wallet", timeout=60)
        self.rpc_client.verify_json_schema(signal_response, method)
        assert signal_response['event']['type'] == "wallet-owned-collectibles-filtering-done"
        message = json.loads(signal_response['event']['message'].replace("'", "\""))
        assert user_1.address in message['ownershipStatus'].keys()

    def test_wallet_filter_activity_async(self):
        method = "wallet_filterActivityAsync"
        params = [1, [user_1.address], [self.network_id],
                  {"period": {"startTimestamp": 0, "endTimestamp": 0}, "types": [], "statuses": [],
                   "counterpartyAddresses": [], "assets": [], "collectibles": [], "filterOutAssets": False,
                   "filterOutCollectibles": False}, 0, 50]
        self.rpc_client.rpc_valid_request(method, params, self.request_id)
        signal_response = self.rpc_client.wait_for_signal("wallet", timeout=60)
        self.rpc_client.verify_json_schema(signal_response, method)
        assert signal_response['event']['type'] == "wallet-activity-filtering-done"
        message = json.loads(signal_response['event']['message'].replace("'", "\""))
        for item in message['activities']:
            assert user_1.address in item['sender'], item['recipient']
