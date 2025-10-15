import pytest


@pytest.mark.rpc
class TestSettings:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.config = backend_new_profile("sender")

    def test_verify_node_config_stability(self):
        cfg1 = self.config.settings_service.get_node_config()
        cfg2 = self.config.settings_service.get_node_config()
        assert cfg1 == cfg2, "NodeConfig should be stable across calls"

    def test_check_node_config_params(self):
        cfg = self.config.settings_service.get_node_config()
        boot_api = self.config.get_boot_api_config()
        assert cfg["WSEnabled"] == boot_api["wsEnabled"]
        assert cfg["WSHost"] == boot_api["wsHost"]
        assert cfg["WSPort"] == boot_api["wsPort"]
        assert cfg["HTTPEnabled"] == boot_api["httpEnabled"]
        assert cfg["HTTPPort"] == boot_api["httpPort"]

    def test_verify_node_config_enforce(self, backend_new_profile):
        forced_network_id = 4242
        forced = backend_new_profile("forced_network", network_id=forced_network_id)
        cfg = forced.settings_service.get_node_config()
        network_id = cfg.get("NetworkId", cfg.get("networkId"))
        assert network_id is not None, f"NetworkId key missing in node config: {cfg}"
        assert int(network_id) == forced_network_id, f"Expected NetworkId={forced_network_id}, got {network_id}"

    def test_news_feed_enabled(self):
        result = self.config.settings_service.news_feed_enabled()
        assert isinstance(result, bool), f"Expected boolean, got {type(result)}"
        print(f"News feed enabled: {result}")

    def test_toggle_news_feed_enabled(self):
        self.config.settings_service.save_setting("news-feed-enabled?", False)
        is_enabled = self.config.settings_service.news_feed_enabled()
        assert is_enabled is False, "Expected False"

        # Optional: verify the raw settings map also reflects it
        settings_map = self.config.settings_service.get_settings()
        assert settings_map.get("news-feed-enabled?") is False

        # Cleanup: re-enable
        self.config.settings_service.save_setting("news-feed-enabled?", True)
        assert self.config.settings_service.news_feed_enabled() is True
