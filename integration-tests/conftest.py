import os
from dataclasses import dataclass


def pytest_addoption(parser):
    parser.addoption(
        "--rpc_url",
        action="store",
        help="",
        default="http://0.0.0.0:3333",
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

user_1 = Account(
    address="0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
    private_key="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
)
user_2 = Account(
    address="0x70997970C51812dc3A010C7d01b50e0d17dc79C8",
    private_key="0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d",
)

@dataclass
class Option:
    pass

option = Option()

def pytest_configure(config):
    global option
    option = config.option
    option.base_dir = os.path.dirname(os.path.abspath(__file__))
