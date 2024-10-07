import os
from dataclasses import dataclass


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
class Account():
    
    address: str
    private_key: str
    password: str

user_1 = Account(
    address="0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266",
    private_key="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
    password="Strong12345"
)
user_2 = Account(
    address="0x70997970c51812dc3a010c7d01b50e0d17dc79c8",
    private_key="0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d",
    password="Strong12345"
)

@dataclass
class Option:
    pass

option = Option()

def pytest_configure(config):
    global option
    option = config.option
    option.base_dir = os.path.dirname(os.path.abspath(__file__))
