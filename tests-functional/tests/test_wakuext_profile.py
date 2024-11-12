import random

import pytest
import logging

from test_cases import StatusBackendTestCase


class TestProfile(StatusBackendTestCase):
    @pytest.mark.parametrize(
        "method, params",
        [
            ("wakuext_setDisplayName", ["new valid username"]),
            ("wakuext_setBio", ["some valid bio"]),
            ("wakuext_setCustomizationColor", [{'customizationColor': 'magenta',
                                                'keyUid': '0xea42dd9a4e668b0b76c7a5210ca81576d51cd19cdd0f6a0c22196219dc423f29'}]),
            ("wakuext_setUserStatus", [3, ""]),
            ("wakuext_setSyncingOnMobileNetwork", [{"enabled": False}]),
            ("wakuext_togglePeerSyncing", [{"enabled": True}]),
            ("wakuext_backupData", []),
        ],
    )
    def test_(self, method, params):
        _id = str(random.randint(1, 8888))
        self.rpc_client.rpc_valid_request(method, params, _id)

    @pytest.mark.parametrize(
        "method, setting_name, default_value, changed_value",
        [
            ("settings_saveSetting", "currency", "usd", "eth"),
            ("settings_saveSetting", "messages-from-contacts-only", False, True),
            ("settings_saveSetting", "preview-privacy?", False, True),
            ("settings_saveSetting", "default-sync-period", 777600, 259200),
            ("settings_saveSetting", "appearance", 0, 1),
            ("settings_saveSetting", "profile-pictures-show-to", 2, 1),  # obsolete from v1
            ("settings_saveSetting", "profile-pictures-visibility", 2, 1),  # obsolete from v1
        ],
    )
    def test_(self, method, setting_name, default_value, changed_value):
        _id = str(random.randint(1, 8888))

        logging.info("Step: check that %s is %s by default " % (setting_name, default_value))
        response = self.rpc_client.rpc_valid_request("settings_getSettings", [])
        assert response.json()["result"][setting_name] == default_value

        logging.info("Step: change %s to %s and check it is updated" % (setting_name, changed_value))
        self.rpc_client.rpc_valid_request(method, [setting_name, changed_value], _id)
        response = self.rpc_client.rpc_valid_request("settings_getSettings", [])
        assert response.json()["result"][setting_name] == changed_value

    # tests for `omitempty` params that are set to False or nil by default
    @pytest.mark.parametrize(
        "method, setting_name, set_value",
        [
            ("settings_saveSetting", "mnemonic-removed?", True),
            ("settings_saveSetting", "push-notifications-server-enabled?", True),
            ("settings_saveSetting", "push-notifications-from-contacts-only?", True),
            ("settings_saveSetting", "push-notifications-block-mentions?", True),
            ("settings_saveSetting", "remember-syncing-choice?", True),
            ("settings_saveSetting", "remote-push-notifications-enabled?", True),
            ("settings_saveSetting", "syncing-on-mobile-network?", True),
            ## advanced token settings
            ("settings_saveSetting", "wallet-set-up-passed?", True),
            ("settings_saveSetting", "opensea-enabled?", True),
            ("settings_saveSetting", "waku-bloom-filter-mode", True),
            ("settings_saveSetting", "webview-allow-permission-requests?", True),
            ("settings_saveSetting", "token-group-by-community?", True),
            ("settings_saveSetting", "display-assets-below-balance?", True),
            ## token management settings for collectibles
            ("settings_saveSetting", "collectible-group-by-collection?", True),
            ("settings_saveSetting", "collectible-group-by-community?", True),
        ],
    )
    def test_omitempty_false_(self, method, setting_name, set_value):
        _id = str(random.randint(1, 8888))

        logging.info("Step: assert that %s is not retrieved in settings before setting" % (setting_name))
        response = self.rpc_client.rpc_valid_request("settings_getSettings", [])
        assert setting_name not in response.json()["result"]

        logging.info("Step: change %s to %s and check it is updated" % (setting_name, set_value))
        self.rpc_client.rpc_valid_request(method, [setting_name, set_value], _id)
        response = self.rpc_client.rpc_valid_request("settings_getSettings", [])
        assert response.json()["result"][setting_name] == set_value

    # tests for `omitempty` params that are not nil by default
    @pytest.mark.parametrize(
        "method, setting_name, set_value",
        [
            ("settings_saveSetting", "send-status-updates?", False),
            ("settings_saveSetting", "link-preview-request-enabled", False),
            ("settings_saveSetting", "show-community-asset-when-sending-tokens?", False),
            ("settings_saveSetting", "url-unfurling-mode", 0),
        ],
    )
    def test_omitempty_true_(self, method, setting_name, set_value):
        _id = str(random.randint(1, 8888))

        logging.info("Step: assert that %s is  retrieved in settings before unsetting" % (setting_name))
        response = self.rpc_client.rpc_valid_request("settings_getSettings", [])
        assert setting_name in response.json()["result"]

        logging.info("Step: change %s to %s and check it is updated and does not retrieve anymore" % (setting_name, set_value))
        self.rpc_client.rpc_valid_request(method, [setting_name, set_value], _id)
        response = self.rpc_client.rpc_valid_request("settings_getSettings", [])
        assert setting_name not in response.json()["result"]

