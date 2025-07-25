import os
from dataclasses import dataclass, field
from typing import List, Any
import pytest
import logging
from resources.constants import ANVIL_NETWORK_ID


def pytest_addoption(parser):
    parser.addoption(
        "--status_backend_url",
        action="append",
        help="",
        default=None,
    )
    parser.addoption(
        "--anvil_url",
        action="store",
        help="",
        default="http://0.0.0.0:8545",
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


@dataclass
class Option:
    status_backend_port_range: List[int] = field(default_factory=list)
    statusgo_containers: List[Any] = field(default_factory=list)
    base_dir: str = ""


option = Option()


def status_backend_url_generator(config):
    if hasattr(option, "status_backend_url") and config.option.status_backend_url is not None:
        urls = config.option.status_backend_url
    else:
        print("status_backend_url option not found or is None")
        return

    for url in urls:
        yield url


def pytest_configure(config):
    global option
    option = config.option

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

    option.status_backend_port_range = list(range(start_port, end_port))
    option.statusgo_containers = []

    option.base_dir = os.path.dirname(os.path.abspath(__file__))  # schemas directory
    option.status_backend_urls = status_backend_url_generator(config)


def pytest_report_header(config):
    return [
        f"waku fleets config file: {config.option.waku_fleets_config}",
        f"waku fleet: {config.option.waku_fleet}",
        f"push fleets config file: {config.option.push_fleets_config}",
        f"disable override networks: {config.option.disable_override_networks}",
    ]


def teardown_container(container, log_prefix=""):
    """
    Stops, saves logs, and removes a container with error handling.
    Args:
        container: The container object (should have stop, save_logs, remove methods)
        log_prefix: Optional string for logging context
    """
    if not hasattr(container, "container") or not container.container:
        logging.debug(f"{log_prefix}No container to cleanup.")
        return
    container.stop()  # pyright: ignore[reportAttributeAccessIssue]
    try:
        container.save_logs()  # pyright: ignore[reportAttributeAccessIssue]
    except RuntimeError as e:
        if "Container is not initialized" in str(e):
            logging.warning(f"{log_prefix}Container already stopped, skipping log save: {e}")
        else:
            raise
    container.remove()  # pyright: ignore[reportAttributeAccessIssue]
    logging.debug(f"{log_prefix}Container stopped and removed.")


@pytest.fixture(scope="function", autouse=False)
def close_status_backend_containers(request):
    """
    Fixture to automatically cleanup Status backend containers after each test.
    Should be used ONLY for tests that do not use backend_factory or class_backend fixtures.

    This fixture ensures that all Status backend containers are properly stopped,
    logs are saved, and containers are removed to prevent resource leaks and
    conflicts between tests.

    How it works:
    1. Yields immediately (runs before test execution)
    2. After test completes, checks if container reuse is enabled
    3. If reuse is disabled, stops all containers in option.statusgo_containers
    4. Saves logs from each container for debugging
    5. Removes containers to free up system resources
    6. Clears the containers list

    Usage:
    # Automatic cleanup for all tests in a class
    @pytest.fixture(autouse=True)
    def setup_cleanup(self, close_status_backend_containers):
        yield

    # Manual cleanup for specific test
    def test_something(self, close_status_backend_containers):
        # test code here
        pass

    # Skip cleanup for tests that reuse containers
    class TestReuseContainers:
        reuse_container = True  # This will skip cleanup

    Parameters:
        request: pytest request object containing test metadata

    Dependencies:
        - option.statusgo_containers: Global list of active containers
        - Container objects with stop(), save_logs(), remove() methods

    Scope: function (runs once per test function)
    Autouse: False (must be explicitly requested)
    """
    yield
    if hasattr(request.node.instance, "reuse_container"):
        return
    for container in option.statusgo_containers:
        try:
            teardown_container(container, log_prefix="[close_status_backend_containers] ")
        except Exception as e:
            logging.error(f"Error cleaning up container: {e}")
    option.statusgo_containers = []


@pytest.fixture(scope="function", autouse=False)
def backend_factory(request):
    """
    Individual backend factory that creates backends one by one.
    Each backend is created separately and all are cleaned up at the end.

    Usage:
    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_factory):
        self.sender = backend_factory("sender")
        self.receiver = backend_factory("receiver")

    # Or with parameters:
    @pytest.mark.parametrize("backend_factory", [{"privileged": True, "wakuV2LightClient": True}], indirect=True)
    def test_with_params(self, backend_factory):
        self.sender = backend_factory("sender")
        self.receiver = backend_factory("receiver")
    """
    from clients.status_backend import StatusBackend
    from resources.constants import USE_IPV6

    # Get class-level configuration
    await_signals = getattr(request.cls, "await_signals", ["messages.new", "message.delivered", "node.login", "node.logout"])

    # Get parameters from request.param if available
    params = getattr(request, "param", {})

    # Extract parameters with defaults
    privileged = params.get("privileged", False)
    ipv6 = params.get("ipv6", USE_IPV6)
    wakuV2LightClient = params.get("wakuV2LightClient", False)
    light_client_mode = params.get("light_client_mode", False)

    # Use light_client_mode if specified, otherwise wakuV2LightClient
    final_light_client = light_client_mode if "light_client_mode" in params else wakuV2LightClient

    # Store created backends for cleanup
    created_backends = []

    def create_backend(self, name, *, start_messenger=True):
        """
        Create a single backend with the given name.

        Args:
            name (str): Name for the backend (e.g., "sender", "receiver")
            start_messenger (bool): Whether to start messenger service

        Returns:
            StatusBackend: Created backend instance
        """
        logging.debug(f"🔧 [SETUP] Creating {name} backend for {request.cls.__name__}")
        logging.debug(f"📋 [SETUP] Parameters: privileged={privileged}, ipv6={ipv6}, wakuV2LightClient={final_light_client}")

        # Create backend
        backend = StatusBackend(await_signals=await_signals, privileged=privileged, ipv6=ipv6)
        backend.init_status_backend()
        backend.create_account_and_login(wakuV2LightClient=final_light_client)
        backend.wait_for_login()

        if start_messenger:
            backend.wakuext_service.start_messenger()

        created_backends.append(backend)
        logging.debug(f"✅ [SETUP] {name.capitalize()} backend created")

        return backend

    # Create factory object with create_backend method
    factory = type("BackendFactory", (), {"__call__": create_backend})()

    yield factory

    # Cleanup all created backends
    logging.debug(f"🧹 [TEARDOWN] Cleaning up {len(created_backends)} backends for {request.cls.__name__ if hasattr(request, 'cls') else 'test'}")

    for i, backend in enumerate(reversed(created_backends)):
        logging.debug(f"🧹 [TEARDOWN] Cleaning up backend {len(created_backends) - i}...")

        if hasattr(backend, "container") and backend.container:
            teardown_container(backend.container, log_prefix=f"🧹 [TEARDOWN] Cleaning up backend {len(created_backends) - i} container...")
        else:
            logging.debug(f"ℹ️ [TEARDOWN] Backend {len(created_backends) - i} has no container to cleanup")


@pytest.fixture(scope="class", autouse=False)
def backend_factory_class(request):
    """
    Class-scoped backend factory that allows creating and reusing multiple backends per class.
    Each backend is created only once per class and reused in all tests.
    Always sets reuse_container = True on the test class.

    Usage:
        @pytest.fixture(autouse=True)
        def setup_backends(self, backend_factory_class):
            self.sender = backend_factory_class(name="sender", user=user_1)
            self.receiver = backend_factory_class(name="receiver", user=user_2)
    """
    request.cls.reuse_container = True
    request.cls.network_id = ANVIL_NETWORK_ID

    from clients.status_backend import StatusBackend
    from resources.constants import USE_IPV6, user_1

    await_signals = getattr(request.cls, "await_signals", ["node.login"])
    params = getattr(request, "param", {})

    privileged = params.get("privileged", False)
    ipv6 = params.get("ipv6", USE_IPV6)
    wakuV2LightClient = params.get("wakuV2LightClient", False)
    light_client_mode = params.get("light_client_mode", False)
    final_light_client = light_client_mode if "light_client_mode" in params else wakuV2LightClient

    # Dictionary to store created backends by name
    created_backends = {}

    def recover_backend(*, name, user=user_1, start_messenger=True, skip_login=False):
        if name not in created_backends:
            backend = StatusBackend(await_signals=await_signals, privileged=privileged, ipv6=ipv6)
            backend.init_status_backend()
            if not skip_login:
                backend.restore_account_and_login(user=user, wakuV2LightClient=final_light_client)
                backend.wait_for_login()
            if start_messenger:
                backend.wakuext_service.start_messenger()
            created_backends[name] = backend
        return created_backends[name]

    yield recover_backend

    # Cleanup all created backends
    for backend in created_backends.values():
        if hasattr(backend, "container") and backend.container:
            teardown_container(backend.container, log_prefix="[TEARDOWN] ")


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
