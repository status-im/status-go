from test_cases import StatusBackend
import pytest


@pytest.mark.create_account
@pytest.mark.rpc
class TestInitialiseApp:

    @pytest.mark.init
    def test_init_app(self):

        await_signals = [

            "mediaserver.started",
            "node.started",
            "node.ready",
            "node.login",
        ]

        backend_client = StatusBackend(await_signals)
        backend_client.init_status_backend()
        backend_client.restore_account_and_login()

        assert backend_client is not None
        
        backend_client.verify_json_schema(
            backend_client.wait_for_signal("mediaserver.started"), "signal_mediaserver_started")
        backend_client.verify_json_schema(
            backend_client.wait_for_signal("node.started"), "signal_node_started")
        backend_client.verify_json_schema(
            backend_client.wait_for_signal("node.ready"), "signal_node_ready")
        backend_client.verify_json_schema(
            backend_client.wait_for_signal("node.login"), "signal_node_login")
