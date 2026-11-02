import json

import pytest


def browser_settings_category(public_key: str) -> str:
    return f"BrowserSettings_{public_key}"


def valid_keys_for_open_tabs(open_tabs_json: str) -> list:
    keys = ["restoreOpenTabs", "openTabs", "activeTabIndex"]
    tabs = json.loads(open_tabs_json)
    for tab in tabs:
        keys.append(f"snapshot/{tab['uuid']}")
    return keys


@pytest.mark.rpc
@pytest.mark.browser
class TestBrowserPreferences:

    @pytest.fixture()
    def backend(self, backend_new_profile):
        return backend_new_profile("browser-preferences")

    def test_set_get_and_purge_browser_tab_preferences(self, backend):
        category = browser_settings_category(backend.public_key)
        open_tabs = json.dumps([{"uuid": "tab-1", "url": "https://example.com", "title": "Example", "icon": ""}])

        backend.preferences_service.set(category, "restoreOpenTabs", "true")
        backend.preferences_service.set(category, "openTabs", open_tabs)
        backend.preferences_service.set(category, "snapshot/tab-1", "snapshot-data")
        backend.preferences_service.set(category, "activeTabIndex", "0")
        backend.preferences_service.set(category, "currentTabIndex", "2")

        valid_keys = valid_keys_for_open_tabs(open_tabs)
        all_prefs = backend.preferences_service.load_and_purge_unknown(category, valid_keys)

        assert all_prefs["restoreOpenTabs"] == "true"
        assert json.loads(all_prefs["openTabs"]) == json.loads(open_tabs)
        assert all_prefs["snapshot/tab-1"] == "snapshot-data"
        assert all_prefs["activeTabIndex"] == "0"
        assert "currentTabIndex" not in all_prefs

        got = backend.preferences_service.get(category, "restoreOpenTabs")
        assert got["found"] is True
        assert got["value"] == "true"

        backend.preferences_service.set(category, "snapshot/orphan", "stale")
        purge_result = backend.preferences_service.purge_unknown(category, valid_keys)
        assert purge_result["removed"] == 1
        orphan = backend.preferences_service.get(category, "snapshot/orphan")
        assert orphan["found"] is False

        assert backend.preferences_service.get(category, "restoreOpenTabs")["found"] is True
