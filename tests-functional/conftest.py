import os
import threading
from dataclasses import dataclass

import pytest as pytest


def pytest_addoption(parser):
    parser.addoption(
        "--rpc_url_statusd",
        action="store",
        help="",
        default="http://0.0.0.0:3333",
    )
    parser.addoption(
        "--ws_url_statusd",
        action="store",
        help="",
        default="ws://0.0.0.0:8354",
    )
    parser.addoption(
        "--status_backend_urls",
        action="store",
        help="",
        default=[
            f"http://0.0.0.0:{3314 + i}" for i in range(
                int(os.getenv("STATUS_BACKEND_COUNT", 10))
            )
        ],
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
    if type(option.status_backend_urls) is str:
        option.status_backend_urls = option.status_backend_urls.split(",")
    option.base_dir = os.path.dirname(os.path.abspath(__file__))
