import json
import pytest

from clients.connector import ConnectorClient
from clients.signals import SignalType


@pytest.mark.rpc
@pytest.mark.connector
class TestStatusConnector:

    await_signals = [
        SignalType.NODE_LOGIN.value,
        SignalType.CONNECTOR_SEND_REQUEST_ACCOUNTS.value,
        SignalType.CONNECTOR_SEND_TRANSACTION.value,
        SignalType.CONNECTOR_SIGN.value,
        SignalType.CONNECTOR_DAPP_PERMISSION_GRANTED.value,
        SignalType.CONNECTOR_DAPP_PERMISSION_REVOKED.value,
        SignalType.CONNECTOR_DAPP_CHAIN_ID_SWITCHED.value,
    ]

    @pytest.fixture
    def backend(self, backend_new_profile):
        yield backend_new_profile("sender", connector_enabled=True)

    @pytest.fixture
    def connector(self, backend):
        client = ConnectorClient(backend.connector_ws_url)
        client.connect()
        yield client
        client.disconnect()

    def test_get_accounts(self, backend, connector):
        # Load actual directly from backend
        accs = backend.accounts_service.get_accounts()["result"]
        wallet_acc = next((acc.get("address") for acc in accs if acc.get("wallet") is True), None)
        assert wallet_acc is not None

        # Request accounts through connector
        connector.get_accounts()

        # Expect SEND_REQUEST_ACCOUNTS signal, check dApp name
        signal = backend.wait_for_signal(SignalType.CONNECTOR_SEND_REQUEST_ACCOUNTS.value)
        event = signal.get("event")
        assert event.get("name") == connector.name

        # Accept request
        response = backend.connector_service.request_accounts_accepted(event.get("requestId"), wallet_acc, backend.network_id)
        assert response.get("error") is None

        # Expect DAPP_PERMISSION_GRANTED signal, check dApp name, shared account and chain ID
        signal = backend.wait_for_signal(SignalType.CONNECTOR_DAPP_PERMISSION_GRANTED.value)
        event = signal.get("event")
        assert event.get("name") == connector.name
        assert event.get("sharedAccount") == wallet_acc
        assert len(event.get("chains")) == 1 and event.get("chains")[0] == backend.network_id

        # Receive accounts on connector
        message = connector.receive()
        message = json.loads(message)
        assert message.get("result") is not None
        accounts = message.get("result")
        assert len(accounts) == 1
        assert accounts[0] == wallet_acc
