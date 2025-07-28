from typing import List, Iterator
from dataclasses import field


class Config:
    status_backend_port_range: List[int] = field(default_factory=list)
    base_dir: str = ""

    status_backend_urls: Iterator[str] | None = None
    password: str = ""  # FIXME: remove
    docker_project_name: str = ""
    docker_image: str = ""
    codecov_dir: str = ""
    logs_dir: str = ""
    logout: bool = False
    waku_fleets_config: str = ""
    waku_fleet: str = ""
    push_fleets_config: str = ""
    disable_override_networks: bool = False
