from time import sleep
from src.libs.custom_logger import get_custom_logger
import os
import allure
import uuid

logger = get_custom_logger(__name__)


def attach_allure_file(file):
    logger.debug(f"Attaching file {file}")
    allure.attach.file(file, name=os.path.basename(file), attachment_type=allure.attachment_type.TEXT)


def delay(num_seconds):
    logger.debug(f"Sleeping for {num_seconds} seconds")
    sleep(num_seconds)

def create_unique_data_dir(base_dir: str, index: int) -> str:
    """Generate a unique data directory for each node instance."""
    unique_id = str(uuid.uuid4())[:8]
    unique_dir = os.path.join(base_dir, f"data_{index}_{unique_id}")
    os.makedirs(unique_dir, exist_ok=True)
    return unique_dir

def get_project_root() -> str:
    """Returns the root directory of the project."""
    return os.path.abspath(os.path.join(os.path.dirname(__file__), "../../.."))
