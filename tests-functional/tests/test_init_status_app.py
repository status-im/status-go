import pytest

from test_cases import StatusBackendTestCase


@pytest.mark.create_account
@pytest.mark.rpc
class TestInitialiseApp(StatusBackendTestCase):

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
