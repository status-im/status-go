import os
from typing import List, Iterator


def _calculate_port_range():
    executor_number = int(os.getenv("EXECUTOR_NUMBER", 5))
    base_port = 7000
    range_size = 100
    max_port = 65535
    min_port = 1024

    start_port = base_port + (executor_number * range_size)
    end_port = start_port + 20000

    # Ensure generated ports are within the valid range
    if start_port < min_port or end_port > max_port:
        raise ValueError(f"Generated port range ({start_port}-{end_port}) is outside the allowed range ({min_port}-{max_port}).")

    return list(range(start_port, end_port))


class Config:
    status_backend_port_range: List[int] = _calculate_port_range()
    base_dir: str = ""

    status_backend_urls: Iterator[str] | None = None
    password: str = ""  # FIXME: remove
    docker_project_name: str = ""
    docker_image: str = ""
    codecov_dir: str = ""
    logs_dir: str = ""
    benchmark_results_dir: str = ""
    logout: bool = False
    waku_fleets_config: str = ""
    waku_fleet: str = ""
    push_fleets_config: str = ""
    disable_override_networks: bool = False
