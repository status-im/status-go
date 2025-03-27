from clients.status_backend import StatusBackend
import pytest
from clients.signals import SignalType
import os


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
            backend_client.wait_for_login(),
            "signal_node_login",
        )
        backend_client.verify_json_schema(
            backend_client.wait_for_signal(SignalType.MEDIASERVER_STARTED.value),
            "signal_mediaserver_started",
        )
        backend_client.verify_json_schema(
            backend_client.wait_for_signal(SignalType.NODE_STARTED.value),
            "signal_node_started",
        )
        backend_client.verify_json_schema(
            backend_client.wait_for_signal(SignalType.NODE_READY.value),
            "signal_node_ready",
        )


def assert_file_first_line(path, pattern: str, expected: bool):
    if not expected:
        assert path is None
        return
    assert os.path.exists(path)
    with open(path) as file:
        line = file.readline()
        line_found = line.find(pattern) >= 0
        assert line_found == expected


@pytest.mark.rpc
@pytest.mark.init
@pytest.mark.parametrize("log_enabled,api_logging_enabled", [(True, True), (False, False)])
def test_check_logs(log_enabled: bool, api_logging_enabled: bool):
    backend = StatusBackend()

    data_dir = os.path.join(backend.data_dir, "data")
    logs_dir = os.path.join(backend.data_dir, "logs")

    backend.api_valid_request(
        "InitializeApplication",
        {
            "dataDir": str(data_dir),
            "logDir": str(logs_dir),
            "logEnabled": log_enabled,
            "logLevel": "INFO",
            "apiLoggingEnabled": api_logging_enabled,
        },
    )

    pre_login_log = backend.extract_data(os.path.join(logs_dir, "pre_login.log"))
    local_api_log = backend.extract_data(os.path.join(logs_dir, "api.log"))

    assert_file_first_line(path=pre_login_log, pattern="logging initialised", expected=True)

    assert_file_first_line(
        path=local_api_log,
        pattern='"method": "InitializeApplication"',
        expected=api_logging_enabled,
    )
