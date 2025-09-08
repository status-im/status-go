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
    backend.connector_service.request_accounts_accepted(event.get("requestId"), wallet_acc, backend.network_id)

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
        accs = backend.accounts_service.get_accounts()
        wallet_acc = next((acc.get("address") for acc in accs if acc.get("wallet") is True), None)
        assert wallet_acc is not None
        return wallet_acc

    def test_chain_id_resolved_before_connection(self, backend, connector):
        connector.eth_chain_id()

        message = connector.receive()
        assert message.get("result") == hex(backend.network_id)

    def test_connect_and_accounts(self, backend, connector, wallet_account):
        # Request accounts through connector
        connector.eth_accounts()
        # And accept connection request
        accept_connector(backend, connector, wallet_account)

        # Receive accounts on connector
        message = connector.receive()
        assert message.get("result") is not None
        accounts = message.get("result")
        assert len(accounts) == 1
        assert accounts[0] == wallet_account

    def test_connect_and_request_accounts(self, backend, connector, wallet_account):
        # Request accounts through connector using eth_requestAccounts
        connector.eth_request_accounts()
        # Accept connection request
        accept_connector(backend, connector, wallet_account)

        # Receive accounts on connector
        message = connector.receive()
        assert message.get("result") is not None
        accounts = message.get("result")
        assert len(accounts) == 1
        assert accounts[0] == wallet_account

    def test_block_number(self, backend, connector, wallet_account):
        # Block Number not resolved before connection
        connector.eth_block_number()
        message = connector.receive()
        assert message.get("error").get("message") == "dApp is not permitted by user"

        # Establish connection
        connector.eth_accounts()
        accept_connector(backend, connector, wallet_account)
        message = connector.receive()
        assert message.get("result") is not None

        # Block Number is resolved after connection
        connector.eth_block_number()

        message = connector.receive()
        result = message.get("result")
        assert int(result, 16) >= 0

    def test_get_balance(self, backend, connector, wallet_account):
        # Get balance not resolved before connection
        connector.eth_get_balance(wallet_account)
        message = connector.receive()
        assert message.get("error").get("message") == "dApp is not permitted by user"

        # Establish connection
        connector.eth_accounts()
        accept_connector(backend, connector, wallet_account)
        message = connector.receive()
        assert message.get("result") is not None

        # Get balance is resolved after connection
        connector.eth_get_balance(wallet_account)
        message = connector.receive()
        result = message.get("result")
        assert result.startswith("0x")
        assert int(result, 16) >= 0

    def test_get_transaction_count(self, backend, connector, wallet_account):
        # Get transaction count not resolved before connection
        connector.eth_get_transaction_count(wallet_account)
        message = connector.receive()
        assert message.get("error").get("message") == "dApp is not permitted by user"

        # Establish connection
        connector.eth_accounts()
        accept_connector(backend, connector, wallet_account)
        message = connector.receive()
        assert message.get("result") is not None

        # Get transaction count is resolved after connection
        connector.eth_get_transaction_count(wallet_account)
        message = connector.receive()
        result = message.get("result")
        assert result.startswith("0x")
        assert int(result, 16) >= 0

    def test_eth_call(self, backend, connector, wallet_account):
        # eth_call not resolved before connection
        call_object = {"to": "0x0000000000000000000000000000000000000000", "data": "0x"}
        connector.eth_call(call_object)
        message = connector.receive()
        assert message.get("error").get("message") == "dApp is not permitted by user"

        # Establish connection
        connector.eth_accounts()
        accept_connector(backend, connector, wallet_account)
        message = connector.receive()
        assert message.get("result") is not None

        # eth_call is resolved after connection
        connector.eth_call(call_object)
        message = connector.receive()
        result = message.get("result")
        assert result == "0x"

    def test_estimate_gas_resolved(self, backend, connector, wallet_account):
        # Estimate gas not resolved before connection
        # Simple 0-value transfer to self
        tx = {"from": wallet_account, "to": wallet_account, "value": "0x0"}
        connector.eth_estimate_gas(tx)
        message = connector.receive()
        assert message.get("error").get("message") == "dApp is not permitted by user"

        # Establish connection
        connector.eth_accounts()
        accept_connector(backend, connector, wallet_account)
        message = connector.receive()
        assert message.get("result") is not None

        # Estimate gas is resolved after connection
        connector.eth_estimate_gas(tx)
        message = connector.receive()
        result = message.get("result")
        assert result.startswith("0x")
        assert int(result, 16) == 21000  # minimum gas limit

    def test_get_transaction_receipt(self, backend, connector, wallet_account):
        # Get transaction receipt not resolved before connection
        fake_tx = "0x" + "0" * 64
        connector.eth_get_transaction_receipt(fake_tx)
        message = connector.receive()
        assert message.get("error").get("message") == "dApp is not permitted by user"

        # Establish connection
        connector.eth_accounts()
        accept_connector(backend, connector, wallet_account)
        message = connector.receive()
        assert message.get("result") is not None

        # Query a non-existent tx receipt should return null result
        connector.eth_get_transaction_receipt(fake_tx)
        message = connector.receive()
        assert message.get("error") is None

    def test_send_transaction_flow(self, backend, connector, wallet_account):
        # Send transaction not resolved before connection
        tx = {
            "from": wallet_account,
            "to": wallet_account,
            "value": "0x0",
        }
        connector.eth_send_transaction(tx)
        message = connector.receive()
        assert message.get("error").get("message") == "dApp is not permitted by user"

        # Establish connection
        connector.eth_accounts()
        accept_connector(backend, connector, wallet_account)
        message = connector.receive()
        assert message.get("result") is not None

        # Initiate send transaction
        connector.eth_send_transaction(tx)

        # Wait for signal and approve with a dummy tx hash
        event = wait_event(backend, SignalType.CONNECTOR_SEND_TRANSACTION.value)
        request_id = event.get("requestId")
        fake_hash = "0x" + "1" * 64
        backend.connector_service.send_transaction_accepted(request_id, fake_hash)

        # Connector should now receive the tx hash as result
        message = connector.receive()
        assert message.get("error") is None
        assert message.get("result") == fake_hash

    def test_switch_ethereum_chain(self, backend, connector, wallet_account):
        # Switch chain not resolved before connection
        connector.wallet_switch_ethereum_chain(backend.network_id)
        message = connector.receive()
        assert message.get("error").get("message") == "dApp is not permitted by user"

        # Establish connection
        connector.eth_accounts()
        accept_connector(backend, connector, wallet_account)
        message = connector.receive()
        assert message.get("result") is not None

        # Switch to the same chain ID
        connector.wallet_switch_ethereum_chain(backend.network_id)
        message = connector.receive()
        assert message.get("error") is None

        # Expect CHAIN_ID_SWITCHED signal, verify chain ID
        event = wait_event(backend, SignalType.CONNECTOR_DAPP_CHAIN_ID_SWITCHED.value)
        assert event.get("chainId") == hex(backend.network_id)

    def test_revoke_permissions_on_disconnect(self, backend, connector, wallet_account):
        # Request accounts through connector
        # And accept connection request
        connector.eth_accounts()
        accept_connector(backend, connector, wallet_account)
        message = connector.receive()
        assert message.get("result") is not None

        # Handle Revoke Permissions before disconnect
        connector.wallet_revoke_permissions()

        message = connector.receive()
        assert message.get("error") is None

        # Expect DAPP_PERMISSION_REVOKED signal, check dApp name
        event = wait_event(backend, SignalType.CONNECTOR_DAPP_PERMISSION_REVOKED.value)
        assert event.get("name") == connector.name
