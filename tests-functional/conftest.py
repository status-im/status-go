import inspect
import os
import threading
from dataclasses import dataclass
import pytest as pytest
from src.libs.custom_logger import get_custom_logger

logger = get_custom_logger(__name__)


def pytest_addoption(parser):
    parser.addoption(
        "--rpc_url_statusd",
        action="store",
        help="",
        default="http://0.0.0.0:3333",
    )
    parser.addoption(
        "--rpc_url_status_backend",
        action="store",
        help="",
        default="http://0.0.0.0:3334",
    )
    parser.addoption(
        "--ws_url_statusd",
        action="store",
        help="",
        default="ws://0.0.0.0:8354",
    )
    parser.addoption(
        "--ws_url_status_backend",
        action="store",
        help="",
        default="ws://0.0.0.0:3334",
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


@dataclass
class Option:
    pass


option = Option()


def pytest_configure(config):
    global option
    option = config.option
    option.base_dir = os.path.dirname(os.path.abspath(__file__))


@pytest.fixture(scope="session", autouse=False)
def init_status_backend():
    logger.info(f"Running fixture setup: {inspect.currentframe().f_code.co_name}")

    await_signals = [
        "mediaserver.started",
        "node.started",
        "node.ready",
        "node.login",
        "wallet",  # TODO: a test per event of a different type
    ]

    from src.node.clients.status_backend import StatusBackend

    backend_client = StatusBackend(await_signals=await_signals)

    websocket_thread = threading.Thread(target=backend_client._connect)
    websocket_thread.daemon = True
    websocket_thread.start()

    backend_client.init_status_backend(data_dir="/")
    backend_client.restore_account_and_wait_for_rpc_client_to_start()

    yield backend_client
