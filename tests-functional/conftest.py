import os
import logging
from typing import Iterator
from utils.config import Config


def pytest_addoption(parser):
    parser.addoption(
        "--status_backend_url",
        action="append",
        help="",
        default=None,
    )
    parser.addoption(
        "--password",
        action="store",
        help="",
        default="Strong12345",
    )
    parser.addoption(
        "--docker_project_name",
        action="store",
        help="",
        default="tests-functional",
    )
    parser.addoption(
        "--docker-image",
        action="store",
        help="status-go docker image name to use, defaults to current git commit",
        default="",
    )
    parser.addoption(
        "--codecov_dir",
        action="store",
        help="",
        default=None,
    )
    parser.addoption(
        "--logs-dir",
        action="store",
        help="Path to a directory where containers logs will be saved",
        default="",
    )
    parser.addoption(
        "--benchmark-results-dir",
        action="store",
        help="Path to a directory where benchmarks results will be saved",
        default="./.results/benchmarks/",
    )
    parser.addoption(
        "--logout",
        action="store_true",
        help="When set, will automatically call Logout() before InitializeApplication()",
        default=False,
    )
    parser.addoption(
        "--waku-fleets-config",
        action="store",
        help="Path to a local JSON file with Waku fleets configuration. Default value is a path to config in Docker to run 2 local waku nodes",
        default="/static/configs/wakufleetconfig.json",
    )
    parser.addoption(
        "--waku-fleet",
        action="store",
        help="Waku fleet to be used. Default: --waku-fleet=status-go.test",
        default="status-go.test",
    )
    parser.addoption(
        "--push-fleets-config",
        action="store",
        help="Path to a local JSON file with Push Notifications fleets configuration. Default value is a path to config in Docker to run 1 pn-server",
        default="/static/configs/pushfleetconfig.json",
    )
    parser.addoption(
        "--disable-override-networks",
        action="store_true",
        help="When set, will disable overriding the networks to use Anvil and use default status-backend networks",
        default=False,
    )


def _status_backend_url_generator(urls) -> Iterator[str]:
    for url in urls:
        yield url


def _calculate_port_range():
    executor_number = int(os.getenv("EXECUTOR_NUMBER", 5))
    base_port = 7000
    range_size = 100
    max_port = 65535
    min_port = 1024

    start_port = base_port + (executor_number * range_size)
    end_port = start_port + 20000

    # Ensure generated ports are within the valid range
    if start_port < min_port or end_port > max_port:
        raise ValueError(f"Generated port range ({start_port}-{end_port}) is outside the allowed range ({min_port}-{max_port}).")

    return list(range(start_port, end_port))


def pytest_configure(config):
    status_backend_urls = config.getoption("--status_backend_url")
    Config.status_backend_urls = _status_backend_url_generator(status_backend_urls) if status_backend_urls else None
    Config.password = config.getoption("--password")
    Config.docker_project_name = config.getoption("--docker_project_name")
    Config.docker_image = config.getoption("--docker-image")
    Config.codecov_dir = config.getoption("--codecov_dir")
    Config.logs_dir = config.getoption("--logs-dir")
    Config.benchmark_results_dir = config.getoption("--benchmark-results-dir")
    Config.logout = config.getoption("--logout")
    Config.waku_fleets_config = config.getoption("--waku-fleets-config")
    Config.waku_fleet = config.getoption("--waku-fleet")
    Config.push_fleets_config = config.getoption("--push-fleets-config")
    Config.disable_override_networks = config.getoption("--disable-override-networks")
    Config.status_backend_port_range = _calculate_port_range()
    Config.base_dir = os.path.dirname(os.path.abspath(__file__))  # schemas directory


def pytest_report_header(config):
    return [
        f"docker image: {Config.docker_image}",
        f"waku fleets config file: {config.option.waku_fleets_config}",
        f"waku fleet: {config.option.waku_fleet}",
        f"push fleets config file: {config.option.push_fleets_config}",
        f"disable override networks: {config.option.disable_override_networks}",
    ]


class SecretRedactingFilter(logging.Filter):
    secrets = []
    placeholder = "***"

    def __init__(self):
        env_vars = [
            "STATUS_BUILD_PROXY_USER",
            "STATUS_BUILD_PROXY_PASSWORD",
            "STATUS_BUILD_INFURA_TOKEN",
            "STATUS_BUILD_INFURA_SECRET",
            "STATUS_BUILD_POKT_TOKEN",
        ]
        for env_var in env_vars:
            if env_var in os.environ:
                self.secrets.append(os.environ[env_var])
        super().__init__()

    def redact(self, message):
        for secret in self.secrets:
            if secret:
                message = message.replace(secret, self.placeholder)
        return message

    def filter(self, record):
        # Redact secrets in the log message
        if isinstance(record.msg, str):
            message = record.getMessage()
            record.msg = self.redact(message)

        # Also redact secrets in args (if used with parameterized logging)
        if record.args:
            new_args = []
            for arg in record.args:
                redacted_arg = arg
                if isinstance(arg, str):
                    redacted_arg = self.redact(arg)
                new_args.append(redacted_arg)
            record.args = tuple(new_args)

        return True


logging.basicConfig(level=logging.DEBUG)
logger = logging.getLogger()
logger.addFilter(SecretRedactingFilter())

logging.getLogger("websocket").setLevel(logging.ERROR)
