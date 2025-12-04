import pytest


@pytest.mark.rpc
class TestCollectiblePreferences:

    @pytest.fixture()
    def account(self, backend_new_profile):
        return backend_new_profile("account-backend")

    def test_update_with_valid_collectible_preferences(self, account):
        collectible_preferences_before = account.accounts_service.get_collectible_preferences()
        assert collectible_preferences_before is None

        new_collectible_preferences = [{"key": "collection:1234:1", "position": 3, "type": 0, "visible": True}]
        update_response = account.accounts_service.update_collectible_preferences(new_collectible_preferences)
        assert update_response is None

        collectible_preferences_after = account.accounts_service.get_collectible_preferences()
        assert collectible_preferences_after == new_collectible_preferences

    def test_update_with_invalid_collectible_preferences(self, account):
        new_collectible_preferences = [{"key": "collection:1234:1", "foo": "bar"}]
        account.accounts_service.update_collectible_preferences(new_collectible_preferences)
        collectible_preferences_after = account.accounts_service.get_collectible_preferences()
        assert len(collectible_preferences_after)

        # missing fields are assigned default values
        # extra fields are ignored
        # existing fields are updated
        cp = collectible_preferences_after[0]
        assert cp.get("key") == new_collectible_preferences[0]["key"]
        assert cp.get("position") == 0
        assert cp.get("type") == 0
        assert cp.get("visible") is False

    def test_overwrite_existing_collectible_preferences(self, account):
        new_collectible_preferences1 = [{"key": "collection:1234:1", "position": 3, "type": 0, "visible": True}]
        account.accounts_service.update_collectible_preferences(new_collectible_preferences1)
        new_collectible_preferences2 = [{"key": "collection:5678:2", "position": 1, "type": 1, "visible": False}]
        account.accounts_service.update_collectible_preferences(new_collectible_preferences2)
        collectible_preferences_after = account.accounts_service.get_collectible_preferences()
        assert collectible_preferences_after == new_collectible_preferences2

    def test_add_multiple_collectible_preferences(self, account):
        new_collectible_preferences = [
            {"key": "collection:1234:1", "position": 3, "type": 0, "visible": True},
            {"key": "collection:5678:2", "position": 1, "type": 1, "visible": False},
        ]
        account.accounts_service.update_collectible_preferences(new_collectible_preferences)
        collectible_preferences_after = account.accounts_service.get_collectible_preferences()
        assert collectible_preferences_after == new_collectible_preferences
