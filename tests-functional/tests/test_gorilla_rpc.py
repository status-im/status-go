import logging

import pytest

import utils.fake as fake
from clients.signals import SignalType


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

    def test_gorilla_rpc(self, backend_new_profile):
        backend = backend_new_profile("sender")

        testvalue = fake.profile_name()
        args = {"a": testvalue}

        request = {"jsonrpc": "2.0", "method": "eth.TestGorilla", "id": 1, "params": [args]}
        reply = backend.api_request_json("CallGorillaRPC", request)
        result = reply.get("result")

        assert result is not None
        assert result.get("b") == ("a: " + testvalue)

        request = {"jsonrpc": "2.0", "method": "eth.TestPanic", "id": 1, "params": [args]}
        reply = backend.api_request_json("CallGorillaRPC", request)
        logging.debug(f"<<< reply {reply}")
