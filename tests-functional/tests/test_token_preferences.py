import pytest


@pytest.mark.rpc
class TestTokenPreferences:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("sender")

    def test_update_token_preferences(self):
        # Example token preferences payload
        token_preferences = [{"tokenAddress": "0x1234567890abcdef1234567890abcdef12345678", "enabled": True, "displayOrder": 1}]

        response = self.account.accounts_service.update_token_preferences(token_preferences)
        assert "error" not in response

    def test_get_token_preferences(self):
        response = self.account.accounts_service.get_token_preferences()
        assert "error" not in response
        # Optionally check that response is a list
        assert isinstance(response.get("result", []), list)
