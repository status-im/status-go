import logging
import pytest
from clients.api import ApiResponseError
import datetime

# import time

logger = logging.getLogger(__name__)


@pytest.mark.rpc
class TestSettings:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.config = backend_new_profile("sender")

    def test_verify_node_config_stability(self):
        cfg1 = self.config.settings_service.get_node_config()
        cfg2 = self.config.settings_service.get_node_config()
        assert cfg1 == cfg2, "NodeConfig should be stable across calls"

    # def test_check_node_config_params(self):
    #     cfg = self.config.settings_service.get_node_config()
    #     boot_api = self.config.get_boot_api_config()
    #     assert cfg["WSEnabled"] == boot_api["wsEnabled"]
    #     assert cfg["WSHost"] == boot_api["wsHost"]
    #     assert cfg["WSPort"] == boot_api["wsPort"]
    #     assert cfg["HTTPEnabled"] == boot_api["httpEnabled"]
    #     assert cfg["HTTPPort"] == boot_api["httpPort"]

    def test_verify_node_config_enforce(self, backend_new_profile):
        forced_network_id = 4242
        forced = backend_new_profile("forced_network", network_id=forced_network_id)
        cfg = forced.settings_service.get_node_config()
        network_id = cfg.get("NetworkId", cfg.get("networkId"))
        assert network_id is not None, f"NetworkId key missing in node config: {cfg}"
        assert int(network_id) == forced_network_id, f"Expected NetworkId={forced_network_id}, got {network_id}"

    # def test_news_feed_enabled(self):
    #     result = self.config.settings_service.news_feed_enabled()
    #     assert isinstance(result, bool), f"Expected boolean, got {type(result)}"

    # def test_news_notifications_enabled(self):
    #     result = self.config.settings_service.news_notifications_enabled()
    #     assert result is not None, "Expected a non-null result"
    #     assert isinstance(result, bool), f"Expected bool, got {type(result)}"

    # def test_toggle_news_notifications_enabled(self):
    #     settings = self.config.settings_service
    #     settings.save_setting("news-notifications-enabled?", True)
    #     enabled_state = settings.news_notifications_enabled()
    #     assert enabled_state is True, "Expected news notifications to be enabled"
    #     settings.save_setting("news-notifications-enabled?", False)
    #     disabled_state = settings.news_notifications_enabled()
    #     assert disabled_state is False, "Expected news notifications to be disabled"

    # def test_toggle_news_rss_enabled(self):
    #     s = self.config.settings_service
    #     ret = s.save_setting("news-rss-enabled?", False)
    #     assert ret is None
    #     assert s.news_rss_enabled() is False
    #     ret = s.save_setting("news-rss-enabled?", True)
    #     assert ret is None
    #     assert s.news_rss_enabled() is True

    @pytest.mark.xfail(reason="backend currently does not validate backup-path; it is stored verbatim")
    def test_set_invalid_backup_path(self):
        invalid_path = "/invalid/path/that/does@$<>not|exist"
        result = self.config.settings_service.save_setting("backup-path", invalid_path)
        backup_path = self.config.settings_service.backup_path()
        assert backup_path != invalid_path, f"Backend incorrectly saved invalid path: {backup_path}"
        assert result is None or result == "", f"Expected save_setting to fail or return None, got: {result}"

    # def test_set_valid_backup_path(self):
    #     valid_path = "/root/.config/Status/backups"
    #     assert self.config.settings_service.backup_path() == valid_path
    #     result = self.config.settings_service.save_setting("backup-path", valid_path)
    #     assert result is None
    #     assert self.config.settings_service.backup_path() == valid_path

    def test_get_backup_path_type_and_value(self):
        backup_path = self.config.settings_service.backup_path()
        assert backup_path is not None, "Expected a non-null backup path"
        assert isinstance(backup_path, str), f"Expected string, got {type(backup_path)}"
        assert backup_path != "", "Expected backup path to be non-empty"

    def test_messages_backup_enabled_type(self):
        result = self.config.settings_service.messages_backup_enabled()
        assert result is not None, "Expected a non-null result"
        assert isinstance(result, bool), f"Expected bool, got {type(result)}"

    def test_toggle_messages_backup_enabled(self):
        value = True
        self.config.settings_service.save_setting("messages-backup-enabled?", value)
        result = self.config.settings_service.messages_backup_enabled()
        assert isinstance(result, bool)
        assert result is value

    def test_notifications_get_allow_notifications(self):
        result = self.config.settings_service.notifications_get_allow_notifications()
        assert result is not None, "Expected a non-null result"
        assert isinstance(result, bool), f"Expected bool, got {type(result)}"

    def test_notifications_setter_allow_notifications_invalid_values(self):
        with pytest.raises(ApiResponseError) as exc:
            self.config.settings_service.notifications_set_allow_notifications(1)
        msg = str(exc.value)
        assert "-32602" in msg or "cannot unmarshal" in msg.lower()

    def test_notifications_setter_allow_notifications_none(self):
        res = self.config.settings_service.notifications_set_allow_notifications(None)
        assert res is None
        got = self.config.settings_service.notifications_get_allow_notifications()
        assert isinstance(got, bool), f"Expected bool, got {type(got)}"

    def test_notifications_allow_alter_value(self):
        for value in [True, False, True]:
            result = self.config.settings_service.notifications_set_allow_notifications(value)
            assert result is None
            got = self.config.settings_service.notifications_get_allow_notifications()
            assert isinstance(got, bool)
            assert got is value

    def test_notifications_get_one_to_one_chats_return_type(self):
        result = self.config.settings_service.notifications_get_one_to_one_chats()
        assert result is not None, "Expected a non-null return value"
        assert isinstance(result, str), f"Expected str, got {type(result)}"

    def test_notifications_set_one_to_one_chats_invalid_type(self):
        with pytest.raises(Exception):
            self.config.settings_service.notifications_set_one_to_one_chats(123)

    @pytest.mark.parametrize(
        "value",
        [
            "test-value",
            "",
        ],
    )
    def test_notifications_set_get_one_to_one_chats(self, value):
        logger.info(f"Setting OneToOneChats to: '{value}'")
        result = self.config.settings_service.notifications_set_one_to_one_chats(value)
        assert result is None, f"Expected None, got {result}"

        got = self.config.settings_service.notifications_get_one_to_one_chats()
        assert isinstance(got, str)
        assert got == value

    def test_notifications_get_group_chats_return_type(self):
        result = self.config.settings_service.notifications_get_group_chats()
        assert result is not None, "Expected a non-null return value"
        assert isinstance(result, str), f"Expected str, got {type(result)}"

    def test_notifications_set_and_get_group_chats(self):
        value = "SendAlerts"
        result = self.config.settings_service.notifications_set_group_chats(value)
        assert result is None, f"Expected None, got {result}"
        got = self.config.settings_service.notifications_get_group_chats()
        assert got is not None, "Expected non-null value from getter"
        assert isinstance(got, str), f"Expected str, got {type(got)}"
        assert got == value, f"Expected '{value}', got '{got}'"

    @pytest.mark.xfail(reason="API accepts wrong values")
    def test_notifications_set_group_chats_invalid_values(self):
        with pytest.raises(ApiResponseError):
            self.config.settings_service.notifications_set_group_chats("")

    def test_notifications_set_and_get_all_messages(self):
        value = "AllMessages"
        result = self.config.settings_service.notifications_set_all_messages(value)
        assert result is None, f"Expected None, got {result}"
        got = self.config.settings_service.notifications_get_all_messages()
        assert got is not None, "Expected non-null value from getter"
        assert isinstance(got, str), f"Expected str, got {type(got)}"
        assert got == value, f"Expected '{value}', got '{got}'"

    def test_notifications_get_contact_requests(self):
        result = self.config.settings_service.notifications_get_contact_requests()
        assert result is not None, "Expected non-null result from NotificationsGetContactRequests()"
        assert isinstance(result, str), f"Expected type str, got {type(result)}"

    def test_notifications_set_contact_requests_rejects_wrong_types_and_preserves_value(self):
        original_value = self.config.settings_service.notifications_get_contact_requests()
        with pytest.raises(ApiResponseError):
            self.config.settings_service.notifications_set_contact_requests(123)
        current_value = self.config.settings_service.notifications_get_contact_requests()
        assert current_value == original_value, f"Value changed after invalid set attempt. Expected {original_value!r}, got {current_value!r}"

    def test_notifications_set_and_get_contact_requests(self):
        valid_value = "SendAlerts"
        self.config.settings_service.notifications_set_contact_requests(valid_value)
        get_result = self.config.settings_service.notifications_get_contact_requests()
        assert get_result is not None, "Expected non-null value from getter"
        assert isinstance(get_result, str), f"Expected string from getter, got {type(get_result)}"
        assert get_result == valid_value, f"Expected getter to return '{valid_value}', but got '{get_result}'"

    def test_notifications_get_identity_verification_requests(self):
        result = self.config.settings_service.notifications_get_identity_verification_requests()
        assert isinstance(result, str), f"Expected string, got {type(result)}"
        valid_values = ["SendAlerts", "TurnOff"]
        assert result in valid_values, f"Unexpected value: {result}"

    @pytest.mark.xfail(reason="API accepts wrong values")
    def test_notifications_set_identity_verification_requests_rejects_invalid_values(self):
        invalid_value = ""
        original_value = self.config.settings_service.notifications_get_identity_verification_requests()
        with pytest.raises(ApiResponseError):
            self.config.settings_service.notifications_set_identity_verification_requests(invalid_value)
        current_value = self.config.settings_service.notifications_get_identity_verification_requests()
        assert current_value == original_value, f"Value changed after invalid input {invalid_value!r}: expected {original_value}, got {current_value}"
        assert current_value in ["SendAlerts", "TurnOff"], f"Unexpected stored value: {current_value}"

    def test_notifications_get_sound_enabled(self):
        result = self.config.settings_service.notifications_get_sound_enabled()
        assert result is not None, "Expected non-null result"
        assert isinstance(result, bool), f"Expected bool, got {type(result)}"

    def test_toggle_notifications_sound_enabled(self):
        initial_value = self.config.settings_service.notifications_get_sound_enabled()
        assert isinstance(initial_value, bool), f"Expected bool, got {type(initial_value)}"
        new_value = not initial_value
        self.config.settings_service.notifications_set_sound_enabled(new_value)
        updated_value = self.config.settings_service.notifications_get_sound_enabled()
        assert updated_value == new_value, f"Expected sound enabled to be {new_value}, but got {updated_value}"
        self.config.settings_service.notifications_set_sound_enabled(initial_value)
        reverted_value = self.config.settings_service.notifications_get_sound_enabled()
        assert reverted_value == initial_value, "Failed to revert sound enabled to its original state"

    def test_notifications_set_sound_enabled_invalid_type(self):
        invalid_value = "true"
        initial_value = self.config.settings_service.notifications_get_sound_enabled()
        try:
            self.config.settings_service.notifications_set_sound_enabled(invalid_value)
        except Exception:
            pass
        final_value = self.config.settings_service.notifications_get_sound_enabled()
        assert final_value == initial_value, f"Sound enabled changed after invalid input {invalid_value}"

    def test_thirdparty_services_enabled(self):
        result = self.config.settings_service.thirdparty_services_enabled()
        assert result is not None, "Expected a non-null result"
        assert isinstance(result, bool), f"Expected bool, got {type(result)}"

    def test_last_tokens_update_type_check(self):
        result = self.config.settings_service.last_tokens_update()
        assert result is not None, "Expected non-null timestamp"
        assert isinstance(result, str), f"Expected string, got {type(result)}"

    def test_last_tokens_update_returns_valid_iso_datetime(self):
        result = self.config.settings_service.last_tokens_update()
        try:
            _ = datetime.datetime.fromisoformat(result.replace("Z", "+00:00"))
        except Exception as e:
            pytest.fail(f"Returned value is not a valid ISO datetime string: {result}. Error: {e}")

    # def test_last_tokens_update_advances_after_updating_token_preferences(self):
    #     t1_raw = self.config.settings_service.last_tokens_update()
    #     t1 = datetime.datetime.fromisoformat(t1_raw.replace("Z", "+00:00"))
    #     time.sleep(1.2)
    #     current_prefs = self.config.accounts_service.get_token_preferences()
    #     self.config.accounts_service.update_token_preferences(current_prefs)
    #     t2_raw = self.config.settings_service.last_tokens_update()
    #     t2 = datetime.datetime.fromisoformat(t2_raw.replace("Z", "+00:00"))
    #     assert t2 >= t1, f"Expected last-tokens-update to advance or stay same; got T1={t1} T2={t2}"

    def test_mnemonic_was_shown(self):
        result = self.config.settings_service.mnemonic_was_shown()
        assert result is None or result == "", f"Expected no return value, got: {result}"

    def test_set_bio_valid_string(self):
        valid_bio = "This is a new test bio"
        result = self.config.settings_service.set_bio(valid_bio)
        assert result is None or result == "", f"Expected None or empty string, got {result}"

    def test_set_bio_invalid_type(self):
        invalid_bio = 123
        try:
            result = self.config.settings_service.set_bio(invalid_bio)
            pytest.fail(f"Expected an exception for invalid bio {invalid_bio}, got {result}")
        except Exception:
            assert True

    def test_delete_exemptions_valid_id(self):
        result = self.config.settings_service.delete_exemptions("12345")
        assert result is None or result == "", f"Expected None or empty string, got {result}"

    # def test_delete_exemptions_invalid_id(self):
    #     invalid_id = None
    #     try:
    #         result = self.config.settings_service.delete_exemptions(invalid_id)
    #         pytest.fail(f"Expected exception for invalid id: {invalid_id}, but got {result}")
    #     except Exception:
    #         assert True

    def test_notifications_set_exemptions_valid(self):
        test_id = "chat:12345"
        mute_all = True
        personal = "all"
        global_ = "mentions-only"
        other = "none"
        result = self.config.settings_service.notifications_set_exemptions(test_id, mute_all, personal, global_, other)
        assert result is None or result == "", f"Expected None or empty string, got {result}"

    def test_set_exemptions_and_verify_each_field(self):
        eid = "chat:test001"
        mute = True
        personal = "all"
        global_ = "mentions-only"
        other = "none"

        res = self.config.settings_service.notifications_set_exemptions(eid, mute, personal, global_, other)
        assert res is None or res == "", f"Expected no payload, got {res}"

        got_mute = self.config.settings_service.notifications_get_ex_mute_all_messages(eid)
        assert isinstance(got_mute, bool)
        assert got_mute is mute

        got_personal = self.config.settings_service.notifications_get_ex_personal_mentions(eid)
        assert isinstance(got_personal, str)
        assert got_personal == personal

        got_global = self.config.settings_service.notifications_get_ex_global_mentions(eid)
        assert isinstance(got_global, str)
        assert got_global == global_

        got_other = self.config.settings_service.notifications_get_ex_other_messages(eid)
        assert isinstance(got_other, str)
        assert got_other == other

    def test_update_existing_exemptions(self):
        eid = "chat:test002"
        self.config.settings_service.notifications_set_exemptions(eid, False, "none", "none", "none")
        mute2 = True
        personal2 = "all"
        global2 = "mentions-only"
        other2 = "important-only"
        res = self.config.settings_service.notifications_set_exemptions(eid, mute2, personal2, global2, other2)
        assert res is None or res == ""
        assert self.config.settings_service.notifications_get_ex_mute_all_messages(eid) is mute2
        assert self.config.settings_service.notifications_get_ex_personal_mentions(eid) == personal2
        assert self.config.settings_service.notifications_get_ex_global_mentions(eid) == global2
        assert self.config.settings_service.notifications_get_ex_other_messages(eid) == other2

    @pytest.mark.parametrize(
        "params",
        [
            (123, True, "all", "all", "all"),
            ("chat:x", "true", "all", "all", "all"),
            ("chat:x", True, 5, "all", "all"),
            ("chat:x", True, "all", 0, "all"),
            ("chat:x", True, "all", "all", 0),
        ],
    )
    def test_set_exemptions_invalid_inputs(self, params):
        eid, mute, personal, global_, other = params
        logger.info(
            f"\n[TEST] Invalid call: id={eid!r}({type(eid).__name__}), "
            f"mute={mute!r}({type(mute).__name__}), "
            f"personal={personal!r}({type(personal).__name__}), "
            f"global={global_!r}({type(global_).__name__}), "
            f"other={other!r}({type(other).__name__})"
        )
        try:
            self.config.settings_service.notifications_set_exemptions(eid, mute, personal, global_, other)
            pytest.fail("Expected exception or backend error for invalid types, but call succeeded.")
        except Exception:
            assert True

    @pytest.mark.skip(reason="Backend RPC notificationsGetDefaultExemptions not yet fully deployed/testable")
    def test_delete_exemptions_matches_defaults(self):
        eid = "chat:defaults-check"
        mute = True
        personal = "all"
        global_ = "mentions-only"
        other = "none"

        res = self.config.settings_service.notifications_set_exemptions(eid, mute, personal, global_, other)
        assert res is None or res == ""

        res_del = self.config.settings_service.delete_exemptions(eid)
        assert res_del is None or res_del == ""

        got_mute = self.config.settings_service.notifications_get_ex_mute_all_messages(eid)
        got_personal = self.config.settings_service.notifications_get_ex_personal_mentions(eid)
        got_global = self.config.settings_service.notifications_get_ex_global_mentions(eid)
        got_other = self.config.settings_service.notifications_get_ex_other_messages(eid)

        defaults = self.config.settings_service.notifications_get_default_exemptions()

        assert got_mute is defaults["muteAllMessages"], f"Expected mute={defaults['muteAllMessages']}, got {got_mute}"
        assert got_personal == defaults["personalMentions"], f"Expected personal={defaults['personalMentions']}, got {got_personal}"
        assert got_global == defaults["globalMentions"], f"Expected global={defaults['globalMentions']}, got {got_global}"
        assert got_other == defaults["otherMessages"], f"Expected other={defaults['otherMessages']}, got {got_other}"

    def test_notifications_get_message_preview_type(self):
        value = self.config.settings_service.notifications_get_message_preview()
        assert value is not None
        assert isinstance(value, int)

    def test_notifications_message_preview_roundtrip_set_and_get(self):
        original = self.config.settings_service.notifications_get_message_preview()
        assert isinstance(original, int)
        candidate = 1 if original == 0 else 0
        res = self.config.settings_service.notifications_set_message_preview(candidate)
        assert res is None or res == ""
        got = self.config.settings_service.notifications_get_message_preview()
        assert got == candidate
        try:
            self.config.settings_service.notifications_set_message_preview(original)
        except Exception:
            pass

    @pytest.mark.xfail(reason=" API accepts -1")
    def test_notifications_message_preview_rejects_invalid_values(self):
        bad_value = -1
        before = self.config.settings_service.notifications_get_message_preview()
        try:
            _ = self.config.settings_service.notifications_set_message_preview(bad_value)
        except ApiResponseError:
            pass
        except Exception:
            pass
        after = self.config.settings_service.notifications_get_message_preview()
        assert after == before

    def test_notifications_get_volume_returns_valid_type(self):
        result = self.config.settings_service.notifications_get_volume()
        assert isinstance(result, int), f"Expected int, got {type(result)}"

    def test_notifications_set_and_get_volume(self):
        """Setter should accept integer; getter should return the same integer type."""
        test_values = [0, 5, 10, 42]

        for value in test_values:
            result = self.config.settings_service.notifications_set_volume(value)
            assert result is None or result == "", f"Unexpected setter response: {result}"
            get_result = self.config.settings_service.notifications_get_volume()

            assert isinstance(get_result, int), f"Expected int, got {type(get_result)}"
            assert get_result == value, f"Getter returned {get_result}, expected {value}"

    def test_notifications_set_volume_rejects_invalid_types(self):
        invalid_value = "loud"

        original_value = self.config.settings_service.notifications_get_volume()
        assert isinstance(original_value, int), "Precondition failed: getter did not return int"

        with pytest.raises(ApiResponseError):
            self.config.settings_service.notifications_set_volume(invalid_value)
        current_value = self.config.settings_service.notifications_get_volume()
        assert isinstance(current_value, int), f"Getter returned {type(current_value)} after invalid input"
        assert current_value == original_value, f"Volume changed after invalid input: was {original_value}, now {current_value}"

    def test_notifications_get_personal_mentions_type_check(self):
        value = self.config.settings_service.notifications_get_personal_mentions()
        assert value is not None
        assert isinstance(value, str)

    def test_notifications_personal_mentions_setter_getter(self):
        new_value = "all"
        self.config.settings_service.notifications_get_personal_mentions()
        res = self.config.settings_service.notifications_set_personal_mentions(new_value)
        assert res is None or res == ""
        got = self.config.settings_service.notifications_get_personal_mentions()
        assert isinstance(got, str)
        assert got == new_value

    def test_notifications_set_personal_mentions_rejects_wrong_types(self):
        bad_value = 123
        before = self.config.settings_service.notifications_get_personal_mentions()
        with pytest.raises(ApiResponseError):
            self.config.settings_service.notifications_set_personal_mentions(bad_value)
        after = self.config.settings_service.notifications_get_personal_mentions()
        assert after == before

    def test_notifications_get_global_mentions_returns_string(self):
        value = self.config.settings_service.notifications_get_global_mentions()
        assert isinstance(value, str)

    def test_notifications_global_mentions_setter_getter(self):
        new_value = "all"
        self.config.settings_service.notifications_get_global_mentions()
        res = self.config.settings_service.notifications_set_global_mentions(new_value)
        assert res is None or res == ""
        got = self.config.settings_service.notifications_get_global_mentions()
        assert got == new_value

    # @pytest.mark.parametrize("bad_value", [123, True, ["list"], {"k": "v"}, None])
    # def test_notifications_set_global_mentions_rejects_wrong_types_and_preserves_value(self, bad_value):
    #     before = self.config.settings_service.notifications_get_global_mentions()
    #     logger.info(f"[SETUP] before={before!r}, bad_value={bad_value!r} ({type(bad_value).__name__})")
    #     with pytest.raises(ApiResponseError):
    #         self.config.settings_service.notifications_set_global_mentions(bad_value)
    #     after = self.config.settings_service.notifications_get_global_mentions()
    #     assert after == before

    def test_get_settings_reflects(self):
        all_before = self.config.settings_service.get_settings()
        original_value = self.config.settings_service.notifications_get_global_mentions()
        new_value = "mentions-only" if original_value != "mentions-only" else "all"
        self.config.settings_service.notifications_set_global_mentions(new_value)
        all_after = self.config.settings_service.get_settings()
        assert len(all_before) == len(all_after)

    def test_notifications_get_identity_verification_requests_returns_string(self):
        value = self.config.settings_service.notifications_get_identity_verification_requests()
        assert isinstance(value, str)

    def test_notifications_identity_verification_requests_roundtrip(self):
        original = self.config.settings_service.notifications_get_identity_verification_requests()
        new_value = "enabled" if original != "enabled" else "disabled"
        self.config.settings_service.notifications_set_identity_verification_requests(new_value)
        got = self.config.settings_service.notifications_get_identity_verification_requests()
        assert isinstance(got, str)
        assert got == new_value
