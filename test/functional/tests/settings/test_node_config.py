import logging

import pytest

logger = logging.getLogger(__name__)


@pytest.mark.rpc
class TestNodeConfig:

    @pytest.fixture()
    def config(self, backend_new_profile):
        return backend_new_profile("settings-backend")

    def test_verify_node_config_stability(self, config):
        cfg1 = config.settings_service.get_node_config()
        cfg2 = config.settings_service.get_node_config()
        assert cfg1 == cfg2, "NodeConfig should be stable across calls"

    def test_check_node_config_params(self, config):
        cfg = config.settings_service.get_node_config()
        boot_api = config.get_boot_api_config()
        assert cfg["WSEnabled"] == boot_api["wsEnabled"]
        assert cfg["WSHost"] == boot_api["wsHost"]
        assert cfg["WSPort"] == boot_api["wsPort"]
        assert cfg["HTTPEnabled"] == boot_api["httpEnabled"]
        assert cfg["HTTPPort"] == boot_api["httpPort"]

    def test_verify_node_config_enforce(self, backend_new_profile):
        forced_network_id = 4242
        backend = backend_new_profile("forced_network", network_id=forced_network_id)
        cfg = backend.settings_service.get_node_config()
        network_id = cfg["NetworkId"]
        assert network_id is not None, f"NetworkId key missing in node config: {cfg}"
        assert int(network_id) == forced_network_id, f"Expected NetworkId={forced_network_id}, got {network_id}"
