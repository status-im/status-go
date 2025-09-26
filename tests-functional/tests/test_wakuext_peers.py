import pytest


class TestRpc:

    @pytest.fixture(autouse=True)
    def setup_backend(self, backend_new_profile):
        """Initialize one backend for each test function"""
        self.rpc_client = backend_new_profile("peers")

    def test_peers(self):
        self.rpc_client.wait_for_online()
        peers = self.rpc_client.wakuext_service.peers()
        assert len(peers) >= 1
        for _, peer in peers.items():
            assert "/vac/waku/relay/2.0.0" in peer.get("protocols")
