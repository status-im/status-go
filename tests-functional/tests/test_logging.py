from resources.constants import USER_DIR
import re
from clients.status_backend import StatusBackend
import pytest
import os


@pytest.mark.rpc
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
        backend_client.init_status_backend()
        backend_client.create_account_and_login()

        # Configure logging
        backend_client.api_valid_request("SetLogLevel", {"logLevel": "ERROR"})
        backend_client.api_valid_request("SetLogNamespaces", {"logNamespaces": "test1.test2:debug,test1.test2.test3:info"})

        log_pattern = [
            r"DEBUG\s+test1\.test2\s+",
            r"INFO\s+test1\.test2\s+",
            r"INFO\s+test1\.test2\.test3\s+",
            r"WARN\s+test1\.test2\s+",
            r"WARN\s+test1\.test2\.test3\s+",
            r"ERROR\s+test1\s+",
            r"ERROR\s+test1\.test2\s+",
            r"ERROR\s+test1\.test2\.test3\s+",
        ]

        # Ensure changes take effect at runtime
        backend_client.rpc_valid_request("wakuext_logTest")
        geth_log = backend_client.extract_data(os.path.join(USER_DIR, "geth.log"))
        self.expect_logs(geth_log, "test message", log_pattern, count=1)

        # Disable logging
        backend_client.api_valid_request("SetLogEnabled", {"enabled": False})
        backend_client.rpc_valid_request("wakuext_logTest")
        geth_log = backend_client.extract_data(os.path.join(USER_DIR, "geth.log"))
        self.expect_logs(geth_log, "test message", log_pattern, count=1)

        # Enable logging
        backend_client.api_valid_request("SetLogEnabled", {"enabled": True})
        backend_client.rpc_valid_request("wakuext_logTest")
        geth_log = backend_client.extract_data(os.path.join(USER_DIR, "geth.log"))
        self.expect_logs(geth_log, "test message", log_pattern, count=2)

        # Ensure changes are persisted after re-login
        backend_client.logout()
        backend_client.login(str(backend_client.find_key_uid()))
        backend_client.wait_for_login()
        backend_client.rpc_valid_request("wakuext_logTest")
        geth_log = backend_client.extract_data(os.path.join(USER_DIR, "geth.log"))
        self.expect_logs(geth_log, "test message", log_pattern, count=3)

    def expect_logs(self, log_file, filter_keyword, expected_logs, count):
        with open(log_file, "r") as f:
            log_content = f.read()

        filtered_logs = [line for line in log_content.splitlines() if filter_keyword in line]
        for expected_log in expected_logs:
            assert sum(1 for log in filtered_logs if re.search(expected_log, log)) == count, f"Log entry not found or count mismatch: {expected_log}"
