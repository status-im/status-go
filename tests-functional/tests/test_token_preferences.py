import pytest


@pytest.mark.rpc
class TestTokenPreferences:

    @pytest.fixture()
    def account(self, backend_new_profile):
        return backend_new_profile("account-backend")

    def test_update_with_valid_token_preferences(self, account):
        token_preferences_before = account.accounts_service.get_token_preferences()
        assert token_preferences_before is None

        new_token_preferences = [
            {"communityId": "123", "groupPosition": 2, "key": "0x1234567890abcdef1234567890abcdef12345678", "position": 3, "visible": True}
        ]
        update_response = account.accounts_service.update_token_preferences(new_token_preferences)
        assert update_response is None

        token_preferences_after = account.accounts_service.get_token_preferences()
        assert token_preferences_after == new_token_preferences

    def test_update_with_invalid_token_preferences(self, account):
        new_token_preferences = [{"key": "0x1234567890abcdef1234567890abcdef12345678", "foo": "bar"}]
        account.accounts_service.update_token_preferences(new_token_preferences)
        token_preferences_after = account.accounts_service.get_token_preferences()
        assert len(token_preferences_after)

        # missing fields are assigned default values
        # extra fields are ignored
        # exiting fields are updated
        token_preference = token_preferences_after[0]
        assert token_preference.get("communityId") == ""
        assert token_preference.get("groupPosition") == 0
        assert token_preference.get("key") == new_token_preferences[0]["key"]
        assert token_preference.get("position") == 0
        assert token_preference.get("visible") is False

    def test_overwrite_existing_token_preferences(self, account):
        new_token_preferences1 = [
            {"communityId": "123", "groupPosition": 2, "key": "0x1234567890abcdef1234567890abcdef12345678", "position": 3, "visible": True}
        ]
        account.accounts_service.update_token_preferences(new_token_preferences1)
        new_token_preferences2 = [
            {"communityId": "567", "groupPosition": 1, "key": "0x1234567890abcdef1234567890abcdef12345676", "position": 1, "visible": False}
        ]
        account.accounts_service.update_token_preferences(new_token_preferences2)
        token_preferences_after = account.accounts_service.get_token_preferences()
        assert token_preferences_after == new_token_preferences2

    def test_add_multiple_token_preferences(self, account):
        new_token_preferences = [
            {"communityId": "123", "groupPosition": 2, "key": "0x1234567890abcdef1234567890abcdef12345678", "position": 3, "visible": True},
            {"communityId": "567", "groupPosition": 1, "key": "0x1234567890abcdef1234567890abcdef12345676", "position": 1, "visible": False},
        ]
        account.accounts_service.update_token_preferences(new_token_preferences)
        token_preferences_after = account.accounts_service.get_token_preferences()
        assert token_preferences_after == new_token_preferences
