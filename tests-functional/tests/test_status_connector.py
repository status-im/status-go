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

    def test_get_chain_id_resolved_before_connection(self, backend, connector):
        connector.get_chain_id()

        message = connector.receive()
        chain_id = int(message.get("result"), 16)
        assert chain_id == backend.network_id

    def test_other_methods_not_resolved_before_connection(self, connector):
        connector.get_block_number()

        message = connector.receive()
        assert message.get("error").get("message") == "dApp is not permitted by user"

    def test_connect_and_get_accounts(self, backend, connector):
        # Load actual directly from backend
        wallet_acc = self._get_wallet_account(backend)
        assert wallet_acc is not None

        # Request accounts through connector
        connector.get_accounts()

        # Expect SEND_REQUEST_ACCOUNTS signal, check dApp name
        event = self._get_event(backend, SignalType.CONNECTOR_SEND_REQUEST_ACCOUNTS.value)
        assert event.get("name") == connector.name
        request_id = event.get("requestId")

        # Accept request
        response = backend.connector_service.request_accounts_accepted(request_id, wallet_acc, backend.network_id)
        assert response.get("error") is None

        # Expect DAPP_PERMISSION_GRANTED signal, check dApp name, shared account and chain ID
        event = self._get_event(backend, SignalType.CONNECTOR_DAPP_PERMISSION_GRANTED.value)
        assert event.get("name") == connector.name
        assert event.get("sharedAccount") == wallet_acc
        assert len(event.get("chains")) == 1 and event.get("chains")[0] == backend.network_id

        # Receive accounts on connector
        message = connector.receive()
        assert message.get("result") is not None
        accounts = message.get("result")
        assert len(accounts) == 1
        assert accounts[0] == wallet_acc

    def test_other_methods_resolved_after_connection(self, backend, connector):
        # Request accounts through connector and connect
        wallet_acc = self._get_wallet_account(backend)
        connector.get_accounts()
        request_id = self._get_event(backend, SignalType.CONNECTOR_SEND_REQUEST_ACCOUNTS.value).get("requestId")
        backend.connector_service.request_accounts_accepted(request_id, wallet_acc, backend.network_id)
        message = connector.receive()
        assert message.get("result") is not None

        # Other methods are resolved after connection
        connector.get_block_number()

        message = connector.receive()
        assert message.get("result") is not None

    def test_revoke_permissions_on_disconnect(self, backend, connector):
        # Request accounts through connector and connect
        wallet_acc = self._get_wallet_account(backend)
        connector.get_accounts()
        request_id = self._get_event(backend, SignalType.CONNECTOR_SEND_REQUEST_ACCOUNTS.value).get("requestId")
        backend.connector_service.request_accounts_accepted(request_id, wallet_acc, backend.network_id)
        message = connector.receive()
        assert message.get("result") is not None

        # Handle Revoke Permissions before disconnect
        connector.revoke_permissions()

        message = connector.receive()
        assert message.get("error") is None

        # Expect DAPP_PERMISSION_REVOKED signal, check dApp name
        event = self._get_event(backend, SignalType.CONNECTOR_DAPP_PERMISSION_REVOKED.value)
        assert event.get("name") == connector.name

    def _get_wallet_account(self, backend):
        accs = backend.accounts_service.get_accounts()["result"]
        wallet_acc = next((acc.get("address") for acc in accs if acc.get("wallet") is True), None)
        return wallet_acc

    def _get_event(self, backend, signal_type):
        signal = backend.wait_for_signal(signal_type)
        event = signal.get("event")
        return event
