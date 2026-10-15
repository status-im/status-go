import tempfile
from functools import lru_cache

from filelock import BaseFileLock, FileLock

from utils.config import Config


@lru_cache
def _anvil_lock(project: str) -> BaseFileLock:
    return FileLock(f"{tempfile.gettempdir()}/status-go-anvil-{project}.lock")


def anvil_lock() -> BaseFileLock:
    return _anvil_lock(Config.docker_project_name or "functional")
