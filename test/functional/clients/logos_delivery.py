"""Client for the logos-delivery store REST API.

The transport that status-go currently runs on (and that backs the functional
test fleet) is historically called "waku". The project term going forward is
"logos delivery", so new code uses that name. The fleet store node exposes the
nwaku REST API (`/store/v3/messages`); this client queries it so tests can
assert that messages were actually persisted by the store node, instead of
sleeping and hoping.
"""

import logging
import time

import docker
import requests
from tenacity import retry, stop_after_attempt, wait_fixed

from utils.config import Config

logger = logging.getLogger(__name__)

# Container port the nwaku store node serves its REST API on (mapped to an
# ephemeral host port by docker-compose.waku.yml).
STORE_REST_CONTAINER_PORT = "8645/tcp"


class LogosDeliveryClient:
    def __init__(self, base_url: str | None = None, timeout: float = 10.0):
        # The store REST port is published on an ephemeral host port to avoid
        # collisions on shared CI hosts, so discover it at runtime via the
        # Docker API (same approach as clients/anvil.py). An explicit base_url
        # can still be passed (e.g. for --status_backend_url style setups).
        self.base_url = (base_url or self._discover_store_url()).rstrip("/")
        self.timeout = timeout

    @retry(stop=stop_after_attempt(15), wait=wait_fixed(0.2), reraise=True)
    def _discover_store_url(self) -> str:
        docker_client = docker.from_env()
        project = Config.docker_project_name
        network = docker_client.networks.get(f"{project}_default")
        prefix = f"{project}-store"
        store = next((c for c in network.containers if c.name and prefix in c.name), None)
        if store is None:
            raise RuntimeError(f"store container ('{prefix}*') not found")
        mappings = store.attrs["NetworkSettings"]["Ports"].get(STORE_REST_CONTAINER_PORT) or []
        if not mappings:
            raise RuntimeError("store REST port is not exposed")
        host_ip = mappings[0]["HostIp"] or "127.0.0.1"
        if host_ip == "0.0.0.0":
            host_ip = "127.0.0.1"
        return f"http://{host_ip}:{mappings[0]['HostPort']}"

    def get_messages(
        self,
        *,
        content_topics: list[str] | None = None,
        pubsub_topic: str | None = None,
        start_time: int | None = None,
        end_time: int | None = None,
        hashes: list[str] | None = None,
        page_size: int = 100,
        ascending: bool = True,
        include_data: bool = True,
    ) -> list[dict]:
        """Return all stored messages matching the filters, following pagination.

        Times are Unix nanoseconds (matching the store node's timestamps).
        """
        params: dict[str, str] = {
            "pageSize": str(page_size),
            "ascending": "true" if ascending else "false",
            "includeData": "true" if include_data else "false",
        }
        if content_topics:
            params["contentTopics"] = ",".join(content_topics)
        if pubsub_topic:
            params["pubsubTopic"] = pubsub_topic
        if start_time is not None:
            params["startTime"] = str(start_time)
        if end_time is not None:
            params["endTime"] = str(end_time)
        if hashes:
            params["hashes"] = ",".join(hashes)

        messages: list[dict] = []
        cursor = None
        while True:
            page_params = dict(params)
            if cursor:
                page_params["cursor"] = cursor
            response = requests.get(f"{self.base_url}/store/v3/messages", params=page_params, timeout=self.timeout)
            response.raise_for_status()
            body = response.json()
            messages.extend(body.get("messages", []) or [])
            cursor = body.get("paginationCursor") or body.get("cursor")
            if not cursor:
                break
        return messages

    def count_messages(self, **kwargs) -> int:
        return len(self.get_messages(**kwargs))

    def get_peers(self) -> list[dict]:
        response = requests.get(f"{self.base_url}/admin/v1/peers", timeout=self.timeout)
        response.raise_for_status()
        return response.json()

    def wait_for_message_count(self, expected_count: int, *, timeout: float = 60.0, poll_interval: float = 2.0, **kwargs) -> list[dict]:
        """Poll the store until at least *expected_count* messages match, or time out."""
        deadline = time.time() + timeout
        messages: list[dict] = []
        while time.time() < deadline:
            messages = self.get_messages(**kwargs)
            if len(messages) >= expected_count:
                return messages
            time.sleep(poll_interval)
        return messages
