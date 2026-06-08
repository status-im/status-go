from dataclasses import field
from typing import List, Iterator


class Config:
    executor_number: int = 0  # Always overwritten by _calculate_port_range() in conftest.py
    status_backend_port_range: List[int] = field(default_factory=list)
    base_dir: str = ""

    status_backend_urls: Iterator[str] | None = None
    anvil_url: str = ""
    password: str = ""  # FIXME: remove
    docker_project_name: str = ""
    docker_image: str = ""
    peer_docker_images: List[str] = field(default_factory=list)
    codecov_dir: str = ""
    logs_dir: str = ""
    benchmark_results_dir: str = ""
    logout: bool = False
    waku_fleets_config: str = ""
    waku_fleet: str = ""
    push_fleets_config: str = ""
    disable_override_networks: bool = False
