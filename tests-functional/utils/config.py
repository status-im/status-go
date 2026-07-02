from typing import List, Iterator


class Config:
    base_dir: str = ""

    status_backend_urls: Iterator[str] | None = None
    anvil_url: str = ""
    password: str = ""  # FIXME: remove
    docker_project_name: str = ""
    docker_image: str = ""
    peer_docker_images: List[str] | None = None
    codecov_dir: str = ""
    logs_dir: str = ""
    benchmark_results_dir: str = ""
    logout: bool = False
    waku_fleets_config: str = ""
    waku_fleet: str = ""
    push_fleets_config: str = ""
    disable_override_networks: bool = False
