import logging
import os

import pytest

from resources.constants import user_1


@pytest.mark.create_account
@pytest.mark.rpc
class TestInitialiseApp:

    @pytest.mark.init
    def test_init_app(self, backend_recovered_profile):
        backend_client = backend_recovered_profile("init_app", user=user_1)

        assert backend_client is not None


def assert_file_first_line(path, pattern: str, expected: bool):
    if not expected:
        assert path is None
        return
    assert os.path.exists(path)
    with open(path) as file:
        line = file.readline()
        logging.info(f"Checking pattern '{pattern}': {line}")
        line_found = line.find(pattern) >= 0
        assert line_found == expected, f"""Expected {pattern} to be {"found" if expected else "not found"} as first line in {path}"""


@pytest.mark.rpc
@pytest.mark.init
@pytest.mark.parametrize("log_enabled,api_logging_enabled", [(False, False), (True, True)])
def test_check_logs(log_enabled: bool, api_logging_enabled: bool, backend_factory):
    backend = backend_factory("prelogin")

    data_dir = os.path.join(backend.data_dir, "data")
    logs_dir = os.path.join(backend.data_dir, "logs")

    backend.api_request_json(
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

    assert_file_first_line(path=pre_login_log, pattern="logging initialised", expected=log_enabled)

    assert_file_first_line(
        path=local_api_log,
        pattern='"type": "mediaserver.started"',
        expected=api_logging_enabled,
    )
