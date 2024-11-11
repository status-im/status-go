import json
from time import sleep
from src.libs.custom_logger import get_custom_logger
import subprocess
import shutil
import os
from datetime import datetime
from src.constants import PROJECT_ROOT, BINARY_PATH, DEST_DIR, SIGNALS_DIR
from pathlib import Path


logger = get_custom_logger(__name__)
Path(SIGNALS_DIR).mkdir(parents=True, exist_ok=True)


def delay(num_seconds):
    logger.debug(f"Sleeping for {num_seconds} seconds")
    sleep(num_seconds)


def write_signal_to_file(signal_data):
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    signal_file_path = os.path.join(SIGNALS_DIR, f"signals_log_{timestamp}.json")
    with open(signal_file_path, "a+") as file:
        json.dump(signal_data, file)
        file.write("\n")


def build_and_copy_binary():
    logger.info(f"Building status-backend binary in {PROJECT_ROOT}")
    result = subprocess.run(["make", "status-backend"], cwd=PROJECT_ROOT, capture_output=True, text=True)

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
