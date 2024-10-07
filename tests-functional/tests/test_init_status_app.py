import pytest
import threading
from conftest import option
from clients.status_backend import StatusBackend
from test_cases import StatusBackendTestCase


@pytest.mark.create_account
@pytest.mark.rpc
class TestInitialiseApp(StatusBackendTestCase):

    @pytest.fixture(scope="session", autouse=True)
    def init_status_backend(self):

        await_signals = [

            "mediaserver.started",
            "node.started",
            "node.ready",
            "node.login",

            "wallet",  # TODO: a test per event of a different type
        ]

        self.backend_client = StatusBackend(
            await_signals
        )

        websocket_thread = threading.Thread(
            target=self.backend_client._connect)
        websocket_thread.daemon = True
        websocket_thread.start()

        self.backend_client.init_status_backend()
        self.backend_client.create_account_and_login()

        yield self.backend_client

    @pytest.mark.init
    def test_init_app(self, init_status_backend):
        # this test is going to fail on every call except first since status-backend will be already initialized

        backend_client = init_status_backend
        assert backend_client is not None
        mediaserver_started = backend_client.wait_for_signal(
            "mediaserver.started")

        port = mediaserver_started['event']['port']
        assert type(port) is int, f"Port is not an integer, found {type(port)}"

        backend_client.wait_for_signal("node.started")
        backend_client.wait_for_signal("node.ready")
        backend_client.wait_for_signal("node.login")
