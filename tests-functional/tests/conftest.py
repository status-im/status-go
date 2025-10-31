import logging
from uuid import uuid4

import pytest
from requests import ReadTimeout

from clients.anvil import Anvil
from clients.contract_deployers.multicall3 import Multicall3Deployer
from clients.foundry import Foundry
from clients.status_backend import StatusBackend
from clients.statusgo_container import StatusGoContainer
from resources.constants import USE_IPV6
from utils import fake


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

    # Get parameters from request.param if available
    params = getattr(request, "param", {})

    # Extract parameters with defaults
    privileged = params.get("privileged", False)
    ipv6 = params.get("ipv6", USE_IPV6)

    # Store created backends for cleanup
    created_backends: list[StatusBackend] = []

    test_name = request.node.name if hasattr(request, "cls") else str(uuid4)

    def factory(name, **kwargs) -> StatusBackend:
        """
        Create a single backend with the given name.

        Args:
            name (str): Name for the backend (e.g., "sender", "receiver")
            start_messenger (bool): Whether to start messenger service

        Returns:
            StatusBackend: Created backend instance
        """
        logging.debug(f"🔧 [SETUP] Creating {name} backend for {request.cls.__name__}")
        logging.debug(f"🔧 [SETUP] Creating {name} backend for {test_name}")
        logging.debug(f"📋 [SETUP] Parameters: privileged={privileged}, ipv6={ipv6}")

        # Create backend
        backend = StatusBackend(privileged=privileged, ipv6=ipv6, **kwargs)
        created_backends.append(backend)
        logging.debug(f"✅ [SETUP] {name.capitalize()} backend created")

        return backend

    yield factory

    # Cleanup all created backends
    logging.debug(f"🧹 [TEARDOWN] Cleaning up {len(created_backends)} backends for {request.cls.__name__ if hasattr(request, 'cls') else 'test'}")

    for i, backend in enumerate(reversed(created_backends)):
        logging.debug(f"🧹 [TEARDOWN] Cleaning up backend {len(created_backends) - i}...")
        backend.shutdown(log_sufix=test_name)


@pytest.fixture(scope="function", autouse=False)
def waku_light_client(request) -> bool:
    return getattr(request, "param", False)


@pytest.fixture(scope="function", autouse=False)
def backend_new_profile(request, backend_factory):
    backends: list[StatusBackend] = []

    def factory(name: str = "", waku_light_client: bool = False, **kwargs) -> StatusBackend:
        password = kwargs.pop("password", fake.profile_password())

        logging.debug(f"📋 [SETUP] backend_new_profile parameters: wakuV2LightClient={waku_light_client}")
        backend = backend_factory(name, **kwargs)
        backends.append(backend)

        backend.initialize()
        backend.create_account_and_login(password=password, waku_light_client=waku_light_client, **kwargs)
        backend.wait_for_login()
        backend.wakuext_service.start_messenger()
        return backend

    yield factory

    for backend in backends:
        try:
            backend.logout(timeout=10)
        except ReadTimeout as e:
            logging.warning(f"Failed to logout during shutdown: {e}")


@pytest.fixture(scope="function", autouse=False)
def backend_recovered_profile(request, backend_factory):
    backends: list[StatusBackend] = []

    def _backend_recovered_profile(name: str, user: object, waku_light_client: bool = False, **kwargs) -> StatusBackend:
        logging.debug(f"📋 [SETUP] backend_recovered_profile parameters: wakuV2LightClient={waku_light_client}")
        backend = backend_factory(name, **kwargs)
        backends.append(backend)

        backend.initialize()
        backend.restore_account_and_login(user=user, waku_light_client=waku_light_client, **kwargs)
        backend.wait_for_login()
        backend.wakuext_service.start_messenger()
        return backend

    yield _backend_recovered_profile

    for backend in backends:
        backend.logout()


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
            container.shutdown(log_sufix=request.node.name)  # pyright: ignore[reportAttributeAccessIssue]
        except Exception as e:
            logging.error(f"Error cleaning up container: {e}")
    StatusGoContainer.all_containers = []


@pytest.fixture(scope="session")
def anvil_client():
    return Anvil()


@pytest.fixture(scope="session")
def foundry_client():
    return Foundry()


@pytest.fixture(scope="session")
def multicall3_deployer(foundry_client):
    return Multicall3Deployer(foundry_client)
