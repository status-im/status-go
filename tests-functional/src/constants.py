import os
import random
from dataclasses import dataclass
import uuid


def create_unique_data_dir(base_dir: str, index: int) -> str:
    unique_id = str(uuid.uuid4())[:8]
    unique_dir = os.path.join(base_dir, f"data_{index}_{unique_id}")
    os.makedirs(unique_dir, exist_ok=True)
    return unique_dir


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
    passphrase="test test test test test test test test test test test junk",
)
user_2 = Account(
    address="0x70997970c51812dc3a010c7d01b50e0d17dc79c8",
    private_key="0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d",
    password="Strong12345",
    passphrase="test test test test test test test test test test nest junk",
)

PROJECT_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
STATUS_BACKEND_URL = os.getenv("STATUS_BACKEND_URL", "http://127.0.0.1")
API_REQUEST_TIMEOUT = int(os.getenv("API_REQUEST_TIMEOUT", "15"))

SOURCE_DIR = os.path.join(PROJECT_ROOT, "build/bin")
DEST_DIR = os.path.join(PROJECT_ROOT, "tests-functional")
BINARY_PATH = os.path.join(SOURCE_DIR, "status-backend")
DATA_DIR = os.path.join(PROJECT_ROOT, "tests-functional/local")
SIGNALS_DIR = os.path.join(DEST_DIR, "signals")
LOCAL_DATA_DIR1 = create_unique_data_dir(DATA_DIR, random.randint(1, 100))
LOCAL_DATA_DIR2 = create_unique_data_dir(DATA_DIR, random.randint(1, 100))
RESOURCES_FOLDER = os.path.join(PROJECT_ROOT, "resources")

ACCOUNT_PAYLOAD_DEFAULTS = {
    "displayName": "user",
    "password": "test_password",
    "customizationColor": "primary",
}

NUM_CONTACT_REQUESTS = int(os.getenv("NUM_CONTACT_REQUESTS", "5"))
NUM_MESSAGES = int(os.getenv("NUM_MESSAGES", "20"))
DELAY_BETWEEN_MESSAGES = int(os.getenv("NUM_MESSAGES", "1"))
EVENT_SIGNAL_TIMEOUT_SEC = int(os.getenv("EVENT_SIGNAL_TIMEOUT_SEC", "5"))
