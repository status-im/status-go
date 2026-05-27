import asyncio
import logging
from collections.abc import Callable
from concurrent.futures import ThreadPoolExecutor, as_completed
from uuid import uuid4

import pytest
import pytest_asyncio
from requests import ReadTimeout
from tenacity import retry, stop_after_attempt, wait_fixed
from web3 import Web3

from clients.anvil import Anvil
from clients.async_status_backend import AsyncStatusBackend
from clients.contract_deployers.multicall3 import Multicall3Deployer
from clients.foundry import Foundry
from clients.status_backend import StatusBackend
from resources.constants import (
    USE_IPV6,
    SNT_ADDRESSES_CONTAINER_PATH,
    COMMUNITIES_ADDRESSES_CONTAINER_PATH,
    ENS_ADDRESSES_CONTAINER_PATH,
)
from steps import messenger
from utils import fake
from utils.config import Config

logger = logging.getLogger(__name__)


def _parallel_teardown(tasks: list[tuple[str, Callable]]):
    if not tasks:
        return
    errors: list[Exception] = []
    with ThreadPoolExecutor(max_workers=len(tasks)) as executor:
        futures = {executor.submit(fn): label for label, fn in tasks}
        for future in as_completed(futures):
            label = futures[future]
            try:
                future.result()
            except Exception as e:
                logging.warning(f"[TEARDOWN] {label} failed: {e}")
                errors.append(e)
    if errors:
        msg = "; ".join(f"{type(e).__name__}: {e}" for e in errors)
        raise RuntimeError(f"teardown failures: {msg}") from errors[0]


@retry(stop=stop_after_attempt(30), wait=wait_fixed(2), reraise=True)
def _load_contract_json(foundry_client, path):
    return foundry_client.load_json(path)


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

    # Cleanup all created backends concurrently
    logging.debug(f"🧹 [TEARDOWN] Cleaning up {len(created_backends)} backends for {cls_name or 'test'}")

    tasks = [
        (f"backend-{len(created_backends) - i}", lambda b=backend: b.shutdown(log_sufix=test_name))
        for i, backend in enumerate(reversed(created_backends))
    ]
    _parallel_teardown(tasks)


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

    def _logout(b):
        try:
            b.logout(timeout=10)
        except ReadTimeout as e:
            logging.warning(f"Failed to logout during shutdown: {e}")

    tasks = [(f"logout-{i}", lambda b=backend: _logout(b)) for i, backend in enumerate(backends)]
    _parallel_teardown(tasks)


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

    def _logout(b):
        try:
            b.logout(timeout=10)
        except ReadTimeout as e:
            logging.warning(f"Failed to logout during shutdown: {e}")

    tasks = [(f"logout-{i}", lambda b=backend: _logout(b)) for i, backend in enumerate(backends)]
    _parallel_teardown(tasks)


@pytest.fixture(scope="function", autouse=False)
def funded_new_profile(backend_new_profile, anvil_client):
    """
    Creates a fresh profile with a funded wallet address.

    Returns a factory that produces (backend, wallet_address) tuples.
    """

    def factory(name: str = "", balance: int = 10_000 * 10**18, **kwargs):
        kwargs.setdefault("password", Config.password)
        backend = backend_new_profile(name, **kwargs)

        wallet_address = Web3.to_checksum_address(messenger.wallet_address(backend))

        anvil_client.set_balance(wallet_address, balance)

        return backend, wallet_address

    yield factory


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
    try:
        data = _load_contract_json(foundry_client, SNT_ADDRESSES_CONTAINER_PATH)
        logger.info(f"Using pre-deployed SNT contracts: token={data['snt']}, controller={data['controller']}")
        return data
    except Exception as e:
        logger.error(f"Failed to load SNT addresses from container: {e}")
        logger.error("SNT contracts should be deployed as part of docker-compose startup")
        raise RuntimeError(
            "SNT contracts not found. Make sure the foundry container has deployed contracts during startup. "
            "This should happen automatically in entrypoint.sh"
        ) from e


@pytest.fixture(scope="session")
def communities_addresses(foundry_client):
    try:
        data = _load_contract_json(foundry_client, COMMUNITIES_ADDRESSES_CONTAINER_PATH)
        logger.info("Using pre-deployed Communities contracts")
        return data
    except Exception as e:
        logger.error(f"Failed to load Communities addresses from container: {e}")
        logger.error("Communities contracts should be deployed as part of docker-compose startup")
        raise RuntimeError(
            "Communities contracts not found. Make sure the foundry container has deployed contracts during startup. "
            "This should happen automatically in entrypoint.sh"
        ) from e


@pytest.fixture(scope="session")
def ens_addresses(foundry_client):
    try:
        data = _load_contract_json(foundry_client, ENS_ADDRESSES_CONTAINER_PATH)
        logger.info(f"Using pre-deployed ENS contracts: registry={data['registry']}, " f"registrar={data['registrar']}, token={data['token']}")
        return data
    except Exception as e:
        logger.error(f"Failed to load ENS addresses from container: {e}")
        logger.error("ENS contracts should be deployed as part of docker-compose startup")
        raise RuntimeError(
            "ENS contracts not found. Make sure the foundry container has deployed contracts during startup. "
            "This should happen automatically in entrypoint.sh"
        ) from e


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


@pytest.fixture(scope="session")
def community_token_deployer_contract_address(communities_addresses):
    return next(info["value"] for info in communities_addresses.values() if info.get("internal_type") == "contract CommunityTokenDeployer")


@pytest.fixture(scope="function", autouse=False)
def community_token_snt_context(snt_addresses, request):
    """Opt-in SNT token addresses for community token membership tests."""
    request.cls.snt_address = snt_addresses["snt"]
    request.cls.snt_controller_address = snt_addresses["controller"]


@pytest.fixture(scope="function", autouse=False)
def community_token_deploy_context(snt_addresses, communities_addresses, request):
    """Opt-in deploy context for community token deploy/edit/master tests."""
    request.cls.snt_address = snt_addresses["snt"]
    request.cls.snt_controller_address = snt_addresses["controller"]
    request.cls.community_token_deployer = next(
        info["value"] for info in communities_addresses.values() if info["internal_type"] == "contract CommunityTokenDeployer"
    )
    from steps import community_token_deploy

    request.cls.deploy_state = community_token_deploy.CommunityTokenDeployState()


@pytest.fixture(scope="function", autouse=False)
def owner_backend(backend_new_profile, snt_token_overrides, multicall3_deployer, community_token_deployer_contract_address):
    return backend_new_profile(
        name="owner",
        token_overrides=snt_token_overrides,
        multicall_contract_address=multicall3_deployer.contract_address,
        community_token_deployer_contract_address=community_token_deployer_contract_address,
    )


@pytest.fixture(scope="function", autouse=False)
def member_backend(backend_new_profile, snt_token_overrides, multicall3_deployer, community_token_deployer_contract_address):
    return backend_new_profile(
        name="member",
        token_overrides=snt_token_overrides,
        multicall_contract_address=multicall3_deployer.contract_address,
        community_token_deployer_contract_address=community_token_deployer_contract_address,
    )


@pytest.fixture(scope="function", autouse=False)
def member_with_snt_backend(backend_new_profile, snt_token_overrides, multicall3_deployer, community_token_deployer_contract_address):
    return backend_new_profile(
        name="member_with_snt",
        token_overrides=snt_token_overrides,
        multicall_contract_address=multicall3_deployer.contract_address,
        community_token_deployer_contract_address=community_token_deployer_contract_address,
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

        # Create sync backend in thread pool to avoid blocking event loop
        # Skip sync signal client - we'll use async one instead
        sync_backend = await asyncio.to_thread(lambda: backend_factory(name, skip_signal_client=True, **kwargs))

        # Wrap with async backend for signal support
        async_backend = AsyncStatusBackend(sync_backend)
        await async_backend.start_signal_client()
        async_backends.append(async_backend)

        logging.debug(f"[ASYNC SETUP] {name.capitalize()} bare backend ready with signal client")
        return async_backend

    yield factory

    # Cleanup: stop signal clients in parallel (sync backend cleanup handled by backend_factory)
    if async_backends:
        results = await asyncio.gather(
            *[ab.stop_signal_client() for ab in async_backends],
            return_exceptions=True,
        )
        for i, result in enumerate(results):
            if isinstance(result, Exception):
                logging.warning(f"Failed to stop signal client {i}: {result}")


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

        # Create sync backend in thread pool to avoid blocking event loop
        # NOTE: We do NOT pass skip_signal_client=True here because backend_new_profile
        # calls wait_for_login() which needs the sync signal client to receive NODE_LOGIN.
        # The async wrapper will disconnect sync client later in start_signal_client().
        sync_backend = await asyncio.to_thread(lambda: backend_new_profile(name, waku_light_client=waku_light_client, **kwargs))

        # Wrap with async backend for signal support
        async_backend = AsyncStatusBackend(sync_backend)
        await async_backend.start_signal_client()
        async_backends.append(async_backend)

        logging.debug(f"[ASYNC SETUP] {name.capitalize()} backend ready with signal client")
        return async_backend

    yield factory

    # Cleanup: stop signal clients in parallel (sync backend cleanup handled by backend_new_profile)
    if async_backends:
        results = await asyncio.gather(
            *[ab.stop_signal_client() for ab in async_backends],
            return_exceptions=True,
        )
        for i, result in enumerate(results):
            if isinstance(result, Exception):
                logging.warning(f"Failed to stop signal client {i}: {result}")
