from dataclasses import dataclass
import os


@dataclass
class Account:
    address: str
    private_key: str
    password: str
    passphrase: str


user_1 = Account(
    address="0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266",
    private_key="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
    password="Strong12345",
    passphrase="test test test test test test test test test test test junk"
)
user_2 = Account(
    address="0x70997970c51812dc3a010c7d01b50e0d17dc79c8",
    private_key="0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d",
    password="Strong12345",
    passphrase="test test test test test test test test test test nest junk"
)
DEFAULT_DISPLAY_NAME = "Mr_Meeseeks"
PROJECT_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../"))
TESTS_DIR = os.path.join(PROJECT_ROOT, "tests-functional")
SIGNALS_DIR = os.path.join(TESTS_DIR, "signals")
LOG_SIGNALS_TO_FILE = False # used for debugging purposes