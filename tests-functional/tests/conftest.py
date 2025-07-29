import logging
import pytest

from resources.constants import USE_IPV6
from clients.status_backend import StatusBackend
from clients.statusgo_container import StatusGoContainer


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
    @pytest.mark.parametrize("backend_factory", [{"privileged": True}], indirect=True)
    def test_with_params(self, backend_factory):
        self.sender = backend_factory("sender")
        self.receiver = backend_factory("receiver")
    """

    # Get class-level configuration
    await_signals = getattr(request.cls, "await_signals", ["messages.new", "message.delivered", "node.login", "node.logout"])

    # Get parameters from request.param if available
    params = getattr(request, "param", {})

    # Extract parameters with defaults
    privileged = params.get("privileged", False)
    ipv6 = params.get("ipv6", USE_IPV6)

    # Store created backends for cleanup
    created_backends = []

    def factory(name, *, start_messenger=True) -> StatusBackend:
        """
        Create a single backend with the given name.

        Args:
            name (str): Name for the backend (e.g., "sender", "receiver")
            start_messenger (bool): Whether to start messenger service

        Returns:
            StatusBackend: Created backend instance
        """
        logging.debug(f"🔧 [SETUP] Creating {name} backend for {request.cls.__name__}")
        logging.debug(f"📋 [SETUP] Parameters: privileged={privileged}, ipv6={ipv6}")

        # Create backend
        backend = StatusBackend(await_signals=await_signals, privileged=privileged, ipv6=ipv6)
        # backend.init_status_backend()
        # backend.create_account_and_login(waku_light_client=final_light_client)
        # backend.wait_for_login()

        # if start_messenger:
        #     backend.wakuext_service.start_messenger()

        created_backends.append(backend)
        logging.debug(f"✅ [SETUP] {name.capitalize()} backend created")

        return backend

    yield factory

    # Cleanup all created backends
    logging.debug(f"🧹 [TEARDOWN] Cleaning up {len(created_backends)} backends for {request.cls.__name__ if hasattr(request, 'cls') else 'test'}")

    for i, backend in enumerate(reversed(created_backends)):
        logging.debug(f"🧹 [TEARDOWN] Cleaning up backend {len(created_backends) - i}...")

        if hasattr(backend, "container") and backend.container:
            teardown_container(backend.container, log_prefix=f"🧹 [TEARDOWN] Cleaning up backend {len(created_backends) - i} container...")
        else:
            logging.debug(f"ℹ️ [TEARDOWN] Backend {len(created_backends) - i} has no container to cleanup")


@pytest.fixture(scope="function", autouse=False)
def waku_light_client(request) -> bool:
    return getattr(request, "param", False)


@pytest.fixture(scope="function", autouse=False)
def backend_new_profile(request, backend_factory):
    def _backend_new_profile(name: str, waku_light_client: bool = False) -> StatusBackend:
        logging.debug(f"📋 [SETUP] backend_new_profile parameters: wakuV2LightClient={waku_light_client}")
        backend = backend_factory(name)
        backend.init_status_backend()
        backend.create_account_and_login(wakuV2LightClient=waku_light_client)
        backend.wait_for_login()
        backend.wakuext_service.start_messenger()
        return backend

    yield _backend_new_profile
    # backend.logout()


@pytest.fixture(scope="function", autouse=False)
def backend_recovered_profile(request, backend_factory):
    def _backend_recovered_profile(name: str, user: object, waku_light_client: bool = False, **kwargs) -> StatusBackend:
        logging.debug(f"📋 [SETUP] backend_recovered_profile parameters: wakuV2LightClient={waku_light_client}")
        backend = backend_factory(name)
        backend.init_status_backend()
        backend.restore_account_and_login(user=user, wakuV2LightClient=waku_light_client, **kwargs)
        backend.wait_for_login()
        backend.wakuext_service.start_messenger()
        return backend

    yield _backend_recovered_profile
    # backend.logout()


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
    2. After test completes, stops all containers in StatusGoContainer.all_containers
    3. Saves logs from each container for debugging
    4. Removes containers to free up system resources
    5. Clears the containers list

    Usage:
    # Automatic cleanup for all tests in a class
    @pytest.fixture(autouse=True)
    def setup_cleanup(self, close_status_backend_containers):
        yield

    # Manual cleanup for specific test
    def test_something(self, close_status_backend_containers):
        # test code here
        pass

    Parameters:
        request: pytest request object containing test metadata

    Dependencies:
        - StatusGoContainer.all_containers: Global list of active containers
        - Container objects with stop(), save_logs(), remove() methods

    Scope: function (runs once per test function)
    Autouse: False (must be explicitly requested)
    """
    yield
    for container in StatusGoContainer.all_containers:
        try:
            teardown_container(container, log_prefix="[close_status_backend_containers] ")
        except Exception as e:
            logging.error(f"Error cleaning up container: {e}")
    StatusGoContainer.all_containers = []
