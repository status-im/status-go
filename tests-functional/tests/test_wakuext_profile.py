import random

import pytest

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
            ("settings_saveSetting", ["messages-from-contacts-only", True]),
            ("settings_saveSetting", ["notifications-enabled?", True]),
            ("settings_saveSetting", ["send-status-updates?", True]),
            ("settings_saveSetting", ["preview-privacy?", False]),
            ("settings_saveSetting", ["currency", "eth"]),
            ("settings_saveSetting", ["default-sync-period", 259200]),
        ],
    )
    def test_(self, method, params):
        _id = str(random.randint(1, 8888))
        self.rpc_client.rpc_valid_request(method, params, _id)
