import pytest


@pytest.mark.rpc
@pytest.mark.browser
class TestBrowserHistory:

    @pytest.fixture()
    def backend(self, backend_new_profile):
        return backend_new_profile("browser-history")

    def test_add_list_and_delete_browser_session_with_history(self, backend):
        browser = {
            "browser-id": "functional-test-1",
            "name": "Example dapp",
            "timestamp": 100,
            "dapp?": True,
            "history-index": 1,
            "history": ["https://example.com/page-a", "https://example.com/page-b"],
        }

        assert backend.wakuext_service.get_browsers() == []

        backend.wakuext_service.add_browser(browser)
        browsers = backend.wakuext_service.get_browsers()
        assert len(browsers) == 1
        assert browsers[0]["browser-id"] == browser["browser-id"]
        assert browsers[0]["name"] == browser["name"]
        assert browsers[0]["history"] == browser["history"]
        assert browsers[0]["history-index"] == browser["history-index"]

        updated = {
            **browser,
            "timestamp": 200,
            "history-index": 0,
            "history": ["https://example.com/page-a"],
        }
        backend.wakuext_service.add_browser(updated)
        browsers = backend.wakuext_service.get_browsers()
        assert len(browsers) == 1
        assert browsers[0]["timestamp"] == 200
        assert browsers[0]["history-index"] == 0
        assert browsers[0]["history"] == ["https://example.com/page-a"]

        backend.wakuext_service.delete_browser(browser["browser-id"])
        assert backend.wakuext_service.get_browsers() == []
