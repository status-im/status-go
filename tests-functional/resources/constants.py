from dataclasses import dataclass
from conftest import option
import os


@dataclass
class Account:
    address: str
    private_key: str
    password: str
    passphrase: str


user_1 = Account(
    address="0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
    private_key="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
    password="Strong12345",
    passphrase="test test test test test test test test test test test junk",
)
user_2 = Account(
    address="0x70997970C51812dc3A010C7d01b50e0d17dc79C8",
    private_key="0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d",
    password="Strong12345",
    passphrase="test test test test test test test test test test nest junk",
)
DEFAULT_DISPLAY_NAME = "Mr_Meeseeks"
PROJECT_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../"))
TESTS_DIR = os.path.join(PROJECT_ROOT, "tests-functional")
SIGNALS_DIR = os.path.join(TESTS_DIR, "signals")
FORGE_OUTPUT_DIR = os.path.join(PROJECT_ROOT, "forge_output")
DEPLOYER_ACCOUNT = user_1
LOG_SIGNALS_TO_FILE = False  # used for debugging purposes
USE_IPV6 = os.getenv("USE_IPV6", "No")
USER_DIR = option.user_dir if option.user_dir else "/usr/status-user"

gas_fee_mode_low = 0
gas_fee_mode_medium = 1
gas_fee_mode_high = 2
gas_fee_mode_custom = 3

processor_name_transfer = "Transfer"

ANVIL_NETWORK_ID = 31337
