import datetime
import time

import pytest


@pytest.mark.rpc
class TestGeneralSettings:

    @pytest.fixture()
    def config(self, backend_new_profile):
        return backend_new_profile("settings-backend")

    def test_thirdparty_services_enabled(self, config):
        result = config.settings_service.thirdparty_services_enabled()
        assert result is not None, "Expected a non-null result"
        assert isinstance(result, bool), f"Expected bool, got {type(result)}"
        assert result is True, "Expected True"

    def test_last_tokens_update_type_check(self, config):
        result = config.settings_service.last_tokens_update()
        assert result is not None, "Expected non-null timestamp"
        assert isinstance(result, str), f"Expected string, got {type(result)}"
        assert result != "", "Expected non-empty string"

    def test_last_tokens_update_returns_valid_iso_datetime(self, config):
        result = config.settings_service.last_tokens_update()
        try:
            _ = datetime.datetime.fromisoformat(result.replace("Z", "+00:00"))
        except Exception as e:
            pytest.fail(f"Returned value is not a valid ISO datetime string: {result}. Error: {e}")

    def test_last_tokens_update_advances_after_updating_token_preferences(self, config):
        dummy_prefs = [{"key": "0x1234567890abcdef1234567890abcdef12345678", "position": 1, "visible": True}]
        config.accounts_service.update_token_preferences(dummy_prefs)
        t1_raw = config.settings_service.last_tokens_update()
        t1 = datetime.datetime.fromisoformat(t1_raw.replace("Z", "+00:00"))
        time.sleep(1.2)
        current_prefs = config.accounts_service.get_token_preferences()
        config.accounts_service.update_token_preferences(current_prefs)
        t2_raw = config.settings_service.last_tokens_update()
        t2 = datetime.datetime.fromisoformat(t2_raw.replace("Z", "+00:00"))
        assert t2 >= t1, f"Expected last-tokens-update to advance or stay same; got T1={t1} T2={t2}"

    def test_mnemonic_was_shown(self, config):
        result = config.settings_service.mnemonic_was_shown()
        assert result is None or result == "", f"Expected no return value, got: {result}"

    def test_set_bio_valid_string(self, config):
        valid_bio = "This is a new test bio"
        result = config.settings_service.set_bio(valid_bio)
        assert result is None or result == "", f"Expected None or empty string, got {result}"

    def test_set_bio_invalid_type(self, config):
        invalid_bio = 123
        try:
            result = config.settings_service.set_bio(invalid_bio)
            pytest.fail(f"Expected an exception for invalid bio {invalid_bio}, got {result}")
        except Exception:
            assert True

    def test_delete_exemptions_valid_id(self, config):
        result = config.settings_service.delete_exemptions("12345")
        assert result is None or result == "", f"Expected None or empty string, got {result}"

    @pytest.mark.skip(reason="Pending on issue resolution https://github.com/status-im/status-go/issues/7102")
    def test_delete_exemptions_invalid_id(self, config):
        invalid_id = None
        try:
            result = config.settings_service.delete_exemptions(invalid_id)
            pytest.fail(f"Expected exception for invalid id: {invalid_id}, but got {result}")
        except Exception:
            assert True
