import json
from time import sleep
from src.libs.custom_logger import get_custom_logger
import subprocess
import shutil
import os
import uuid
from datetime import datetime
from pathlib import Path

logger = get_custom_logger(__name__)
GO_PROJECT_ROOT = Path(__file__).resolve().parents[3]
SOURCE_DIR = GO_PROJECT_ROOT / "cmd/status-backend"
DEST_DIR = GO_PROJECT_ROOT / "tests-functional"
BINARY_PATH = SOURCE_DIR / "status-backend"
REPORTS_DIR = DEST_DIR / "reports"
REPORTS_DIR.mkdir(parents=True, exist_ok=True)
timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
LOG_FILE_PATH = REPORTS_DIR / f"signals_log_{timestamp}.json"


def delay(num_seconds):
    logger.debug(f"Sleeping for {num_seconds} seconds")
    sleep(num_seconds)


def create_unique_data_dir(base_dir: str, index: int) -> str:
    unique_id = str(uuid.uuid4())[:8]
    unique_dir = os.path.join(base_dir, f"data_{index}_{unique_id}")
    os.makedirs(unique_dir, exist_ok=True)
    return unique_dir


def get_project_root() -> str:
    return os.path.abspath(os.path.join(os.path.dirname(__file__), "../../.."))


def write_signal_to_file(signal_data):
    with open(LOG_FILE_PATH, "a") as file:
        json.dump(signal_data, file)
        file.write("\n")


def build_and_copy_binary():
    logger.info(f"Building status-backend binary in {GO_PROJECT_ROOT}")
    result = subprocess.run(["make", "status-backend"], cwd=GO_PROJECT_ROOT, capture_output=True, text=True)

    if result.returncode != 0:
        logger.info("Build failed with the following output:")
        logger.info(result.stderr)
        return False

    if not os.path.exists(BINARY_PATH):
        logger.info("Binary build failed or not found! Exiting.")
        return False

    logger.info(f"Copying binary to {DEST_DIR}")
    shutil.copy(BINARY_PATH, DEST_DIR)

    if os.path.exists(os.path.join(DEST_DIR, "status-backend")):
        logger.info("Binary successfully copied to tests-functional directory.")
        return True
    else:
        logger.info("Failed to copy binary to the tests-functional directory.")
        return False
