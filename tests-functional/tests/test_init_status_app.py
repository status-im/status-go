from test_cases import StatusBackend
import pytest
from clients.signals import SignalType

@pytest.mark.create_account
@pytest.mark.rpc
class TestInitialiseApp:

    @pytest.mark.init
    def test_init_app(self):

        await_signals = [

            SignalType.MEDIASERVER_STARTED.value,
            SignalType.NODE_STARTED.value,
            SignalType.NODE_READY.value,
            SignalType.NODE_LOGIN.value,
        ]

        backend_client = StatusBackend(await_signals)
        backend_client.init_status_backend()
        backend_client.restore_account_and_login()

        assert backend_client is not None
        
        backend_client.verify_json_schema(
            backend_client.wait_for_signal(SignalType.MEDIASERVER_STARTED.value), "signal_mediaserver_started")
        backend_client.verify_json_schema(
            backend_client.wait_for_signal(SignalType.NODE_STARTED.value), "signal_node_started")
        backend_client.verify_json_schema(
            backend_client.wait_for_signal(SignalType.NODE_READY.value), "signal_node_ready")
        backend_client.verify_json_schema(
            backend_client.wait_for_signal(SignalType.NODE_LOGIN.value), "signal_node_login")
