import json
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import parse_qs, urlparse

import pytest

from steps import messenger
from utils import fake

VITALIK = "0x983110309620D911731Ac0932219af06091b6744"
SECOND = "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"
SEARCH_HIT = "0x1111111111111111111111111111111111111111"
PAGE_HIT = "0x2222222222222222222222222222222222222222"

FOLLOWING_BODY = {
    "following": [
        {
            "version": 1,
            "record_type": "address",
            "data": VITALIK,
            "tags": ["ens"],
            "ens": {
                "name": "vitalik.eth",
                "avatar": "https://example.com/avatar.png",
                "records": {"com.twitter": "vitalikbuterin"},
            },
        },
        {
            "version": 1,
            "record_type": "address",
            "data": SECOND,
            "tags": ["friend"],
        },
    ]
}

STATS_BODY = {"following_count": 2, "followers_count": 9}

SEARCH_BODY = {
    "following": [
        {
            "version": 1,
            "record_type": "address",
            "data": SEARCH_HIT,
            "tags": ["ens"],
            "ens": {
                "name": "vitalik.eth",
                "avatar": "https://example.com/search.png",
                "records": {"com.twitter": "vitalikbuterin"},
            },
        }
    ]
}

PAGE_BODY = {
    "following": [
        {
            "version": 1,
            "record_type": "address",
            "data": PAGE_HIT,
            "tags": ["page"],
        }
    ]
}


class _EFPHandler(BaseHTTPRequestHandler):
    paths: list[str] = []

    def do_GET(self):
        _EFPHandler.paths.append(self.path)
        parsed = urlparse(self.path)
        query = parse_qs(parsed.query)
        if parsed.path.endswith("/stats"):
            body = STATS_BODY
        elif parsed.path.endswith("/searchFollowing"):
            body = SEARCH_BODY if query.get("term", [""])[0] == "vitalik" else {"following": []}
        else:
            offset = int(query.get("offset", ["0"])[0] or 0)
            body = PAGE_BODY if offset > 0 else FOLLOWING_BODY
        payload = json.dumps(body).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)

    def log_message(self, format, *args):
        return


@pytest.fixture
def efp_mock():
    _EFPHandler.paths = []
    server = ThreadingHTTPServer(("0.0.0.0", 0), _EFPHandler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        yield server.server_address[1]
    finally:
        server.shutdown()
        thread.join(timeout=5)


def _efp_base_url(backend, port: int) -> str:
    host = "host.docker.internal" if backend.container else "127.0.0.1"
    return f"http://{host}:{port}"


@pytest.mark.rpc
class TestWalletFollowingEFP:
    @pytest.fixture
    def following_profile(self, backend_factory, efp_mock):
        backend = backend_factory("efp-following")
        backend.init_status_backend()
        backend.create_account_and_login(
            password=fake.profile_password(),
            efp_base_url=_efp_base_url(backend, efp_mock),
        )
        backend.wait_for_login()
        backend.wakuext_service.start_messenger()
        backend.wallet_service.start_wallet()
        return backend, messenger.wallet_address(backend)

    def test_get_following_addresses_stats_search_and_page(self, following_profile):
        backend, user_address = following_profile

        addresses = backend.wallet_service.get_following_addresses(user_address, "", 10, 0)
        assert len(addresses) == 2
        assert addresses[0]["address"].lower() == VITALIK.lower()
        assert addresses[0]["ensName"] == "vitalik.eth"
        assert addresses[0]["avatar"] == "https://example.com/avatar.png"
        assert addresses[0]["tags"] == ["ens"]
        assert addresses[0]["records"]["com.twitter"] == "vitalikbuterin"
        assert addresses[1]["address"].lower() == SECOND.lower()
        assert addresses[1]["ensName"] == ""
        paths = _EFPHandler.paths
        assert any("/following?" in p and "limit=10" in p and "offset=0" in p for p in paths)

        assert backend.wallet_service.get_following_stats(user_address) == 2
        assert any("/stats" in p for p in _EFPHandler.paths)

        _EFPHandler.paths = []
        searched = backend.wallet_service.get_following_addresses(user_address, "vitalik", 10, 0)
        assert len(searched) == 1
        assert searched[0]["address"].lower() == SEARCH_HIT.lower()
        assert searched[0]["ensName"] == "vitalik.eth"
        assert searched[0]["avatar"] == "https://example.com/search.png"
        assert any("/searchFollowing?" in p and "term=vitalik" in p for p in _EFPHandler.paths)

        _EFPHandler.paths = []
        paged = backend.wallet_service.get_following_addresses(user_address, "", 5, 20)
        assert len(paged) == 1
        assert paged[0]["address"].lower() == PAGE_HIT.lower()
        assert paged[0]["tags"] == ["page"]
        assert paged[0]["ensName"] == ""
        assert any("limit=5" in p and "offset=20" in p for p in _EFPHandler.paths)
