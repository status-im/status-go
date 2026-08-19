import pytest


@pytest.mark.accounts
@pytest.mark.rpc
class TestAppGeneral:

    @pytest.fixture(autouse=True)
    def setup_backend(self, backend_new_profile):
        """Initialize one backend for each test function"""
        self.rpc_client = backend_new_profile("rpc_client")

    def test_get_currencies(self):
        result = self.rpc_client.appgeneral_service.get_currencies()
        assert result[0].get("emoji") == "🇺🇸"
        assert result[0].get("name") == "United States Dollar"
        assert result[0].get("shortName") == "USD"
        assert result[0].get("isPopular") is True
        assert result[0].get("isToken") is False
        assert result[0].get("symbol") == "$"
        assert result[0].get("unicode") == "1f1fa-1f1f8"

    @pytest.mark.skip(reason="Skipped due to https://github.com/status-im/status-go/issues/6957")
    def test_version(self):
        result = self.rpc_client.appgeneral_service.version()
        assert result != ""
