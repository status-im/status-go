import logging
from typing import Generator, Callable

import pytest

from resources.constants import USE_IPV6
from clients.status_backend import StatusBackend
from clients.statusgo_container import StatusGoContainer


@pytest.fixture(scope="function", autouse=False)
def backend_factory(request) -> Generator[Callable[[str], StatusBackend]]:
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

    def factory(name, *, start_messenger=True):
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
def backend_new_profile(request, backend_factory) -> Generator[Callable[[str, bool], StatusBackend]]:
    def _backend_new_profile(name: str, waku_light_client: bool) -> StatusBackend:
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
def backend_recovered_profile(request, backend_factory) -> Generator[Callable[[str, object, bool], StatusBackend]]:
    def _backend_recovered_profile(name: str, user: object, waku_light_client: bool = False) -> StatusBackend:
        logging.debug(f"📋 [SETUP] backend_recovered_profile parameters: wakuV2LightClient={waku_light_client}")
        backend = backend_factory(name)
        backend.init_status_backend()
        backend.restore_account_and_login(user=user, wakuV2LightClient=waku_light_client)
        backend.wait_for_login()
        backend.wakuext_service.start_messenger()
        return backend

    yield _backend_recovered_profile
    # backend.logout()


# @pytest.fixture(scope="class", autouse=False)
# def backend_factory_class(request):
#     """
#     Class-scoped backend factory that allows creating and reusing multiple backends per class.
#     Each backend is created only once per class and reused in all tests.
#     Always sets reuse_container = True on the test class.
#
#     Usage:
#         @pytest.fixture(autouse=True)
#         def setup_backends(self, backend_factory_class):
#             self.sender = backend_factory_class(name="sender", user=user_1)
#             self.receiver = backend_factory_class(name="receiver", user=user_2)
#     """
#     request.cls.reuse_container = True
#     request.cls.network_id = ANVIL_NETWORK_ID
#
#     from clients.status_backend import StatusBackend
#     from resources.constants import USE_IPV6, user_1
#
#     await_signals = getattr(request.cls, "await_signals", ["node.login"])
#     params = getattr(request, "param", {})
#
#     privileged = params.get("privileged", False)
#     ipv6 = params.get("ipv6", USE_IPV6)
#
#     # Dictionary to store created backends by name
#     created_backends = {}
#
#     def recover_backend(*, name, user=user_1, start_messenger=True, skip_login=False):
#         if name not in created_backends:
#             backend = StatusBackend(await_signals=await_signals, privileged=privileged, ipv6=ipv6)
#             backend.init_status_backend()
#             if not skip_login:
#                 backend.restore_account_and_login(user=user, wakuV2LightClient=final_light_client)
#                 backend.wait_for_login()
#             if start_messenger:
#                 backend.wakuext_service.start_messenger()
#             created_backends[name] = backend
#         return created_backends[name]
#
#     yield recover_backend
#
#     # Cleanup all created backends
#     for backend in created_backends.values():
#         if hasattr(backend, "container") and backend.container:
#             teardown_container(backend.container, log_prefix="[TEARDOWN] ")


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
    3. If reuse is disabled, stops all containers in StatusGoContainer.all_containers
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
        - StatusGoContainer.all_containers: Global list of active containers
        - Container objects with stop(), save_logs(), remove() methods

    Scope: function (runs once per test function)
    Autouse: False (must be explicitly requested)
    """
    yield
    if hasattr(request.node.instance, "reuse_container"):
        return
    for container in StatusGoContainer.all_containers:
        try:
            teardown_container(container, log_prefix="[close_status_backend_containers] ")
        except Exception as e:
            logging.error(f"Error cleaning up container: {e}")
    StatusGoContainer.all_containers = []
