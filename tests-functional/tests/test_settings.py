import pytest
from clients.api import ApiResponseError


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

    def test_news_notifications_enabled(self):
        result = self.config.settings_service.news_notifications_enabled()
        assert result is not None, "Expected a non-null result"
        assert isinstance(result, bool), f"Expected bool, got {type(result)}"

    def test_toggle_news_notifications_enabled(self):
        settings = self.config.settings_service
        settings.save_setting("news-notifications-enabled?", True)
        enabled_state = settings.news_notifications_enabled()
        print(f"[Enabled] news-notifications-enabled = {enabled_state}")
        assert enabled_state is True, "Expected news notifications to be enabled"
        settings.save_setting("news-notifications-enabled?", False)
        disabled_state = settings.news_notifications_enabled()
        print(f"[Disabled] news-notifications-enabled = {disabled_state}")
        assert disabled_state is False, "Expected news notifications to be disabled"

    def test_toggle_news_rss_enabled(self):
        s = self.config.settings_service
        ret = s.save_setting("news-rss-enabled?", False)
        assert ret is None
        assert s.news_rss_enabled() is False
        ret = s.save_setting("news-rss-enabled?", True)
        assert ret is None
        assert s.news_rss_enabled() is True

    @pytest.mark.xfail(reason="backend currently does not validate backup-path; it is stored verbatim")
    def test_set_invalid_backup_path(self):
        invalid_path = "/invalid/path/that/does@$<>not|exist"
        result = self.config.settings_service.save_setting("backup-path", invalid_path)
        backup_path = self.config.settings_service.backup_path()
        assert backup_path != invalid_path, f"Backend incorrectly saved invalid path: {backup_path}"
        assert result is None or result == "", f"Expected save_setting to fail or return None, got: {result}"

    def test_set_valid_backup_path(self):
        valid_path = "/root/.config/Status/backups"
        assert self.config.settings_service.backup_path() == valid_path
        result = self.config.settings_service.save_setting("backup-path", valid_path)
        assert result is None
        assert self.config.settings_service.backup_path() == valid_path

    def test_get_backup_path_type_and_value(self):
        backup_path = self.config.settings_service.backup_path()
        assert backup_path is not None, "Expected a non-null backup path"
        assert isinstance(backup_path, str), f"Expected string, got {type(backup_path)}"
        assert backup_path != "", "Expected backup path to be non-empty"
        print("Current backup path:", backup_path)

    def test_messages_backup_enabled_type(self):
        result = self.config.settings_service.messages_backup_enabled()
        assert result is not None, "Expected a non-null result"
        assert isinstance(result, bool), f"Expected bool, got {type(result)}"
        print("Messages backup enabled:", result)

    @pytest.mark.parametrize("value", [False, True])
    def test_toggle_messages_backup_enabled(self, value):
        self.config.settings_service.save_setting("messages-backup-enabled?", value)
        result = self.config.settings_service.messages_backup_enabled()
        assert isinstance(result, bool)
        assert result is value
        print("Current backup status:", result)

    def test_notifications_get_allow_notifications(self):
        result = self.config.settings_service.notifications_get_allow_notifications()
        assert result is not None, "Expected a non-null result"
        assert isinstance(result, bool), f"Expected bool, got {type(result)}"

    @pytest.mark.parametrize("bad", ["true", 1, 0, [], {}, 3.14])
    def test_notifications_setter_allow_notifications_invalid_values(self, bad):
        print(f"Testing setter with invalid type: {bad!r}")
        with pytest.raises(ApiResponseError) as exc:
            self.config.settings_service.notifications_set_allow_notifications(bad)
        msg = str(exc.value)
        print(f"Received expected ApiResponseError: {msg}")
        assert "-32602" in msg or "cannot unmarshal" in msg.lower()

    def test_notifications_setter_allow_notifications_none(self, bad):
        print("Testing setter with None value")
        res = self.config.settings_service.notifications_set_allow_notifications(None)
        print(f"Setter returned: {res}")
        assert res is None
        got = self.config.settings_service.notifications_get_allow_notifications()
        print(f"Getter returned: {got}")
        assert isinstance(got, bool), f"Expected bool, got {type(got)}"

    def test_notifications_allow_alter_value(self):
        for value in [True, False, True]:
            print(f"Setting allow_notifications to {value}")
            result = self.config.settings_service.notifications_set_allow_notifications(value)
            print(f"Setter returned: {result}")
            assert result is None
            got = self.config.settings_service.notifications_get_allow_notifications()
            print(f"Getter returned: {got}")
            assert isinstance(got, bool)
            assert got is value

    def test_notifications_get_one_to_one_chats_return_type(self):
        print("Calling NotificationsGetOneToOneChats()")
        result = self.config.settings_service.notifications_get_one_to_one_chats()
        print(f"Function returned: {result}")
        assert result is not None, "Expected a non-null return value"
        assert isinstance(result, str), f"Expected str, got {type(result)}"

    @pytest.mark.parametrize("invalid_value", [123, True, [], {}, 3.14])
    def test_notifications_set_one_to_one_chats_invalid_type(self, invalid_value):
        print(f"Testing invalid type for OneToOneChats: {invalid_value!r}")
        with pytest.raises(Exception) as exc:
            self.config.settings_service.notifications_set_one_to_one_chats(invalid_value)
        print(f"Received expected exception: {exc.value}")

    @pytest.mark.parametrize(
        "value",
        [
            "test-value",
            "",
        ],
    )
    def test_notifications_set_get_one_to_one_chats(self, value):
        print(f"Setting OneToOneChats to: '{value}'")
        result = self.config.settings_service.notifications_set_one_to_one_chats(value)
        print(f"Setter returned: {result}")
        assert result is None, f"Expected None, got {result}"

        got = self.config.settings_service.notifications_get_one_to_one_chats()
        print(f"Getter returned: '{got}'")
        assert isinstance(got, str)
        assert got == value

    def test_notifications_get_group_chats_return_type(self):
        print("Calling NotificationsGetGroupChats()")
        result = self.config.settings_service.notifications_get_group_chats()
        print(f"Function returned: {result}")
        assert result is not None, "Expected a non-null return value"
        assert isinstance(result, str), f"Expected str, got {type(result)}"

    @pytest.mark.parametrize("value", ["SendAlerts", "TurnOff", "NULL"])
    def test_notifications_set_and_get_group_chats(self, value):
        print(f"Setting GroupChats to: '{value}'")
        result = self.config.settings_service.notifications_set_group_chats(value)
        print(f"Setter returned: {result}")
        assert result is None, f"Expected None, got {result}"
        got = self.config.settings_service.notifications_get_group_chats()
        print(f"Getter returned: '{got}'")
        assert got is not None, "Expected non-null value from getter"
        assert isinstance(got, str), f"Expected str, got {type(got)}"
        assert got == value, f"Expected '{value}', got '{got}'"

    @pytest.mark.xfail(reason="API accepts wrong values")
    @pytest.mark.parametrize(
        "invalid_value",
        [
            "@#$%^&*()!",
            "🚫invalid🔥",
            "DROP TABLE users;",
            "TurnOn123",
            "",
            "    ",
        ],
    )
    def test_notifications_set_group_chats_invalid_values(self, invalid_value):
        print(f"Testing invalid GroupChats value: '{invalid_value}'")
        with pytest.raises(ApiResponseError) as exc:
            self.config.settings_service.notifications_set_group_chats(invalid_value)
        print(f"Received expected ApiResponseError: {exc.value}")

    def test_notifications_set_and_get_all_messages(self):
        value = "AllMessages"
        print(f"Setting AllMessages to: '{value}'")
        result = self.config.settings_service.notifications_set_all_messages(value)
        print(f"Setter returned: {result}")
        assert result is None, f"Expected None, got {result}"
        got = self.config.settings_service.notifications_get_all_messages()
        print(f"Getter returned: '{got}'")
        assert got is not None, "Expected non-null value from getter"
        assert isinstance(got, str), f"Expected str, got {type(got)}"
        assert got == value, f"Expected '{value}', got '{got}'"
