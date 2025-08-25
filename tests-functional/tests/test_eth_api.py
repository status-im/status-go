import pytest

from clients.signals import SignalType
from resources.constants import user_1, user_2


@pytest.mark.rpc
@pytest.mark.ethclient
@pytest.mark.xdist_group(name="Eth")
class TestEth:
    await_signals = [
        SignalType.NODE_LOGIN.value,
        SignalType.WALLET.value,
        SignalType.WALLET_SUGGESTED_ROUTES.value,
        SignalType.WALLET_ROUTER_SIGN_TRANSACTIONS.value,
        SignalType.WALLET_ROUTER_SENDING_TRANSACTIONS_STARTED.value,
        SignalType.WALLET_ROUTER_TRANSACTIONS_SENT.value,
    ]

    def test_estimate_gas(self, backend_recovered_profile):
        backend = backend_recovered_profile("sender", user=user_1)
        response = backend.rpc_valid_request("eth_estimateGas", params=[31337, {"to": user_2.address, "value": 100}]).json()
        assert response.get("error") is None
        assert response.get("result") is not None
        assert int(response.get("result"), 16) > 0
