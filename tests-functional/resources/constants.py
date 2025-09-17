# Main constants file for tests
from dataclasses import dataclass
import os
from typing import Optional, List, Dict, Any
from resources.test_data import mnemonic_12, mnemonic_15, mnemonic_24


@dataclass
class Account:
    address: str
    private_key: str
    password: str
    passphrase: str
    accounts: Optional[List[Dict[str, Any]]] = None  # Optional list of accounts
    profile_data: Optional[Dict[str, Any]] = None  # Optional profile data


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

user_mnemonic_12 = Account(
    address="0xC43f4Ab94eC965a3EE9815C5Df07383057d261A8",
    private_key="",
    password="Strong12345",
    passphrase="exhibit soldier miracle series edge atom daring alter absorb decide orphan addict",
    accounts=mnemonic_12.accounts,
    profile_data=mnemonic_12.profile_data,
)

user_mnemonic_15 = Account(
    address="0x685d7ec8e08769ca7020a6b65709887e38e68e6d",
    private_key="",
    password="Strong12345",
    passphrase="category two chapter fame hunt horse huge rotate inner monkey affair champion mixed tail final",
    accounts=mnemonic_15.accounts,
    profile_data=mnemonic_15.profile_data,
)

user_mnemonic_24 = Account(
    address="0xf2d58ae5aa880f7c3f65d769296b1061c61e0955",
    private_key="",
    password="Strong12345",
    passphrase=(
        "border cabbage grape stage return enable bamboo main only voyage glad race patient stool drum sort "
        "army abandon elegant grit cinnamon endless rail drink"
    ),
    accounts=mnemonic_24.accounts,
    profile_data=mnemonic_24.profile_data,
)

new_account_data_1 = {
    "address": "0x1234567890abcdef1234567890abcdef12345678",
    "key-uid": "",
    "wallet": False,
    "chat": False,
    "type": "generated",
    "path": "m/44'/60'/0'/0/0",
    "public-key": "0xabcdef",
    "name": "account1",
    "emoji": "🔑",
    "colorId": "blue",
}

new_account_data_2 = {
    "address": "0xf2d58ae5aa880f7c3f65d769296b1061c61e0955",
    "key-uid": "",
    "wallet": False,
    "chat": False,
    "type": "generated",
    "path": "m/44'/60'/0'/0/1",
    "public-key": "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
    "name": "account2",
    "emoji": "🔑",
    "colorId": "blue",
}

user_keycard_1 = {
    "keyUID": "5a0dd657-165a-4810-b800-6005452be42f",
    "address": "0x1234567890abcdef1234567890abcdef12345678",
    "whisperPrivateKey": "example-whisper-private-key",
    "whisperPublicKey": "example-whisper-public-key",
    "whisperAddress": "example-whisper-address",
    "walletPublicKey": "example-wallet-public-key",
    "walletAddress": "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
    "walletRootAddress": "0xrootaddressrootaddressrootaddressrootaddr",
    "eip1581Address": "0xeip1581address1234567890abcdef1234567890",
    "encryptionPublicKey": "example-encryption-public-key",
}

keycard_1 = {
    "keycard-uid": "kc-0xab1948",
    "keycard-name": "TestKeycard-0xab19",
    "accounts-addresses": ["0x5e98dbb30871a33f802a710420bf975095c1645c"],
    "key-uid": "",
}

keypair_name = "ImportedKeypairName"

wallet_account_details_root = {
    "name": keypair_name,
    "path": "m",
    "emoji": "🔑",
    "colorId": "primary",
}

wallet_account_details_derivation = {
    "name": keypair_name,
    "path": "m/44'/60'/0'/0/0",
    "emoji": "🔑",
    "colorId": "primary",
}


PROJECT_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../"))
TESTS_DIR = os.path.join(PROJECT_ROOT, "tests-functional")
SIGNALS_DIR = os.path.join(TESTS_DIR, "signals")
FORGE_OUTPUT_DIR = os.path.join(PROJECT_ROOT, "forge_output")
DEPLOYER_ACCOUNT = user_1
LOG_SIGNALS_TO_FILE = False  # used for debugging purposes
USE_IPV6 = os.getenv("USE_IPV6", "No")

gas_fee_mode_low = 0
gas_fee_mode_medium = 1
gas_fee_mode_high = 2
gas_fee_mode_custom = 3

processor_name_transfer = "Transfer"

ANVIL_NETWORK_ID = 31337

STATUS_CONNECTOR_WS_PORT = 8586
STATUS_MEDIA_SERVER_PORT = 8587
