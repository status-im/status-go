import pytest


@pytest.mark.rpc
@pytest.mark.browser
class TestBrowserBookmarks:

    @pytest.fixture()
    def backend(self, backend_new_profile):
        return backend_new_profile("browser-bookmarks")

    def test_store_list_update_remove_and_delete_bookmark(self, backend):
        url = "https://example.com/status-test"
        name = "Status test bookmark"

        assert (backend.browsers_service.get_bookmarks() or []) == []

        stored = backend.browsers_service.store_bookmark({"url": url, "name": name})
        assert stored["url"] == url
        assert stored["name"] == name

        bookmarks = backend.browsers_service.get_bookmarks() or []
        assert len(bookmarks) == 1
        assert bookmarks[0]["url"] == url
        assert bookmarks[0]["name"] == name
        assert bookmarks[0]["removed"] is False

        new_url = "https://example.com/status-renamed"
        new_name = "Renamed"
        backend.browsers_service.update_bookmark(url, {"url": new_url, "name": new_name})

        bookmarks = backend.browsers_service.get_bookmarks() or []
        assert len(bookmarks) == 1
        assert bookmarks[0]["url"] == new_url
        assert bookmarks[0]["name"] == new_name

        backend.browsers_service.remove_bookmark(new_url)
        bookmarks = backend.browsers_service.get_bookmarks() or []
        assert len(bookmarks) == 1
        assert bookmarks[0]["removed"] is True

        backend.browsers_service.delete_bookmark(new_url)
        assert (backend.browsers_service.get_bookmarks() or []) == []
