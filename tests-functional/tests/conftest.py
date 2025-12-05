import logging
from uuid import uuid4

import pytest
from requests import ReadTimeout

from clients.anvil import Anvil
from clients.contract_deployers.multicall3 import Multicall3Deployer
from clients.contract_deployers.snt import SNTDeployer
from clients.foundry import Foundry
from clients.status_backend import StatusBackend
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

    # Safe generation of names for logs/teardown for both class and function tests
    _cls_obj = getattr(request, "cls", None)
    cls_name = _cls_obj.__name__ if _cls_obj is not None else None
    node = getattr(request, "node", None)
    test_name = getattr(node, "name", f"test-{uuid4()}")

    def factory(name, **kwargs) -> StatusBackend:
        """
        Create a single backend with the given name.

        Args:
            name (str): Name for the backend (e.g., "sender", "receiver")
            start_messenger (bool): Whether to start messenger service

        Returns:
            StatusBackend: Created backend instance
        """
        logging.debug(f"🔧 [SETUP] Creating {name} backend for {cls_name or test_name}")
        logging.debug(f"📋 [SETUP] Parameters: privileged={privileged}, ipv6={ipv6}")

        # Create backend
        backend = StatusBackend(privileged=privileged, ipv6=ipv6, **kwargs)
        created_backends.append(backend)
        logging.debug(f"✅ [SETUP] {name.capitalize()} backend created")

        return backend

    yield factory

    # Cleanup all created backends
    logging.debug(f"🧹 [TEARDOWN] Cleaning up {len(created_backends)} backends for {cls_name or 'test'}")

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

        backend.init_status_backend()
        backend.create_account_and_login(password=password, waku_light_client=waku_light_client, **kwargs)
        backend.wait_for_login()
        backend.wakuext_service.start_messenger()
        backend.wallet_service.start_wallet()
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

        backend.init_status_backend()
        backend.restore_account_and_login(user=user, waku_light_client=waku_light_client, **kwargs)
        backend.wait_for_login()
        backend.wakuext_service.start_messenger()
        backend.wallet_service.start_wallet()
        return backend

    yield _backend_recovered_profile

    for backend in backends:
        backend.logout()


@pytest.fixture(scope="session")
def anvil_client():
    return Anvil()


@pytest.fixture(scope="session")
def foundry_client():
    return Foundry()


@pytest.fixture(scope="session")
def multicall3_deployer(foundry_client):
    return Multicall3Deployer(foundry_client)


@pytest.fixture(scope="function")
def snt_deployment(foundry_client, request):
    deployer = SNTDeployer(foundry_client)
    request.cls.snt_deployer = deployer
    request.cls.snt_address = deployer.snt_contract_address
    request.cls.snt_controller_address = deployer.snt_token_controller_address
    yield deployer


@pytest.fixture(scope="function", autouse=False)
def owner_backend(backend_new_profile):
    return backend_new_profile("owner")


@pytest.fixture(scope="function", autouse=False)
def member_backend(backend_new_profile):
    return backend_new_profile("member")


@pytest.fixture(scope="function", autouse=False)
def member_with_snt_backend(backend_new_profile):
    return backend_new_profile("member_with_snt")
