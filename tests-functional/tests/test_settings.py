import pytest
from clients.api import ApiResponseError
import datetime
import time


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

    def test_notifications_setter_allow_notifications_none(self):
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

    def test_notifications_get_contact_requests(self):
        result = self.config.settings_service.notifications_get_contact_requests()
        print("Returned value from NotificationsGetContactRequests():", result)
        assert result is not None, "Expected non-null result from NotificationsGetContactRequests()"
        assert isinstance(result, str), f"Expected type str, got {type(result)}"

    @pytest.mark.parametrize("invalid_value", [123, True, {"key": "value"}, ["list"], 4.56])
    def test_notifications_set_contact_requests_rejects_wrong_types_and_preserves_value(self, invalid_value):
        original_value = self.config.settings_service.notifications_get_contact_requests()
        print(f"Original NotificationsGetContactRequests value: {original_value!r}")
        print(f"Trying to set invalid value of type {type(invalid_value)}: {invalid_value}")
        with pytest.raises(ApiResponseError):
            self.config.settings_service.notifications_set_contact_requests(invalid_value)
        current_value = self.config.settings_service.notifications_get_contact_requests()
        print(f"Value after invalid attempt: {current_value!r}")
        assert current_value == original_value, f"Value changed after invalid set attempt. Expected {original_value!r}, got {current_value!r}"

    @pytest.mark.parametrize("valid_value", ["SendAlerts", "TurnOff", "NULL"])
    def test_notifications_set_and_get_contact_requests(self, valid_value):
        print(f"Setting NotificationsSetContactRequests to: {valid_value!r}")
        set_result = self.config.settings_service.notifications_set_contact_requests(valid_value)
        print("Setter result:", set_result)
        get_result = self.config.settings_service.notifications_get_contact_requests()
        print("Getter result:", get_result)
        assert get_result is not None, "Expected non-null value from getter"
        assert isinstance(get_result, str), f"Expected string from getter, got {type(get_result)}"
        assert get_result == valid_value, f"Expected getter to return '{valid_value}', but got '{get_result}'"

    def test_notifications_get_identity_verification_requests(self):
        result = self.config.settings_service.notifications_get_identity_verification_requests()
        print("IdentityVerificationRequests value:", result)
        assert isinstance(result, str), f"Expected string, got {type(result)}"
        valid_values = ["SendAlerts", "TurnOff"]
        assert result in valid_values, f"Unexpected value: {result}"

    @pytest.mark.xfail(reason="API accepts wrong values")
    @pytest.mark.parametrize(
        "invalid_value",
        [
            "",
            "🚫wrong",
            "invalid",
            "123",
            123,
            True,
            "@!#%$",
        ],
    )
    def test_notifications_set_identity_verification_requests_rejects_invalid_values(self, invalid_value):
        original_value = self.config.settings_service.notifications_get_identity_verification_requests()
        print(f"Original IdentityVerificationRequests value: {original_value}")
        print(f"Testing invalid input: {invalid_value!r} ({type(invalid_value)})")
        with pytest.raises(ApiResponseError):
            self.config.settings_service.notifications_set_identity_verification_requests(invalid_value)
        current_value = self.config.settings_service.notifications_get_identity_verification_requests()
        assert current_value == original_value, f"Value changed after invalid input {invalid_value!r}: expected {original_value}, got {current_value}"
        assert current_value in ["SendAlerts", "TurnOff"], f"Unexpected stored value: {current_value}"

    def test_notifications_get_sound_enabled(self):
        result = self.config.settings_service.notifications_get_sound_enabled()
        print("Sound enabled status:", result)
        assert result is not None, "Expected non-null result"
        assert isinstance(result, bool), f"Expected bool, got {type(result)}"

    def test_toggle_notifications_sound_enabled(self):
        initial_value = self.config.settings_service.notifications_get_sound_enabled()
        print(f"Initial sound enabled value: {initial_value}")
        assert isinstance(initial_value, bool), f"Expected bool, got {type(initial_value)}"
        new_value = not initial_value
        self.config.settings_service.notifications_set_sound_enabled(new_value)
        updated_value = self.config.settings_service.notifications_get_sound_enabled()
        print(f"Updated sound enabled value: {updated_value}")
        assert updated_value == new_value, f"Expected sound enabled to be {new_value}, but got {updated_value}"
        self.config.settings_service.notifications_set_sound_enabled(initial_value)
        reverted_value = self.config.settings_service.notifications_get_sound_enabled()
        print(f"Reverted sound enabled value: {reverted_value}")
        assert reverted_value == initial_value, "Failed to revert sound enabled to its original state"

    @pytest.mark.parametrize("invalid_value", ["true", 1, 0, [], {}, "False"])
    def test_notifications_set_sound_enabled_invalid_type(self, invalid_value):
        initial_value = self.config.settings_service.notifications_get_sound_enabled()
        print(f"Initial sound enabled value: {initial_value}")
        try:
            self.config.settings_service.notifications_set_sound_enabled(invalid_value)
        except Exception as e:
            print(f"Expected error for invalid value {invalid_value}: {e}")

        final_value = self.config.settings_service.notifications_get_sound_enabled()
        print(f"Final sound enabled value after invalid set: {final_value}")
        assert final_value == initial_value, f"Sound enabled changed after invalid input {invalid_value}"

    def test_thirdparty_services_enabled(self):
        result = self.config.settings_service.thirdparty_services_enabled()
        print(f"Third-party services enabled: {result}")
        assert result is not None, "Expected a non-null result"
        assert isinstance(result, bool), f"Expected bool, got {type(result)}"

    def test_last_tokens_update_type_check(self):
        result = self.config.settings_service.last_tokens_update()
        print("Last tokens update:", result)
        assert result is not None, "Expected non-null timestamp"
        assert isinstance(result, str), f"Expected string, got {type(result)}"

    def test_last_tokens_update_returns_valid_iso_datetime(self):
        result = self.config.settings_service.last_tokens_update()
        print("Last tokens update:", result)

        try:
            _ = datetime.datetime.fromisoformat(result.replace("Z", "+00:00"))
        except Exception as e:
            pytest.fail(f"Returned value is not a valid ISO datetime string: {result}. Error: {e}")

    def test_last_tokens_update_advances_after_updating_token_preferences(self):
        t1_raw = self.config.settings_service.last_tokens_update()
        print("T1 last tokens update:", t1_raw)
        t1 = datetime.datetime.fromisoformat(t1_raw.replace("Z", "+00:00"))
        time.sleep(1.2)
        current_prefs = self.config.accounts_service.get_token_preferences()
        self.config.accounts_service.update_token_preferences(current_prefs)
        t2_raw = self.config.settings_service.last_tokens_update()
        print("T2 last tokens update:", t2_raw)
        t2 = datetime.datetime.fromisoformat(t2_raw.replace("Z", "+00:00"))
        assert t2 >= t1, f"Expected last-tokens-update to advance or stay same; got T1={t1} T2={t2}"
