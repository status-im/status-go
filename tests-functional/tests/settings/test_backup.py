import pytest


@pytest.mark.rpc
class TestBackupSettings:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.config = backend_new_profile("sender")

    @pytest.mark.skip(reason="backend currently does not validate backup-path; it is stored verbatim")
    def test_set_invalid_backup_path(self):
        invalid_path = "/invalid/path/that/does@$<>not|exist"
        result = self.config.settings_service.save_setting("backup-path", invalid_path)
        backup_path = self.config.settings_service.backup_path()
        assert backup_path != invalid_path, f"Backend incorrectly saved invalid path: {backup_path}"
        assert result is None or result == "", f"Expected save_setting to fail or return None, got: {result}"

    def test_set_valid_backup_path(self):
        current_path = self.config.settings_service.backup_path()

        # Verify it's a valid path (not empty and is a string)
        assert current_path is not None, "Backup path should not be None"
        assert isinstance(current_path, str), f"Expected string, got {type(current_path)}"
        assert current_path != "", "Backup path should not be empty"

        # Test setting the same path explicitly (round-trip test)
        result = self.config.settings_service.save_setting("backup-path", current_path)
        assert result is None
        assert self.config.settings_service.backup_path() == current_path

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
