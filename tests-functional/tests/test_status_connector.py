import pytest

from clients.connector import ConnectorClient
from clients.signals import SignalType


def wait_event(backend, signal_type):
    signal = backend.wait_for_signal(signal_type)
    return signal.get("event")


def accept_connector(backend, connector, wallet_acc):
    # Expect SEND_REQUEST_ACCOUNTS signal, check dApp name
    event = wait_event(backend, SignalType.CONNECTOR_SEND_REQUEST_ACCOUNTS.value)
    assert event.get("name") == connector.name

    # Accept request
    response = backend.connector_service.request_accounts_accepted(event.get("requestId"), wallet_acc, backend.network_id)
    assert response.get("error") is None

    # Expect DAPP_PERMISSION_GRANTED signal, check dApp name, shared account and chain ID
    event = wait_event(backend, SignalType.CONNECTOR_DAPP_PERMISSION_GRANTED.value)
    assert event.get("name") == connector.name
    assert event.get("sharedAccount") == wallet_acc
    assert len(event.get("chains")) == 1 and event.get("chains")[0] == backend.network_id


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

    @pytest.fixture
    def wallet_account(self, backend):
        # Load actual wallet account directly from backend
        accs = backend.accounts_service.get_accounts()["result"]
        wallet_acc = next((acc.get("address") for acc in accs if acc.get("wallet") is True), None)
        assert wallet_acc is not None
        return wallet_acc

    def test_chain_id_resolved_before_connection(self, backend, connector):
        connector.chain_id()

        message = connector.receive()
        chain_id = int(message.get("result"), 16)
        assert chain_id == backend.network_id

    def test_other_methods_not_resolved_before_connection(self, connector):
        connector.block_number()

        message = connector.receive()
        assert message.get("error").get("message") == "dApp is not permitted by user"

    def test_connect_and_accounts(self, backend, connector, wallet_account):
        # Request accounts through connector
        # And accept connection request
        connector.accounts()
        accept_connector(backend, connector, wallet_account)

        # Receive accounts on connector
        message = connector.receive()
        assert message.get("result") is not None
        accounts = message.get("result")
        assert len(accounts) == 1
        assert accounts[0] == wallet_account

    def test_block_number_resolved_after_connection(self, backend, connector, wallet_account):
        # Request accounts through connector
        # And accept connection request
        connector.accounts()
        accept_connector(backend, connector, wallet_account)
        message = connector.receive()
        assert message.get("result") is not None

        # Block Number is resolved after connection
        connector.block_number()

        message = connector.receive()
        assert message.get("result") is not None

    def test_revoke_permissions_on_disconnect(self, backend, connector, wallet_account):
        # Request accounts through connector
        # And accept connection request
        connector.accounts()
        accept_connector(backend, connector, wallet_account)
        message = connector.receive()
        assert message.get("result") is not None

        # Handle Revoke Permissions before disconnect
        connector.revoke_permissions()

        message = connector.receive()
        assert message.get("error") is None

        # Expect DAPP_PERMISSION_REVOKED signal, check dApp name
        event = wait_event(backend, SignalType.CONNECTOR_DAPP_PERMISSION_REVOKED.value)
        assert event.get("name") == connector.name
