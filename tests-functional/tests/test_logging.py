import re
import time
from test_cases import StatusBackend
import pytest
import os

@pytest.mark.rpc
@pytest.mark.skip("waiting for status-backend to be executed on the same host/container")
class TestLogging:

    @pytest.mark.init
    def test_logging(self, tmp_path):
        await_signals = [
            "mediaserver.started",
            "node.started",
            "node.ready",
            "node.login",
        ]

        backend_client = StatusBackend(await_signals)
        assert backend_client is not None

        # Init and login
        backend_client.init_status_backend(data_dir=str(tmp_path))
        backend_client.create_account_and_login(data_dir=str(tmp_path))
        key_uid = self.ensure_logged_in(backend_client)

        # Configure logging
        backend_client.rpc_valid_request("wakuext_setLogLevel", [{"logLevel": "ERROR"}])
        backend_client.rpc_valid_request("wakuext_setLogNamespaces", [{"logNamespaces": "test1.test2:debug,test1.test2.test3:info"}])

        # Re-login (logging settings take effect after re-login)
        backend_client.logout()
        backend_client.login(str(key_uid))
        self.ensure_logged_in(backend_client)

        # Test logging
        backend_client.rpc_valid_request("wakuext_logTest")
        self.expect_logs(tmp_path / "geth.log", "test message", [
            r"DEBUG\s+test1\.test2",
            r"INFO\s+test1\.test2",
            r"INFO\s+test1\.test2\.test3",
            r"WARN\s+test1\.test2",
            r"WARN\s+test1\.test2\.test3",
            r"ERROR\s+test1",
            r"ERROR\s+test1\.test2",
            r"ERROR\s+test1\.test2\.test3",
        ])

    def expect_logs(self, log_file, filter_keyword, expected_logs):
        with open(log_file, 'r') as f:
            log_content = f.read()

        filtered_logs = [line for line in log_content.splitlines() if filter_keyword in line]
        for expected_log in expected_logs:
            assert any(re.search(expected_log, log) for log in filtered_logs), f"Log entry not found: {expected_log}"

    def ensure_logged_in(self, backend_client):
        login_response = backend_client.wait_for_signal("node.login")
        backend_client.verify_json_schema(login_response, "signal_node_login")
        key_uid = login_response.get("event", {}).get("account", {}).get("key-uid")
        assert key_uid is not None, "key-uid not found in login response"
        return key_uid
