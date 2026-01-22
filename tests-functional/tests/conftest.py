import json
import logging
import os
from uuid import uuid4

import pytest
import pytest_asyncio
from filelock import FileLock
from requests import ReadTimeout

from clients.anvil import Anvil
from clients.async_status_backend import AsyncStatusBackend
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

    Recommended usage pattern with explicit fixtures:

    ```python
    import pytest

    class TestMessenger:
        @pytest.fixture()
        def sender(self, backend_factory):
            return backend_factory("sender")

        @pytest.fixture()
        def receiver(self, backend_factory):
            return backend_factory("receiver")

        def test_send_message(self, sender, receiver):
            sender.send_message(receiver, "Hello!")
    ```

    # Or with parameters:
    ```python
    @pytest.mark.parametrize("backend_factory", [{"privileged": True}], indirect=True)
    def test_with_params(self, sender, receiver):
        ...  # use sender/receiver created by parametrized backend_factory
    ```
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

    def factory(name="", **kwargs) -> StatusBackend:
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


@pytest.fixture(scope="session")
def snt_addresses(foundry_client):
    logger = logging.getLogger(__name__)
    addresses_file = os.path.join(os.path.dirname(os.path.dirname(__file__)), "snt_addresses.json")
    lock_path = addresses_file + ".lock"  # Lock file next to JSON

    with FileLock(lock_path, timeout=120):  # Acquire lock (120s timeout for deploy)
        data = None
        if os.path.exists(addresses_file):
            try:
                with open(addresses_file, "r") as f:
                    data = json.load(f)
                # Re-check contracts INSIDE lock (critical for races)
                if foundry_client.check_contract_exists(data["snt"]) and foundry_client.check_contract_exists(data["controller"]):
                    logger.info("Using existing SNT deployment addresses")
                    return data
                else:
                    logger.warning("Existing SNT addresses invalid (no code), will redeploy")
            except (json.JSONDecodeError, KeyError, IOError) as e:
                logger.warning(f"Failed to load existing addresses: {e}, will redeploy")

        # Deploy (now EXCLUSIVE due to lock)
        logger.info("Deploying new SNT token...")
        deployer = SNTDeployer(foundry_client)
        data = {
            "snt": deployer.snt_contract_address,
            "controller": deployer.snt_token_controller_address,
        }
        with open(addresses_file, "w") as f:
            json.dump(data, f, indent=2)
        logger.info(f"New SNT deployed: token={data['snt']}, controller={data['controller']}")
        return data


@pytest.fixture(scope="function")
def snt_token_overrides(snt_addresses):
    return [
        {
            "symbol": "SNT",
            "name": "Status Network Token",
            "address": snt_addresses["snt"],
            "decimals": 18,
        }
    ]


@pytest.fixture(scope="function", autouse=False)
def owner_backend(backend_new_profile, snt_token_overrides, multicall3_deployer):
    return backend_new_profile(name="owner", token_overrides=snt_token_overrides, multicall_contract_address=multicall3_deployer.contract_address)


@pytest.fixture(scope="function", autouse=False)
def member_backend(backend_new_profile, snt_token_overrides, multicall3_deployer):
    return backend_new_profile(name="member", token_overrides=snt_token_overrides, multicall_contract_address=multicall3_deployer.contract_address)


@pytest.fixture(scope="function", autouse=False)
def member_with_snt_backend(backend_new_profile, snt_token_overrides, multicall3_deployer):
    return backend_new_profile(
        name="member_with_snt",
        token_overrides=snt_token_overrides,
        multicall_contract_address=multicall3_deployer.contract_address,
    )


@pytest_asyncio.fixture
async def async_backend_factory(backend_factory):
    """
    Async fixture that creates bare backend (not logged in) with async signal support.

    Useful for tests like local_pairing where receiver device starts without account.

    Usage:
        @pytest.mark.asyncio
        async def test_pairing(async_backend_factory):
            device = await async_backend_factory("device")
            device.backend.init_status_backend()
            # ... pairing flow ...
            signal = await device.wait_for_signal(SignalType.LOCAL_PAIRING, ...)
    """
    async_backends: list[AsyncStatusBackend] = []

    async def factory(name: str = "", **kwargs) -> AsyncStatusBackend:
        logging.debug(f"[ASYNC SETUP] Creating bare {name} backend")

        # Create sync backend (no login, just container)
        sync_backend = backend_factory(name, **kwargs)

        # Wrap with async backend for signal support
        async_backend = AsyncStatusBackend(sync_backend)
        await async_backend.start_signal_client()
        async_backends.append(async_backend)

        logging.debug(f"[ASYNC SETUP] {name.capitalize()} bare backend ready with signal client")
        return async_backend

    yield factory

    # Cleanup: stop signal clients (sync backend cleanup handled by backend_factory)
    for async_backend in async_backends:
        try:
            await async_backend.stop_signal_client()
        except Exception as e:
            logging.warning(f"Failed to stop signal client: {e}")


@pytest_asyncio.fixture
async def async_backend_new_profile(backend_new_profile):
    """
    Async fixture that creates backend with new profile and async signal support.

    Uses sync backend_new_profile for RPC, wraps with AsyncStatusBackend for signals.

    Usage:
        @pytest.mark.asyncio
        async def test_something(async_backend_new_profile):
            backend = await async_backend_new_profile("sender")
            # RPC calls are sync: backend.wakuext_service.send_contact_request(...)
            # Signal waiting is async: await backend.wait_for_signal(...)
    """
    async_backends: list[AsyncStatusBackend] = []

    async def factory(
        name: str = "",
        waku_light_client: bool = False,
        **kwargs,
    ) -> AsyncStatusBackend:
        logging.debug(f"[ASYNC SETUP] Creating {name} with wakuV2LightClient={waku_light_client}")

        # Create sync backend (handles RPC, login, etc.)
        sync_backend = backend_new_profile(name, waku_light_client=waku_light_client, **kwargs)

        # Wrap with async backend for signal support
        async_backend = AsyncStatusBackend(sync_backend)
        await async_backend.start_signal_client()
        async_backends.append(async_backend)

        logging.debug(f"[ASYNC SETUP] {name.capitalize()} backend ready with signal client")
        return async_backend

    yield factory

    # Cleanup: stop signal clients (sync backend cleanup handled by backend_new_profile)
    for async_backend in async_backends:
        try:
            await async_backend.stop_signal_client()
        except Exception as e:
            logging.warning(f"Failed to stop signal client: {e}")
