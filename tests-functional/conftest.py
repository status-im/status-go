import os
import docker

from dataclasses import dataclass, field
from typing import List


def pytest_addoption(parser):
    parser.addoption(
        "--status_backend_url",
        action="store",
        help="",
        default=None,
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
    parser.addoption(
        "--docker_project_name",
        action="store",
        help="",
        default="tests-functional",
    )
    parser.addoption(
        "--codecov_dir",
        action="store",
        help="",
        default=None,
    )
    parser.addoption(
        "--user_dir",
        action="store",
        help="",
        default=None,
    )


@dataclass
class Option:
    status_backend_port_range: List[int] = field(default_factory=list)
    status_backend_containers: List[str] = field(default_factory=list)
    base_dir: str = ""


option = Option()


def pytest_configure(config):
    global option
    option = config.option

    executor_number = int(os.getenv("EXECUTOR_NUMBER", 5))
    base_port = 7000
    range_size = 100

    start_port = base_port + (executor_number * range_size)

    option.status_backend_port_range = list(range(start_port, start_port + range_size - 1))
    option.status_backend_containers = []

    option.base_dir = os.path.dirname(os.path.abspath(__file__))


def pytest_unconfigure():
    docker_client = docker.from_env()
    for container_id in option.status_backend_containers:
        try:
            container = docker_client.containers.get(container_id)
            container.stop(timeout=30)
            container.remove()
        except Exception as e:
            print(e)
