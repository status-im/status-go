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

import requests

logger = logging.getLogger(__name__)

# Store node REST API, published to the host by docker-compose.waku.yml.
DEFAULT_STORE_URL = "http://127.0.0.1:8646"


class LogosDeliveryClient:
    def __init__(self, base_url: str = DEFAULT_STORE_URL, timeout: float = 10.0):
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout

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
