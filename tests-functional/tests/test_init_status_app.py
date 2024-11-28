from test_cases import StatusBackend
import pytest
import os


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


@pytest.mark.rpc
@pytest.mark.skip("waiting for status-backend to be executed on the same host/container")
class TestInitializeLogging:

    @pytest.mark.init
    def test_init_logging(self, tmp_path):
        self.check_logs(tmp_path, log_enabled=True, api_logging_enabled=True)

    @pytest.mark.init
    def test_no_logging(self, tmp_path):
        self.check_logs(tmp_path, log_enabled=False, api_logging_enabled=False)


    def assert_file_first_line(self, path, pattern: str, expected: bool):
        assert os.path.exists(path) == expected
        if not expected:
            return
        with open(path) as file:
            line = file.readline()
            line_found = line.find(pattern) >= 0
            assert line_found == expected

    def check_logs(self, path, log_enabled: bool, api_logging_enabled: bool):
        data_dir = path / "data"
        logs_dir = path / "logs"

        data_dir.mkdir()
        logs_dir.mkdir()

        backend = StatusBackend()
        backend.api_valid_request("InitializeApplication", {
            "dataDir": str(data_dir),
            "logDir": str(logs_dir),
            "logEnabled": log_enabled,
            "apiLoggingEnabled": api_logging_enabled,
        })

        self.assert_file_first_line(
            logs_dir / "geth.log",
            pattern="logging initialised",
            expected=log_enabled)

        self.assert_file_first_line(
            logs_dir / "api.log",
            pattern='"method": "InitializeApplication"',
            expected=api_logging_enabled)

