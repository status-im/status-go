import pytest

from clients.connector import ConnectorApiError, ConnectorClient
from test_status_connector import accept_connector

DAPP_BROWSER_CLIENT_ID = "status-desktop/dapp-browser"


@pytest.mark.rpc
@pytest.mark.browser
@pytest.mark.connector
class TestBrowserConnector:

    @pytest.fixture
    def backend(self, backend_new_profile):
        yield backend_new_profile("browser-connector", connector_enabled=True)

    @pytest.fixture
    def connector(self, backend):
        client = ConnectorClient(backend.connector_ws_url)
        client.connect()
        yield client
        client.disconnect()

    @pytest.fixture
    def wallet_account(self, backend):
        accs = backend.accounts_service.get_accounts()
        wallet_acc = next((acc.get("address") for acc in accs if acc.get("wallet") is True), None)
        assert wallet_acc is not None
        return wallet_acc

    def test_untrusted_websocket_rejects_spoofed_dapp_browser_client_id(self, backend, connector):
        connector.eth_chain_id_with_client_id(DAPP_BROWSER_CLIENT_ID)
        with pytest.raises(ConnectorApiError, match="cannot override clientId"):
            connector.receive()

    def test_external_connector_session_uses_empty_client_id(self, backend, connector, wallet_account):
        assert backend.connector_service.get_permitted_dapps_list() == []

        accept_connector(backend, connector, wallet_account)
        connector.receive()

        dapps = backend.connector_service.get_permitted_dapps_list()
        assert len(dapps) == 1
        assert dapps[0].get("url", "").rstrip("/") == "http://localhost"
        assert dapps[0].get("clientId", dapps[0].get("clientID")) in ("", None)

    def test_wallet_get_capabilities_via_connector_after_connect(self, backend, connector, wallet_account):
        accept_connector(backend, connector, wallet_account)
        connector.receive()

        connector._send("wallet_getCapabilities")
        message = connector.receive()
        assert message.get("error") is None
        result = message.get("result")
        assert isinstance(result, dict)
