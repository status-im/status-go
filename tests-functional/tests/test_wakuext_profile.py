import logging

import pytest

from clients.signals import SignalType
from steps.messenger import MessengerSteps


class TestProfile:

    @pytest.fixture(autouse=True)
    def setup_backend(self, backend_new_profile):
        """Initialize one backend for each test function"""
        self.rpc_client = backend_new_profile("rpc_client")

    @pytest.mark.parametrize(
        "method, params",
        [
            ("wakuext_setDisplayName", ["new valid username"]),
            ("wakuext_setBio", ["some valid bio"]),
            (
                "wakuext_setCustomizationColor",
                [
                    {
                        "customizationColor": "magenta",
                        "keyUid": "0xea42dd9a4e668b0b76c7a5210ca81576d51cd19cdd0f6a0c22196219dc423f29",
                    }
                ],
            ),
            ("wakuext_setUserStatus", [3, ""]),
            ("wakuext_setSyncingOnMobileNetwork", [{"enabled": False}]),
            ("wakuext_togglePeerSyncing", [{"enabled": True}]),
            ("wakuext_backupData", []),
        ],
    )
    def test_wakuext_(self, method, params):
        # TODO: Break this down into individual tests and implment the coresponding wakuext methods
        self.rpc_client.rpc_valid_request(method, params)

    @pytest.mark.parametrize(
        "setting_name, default_value, changed_value",
        [
            ("currency", "usd", "eth"),
            ("messages-from-contacts-only", False, True),
            ("preview-privacy?", False, True),
            ("default-sync-period", 777600, 259200),
            ("appearance", 0, 1),
            (
                "profile-pictures-show-to",
                2,
                1,
            ),  # obsolete from v1
            (
                "profile-pictures-visibility",
                2,
                1,
            ),  # obsolete from v1
        ],
    )
    def test_settings_(self, setting_name, default_value, changed_value):
        logging.info("Step: check that %s is %s by default " % (setting_name, default_value))
        response = self.rpc_client.settings_service.get_settings()
        assert response[setting_name] == default_value

        logging.info("Step: change %s to %s and check it is updated" % (setting_name, changed_value))
        # settings_saveSetting -> settings_service.saveSetting
        self.rpc_client.settings_service.save_setting(setting_name, changed_value)
        response = self.rpc_client.settings_service.get_settings()
        assert response[setting_name] == changed_value

    # tests for `omitempty` params that are set to False or nil by default
    @pytest.mark.parametrize(
        "setting_name, set_value",
        [
            ("mnemonic-removed?", True),
            ("push-notifications-from-contacts-only?", True),
            ("push-notifications-block-mentions?", True),
            ("remember-syncing-choice?", True),
            ("remote-push-notifications-enabled?", True),
            ("syncing-on-mobile-network?", True),
            # advanced token settings
            ("wallet-set-up-passed?", True),
            ("opensea-enabled?", True),
            ("waku-bloom-filter-mode", True),
            ("webview-allow-permission-requests?", True),
            ("token-group-by-community?", True),
            ("display-assets-below-balance?", True),
            # token management settings for collectibles
            ("collectible-group-by-collection?", True),
            ("collectible-group-by-community?", True),
        ],
    )
    def test_omitempty_false_(self, setting_name, set_value):
        logging.info("Step: assert that %s is not retrieved in settings before setting" % (setting_name))
        response = self.rpc_client.settings_service.get_settings()
        assert setting_name not in response

        logging.info("Step: change %s to %s and check it is updated" % (setting_name, set_value))
        # settings_saveSetting -> settings_service.saveSetting
        self.rpc_client.settings_service.rpc_request("saveSetting", [setting_name, set_value])
        response = self.rpc_client.settings_service.get_settings()
        assert response[setting_name] == set_value

    # tests for `omitempty` params that are not nil by default
    @pytest.mark.parametrize(
        "setting_name, set_value",
        [
            ("send-status-updates?", False),
            ("link-preview-request-enabled", False),
            (
                "show-community-asset-when-sending-tokens?",
                False,
            ),
            ("url-unfurling-mode", 0),
        ],
    )
    def test_omitempty_true_(self, setting_name, set_value):
        logging.info("Step: assert that %s is  retrieved in settings before unsetting" % (setting_name))
        response = self.rpc_client.settings_service.get_settings()
        assert setting_name in response

        logging.info("Step: change %s to %s and check it is updated and does not retrieve anymore" % (setting_name, set_value))
        # settings_saveSetting -> settings_service.saveSetting
        self.rpc_client.settings_service.save_setting(setting_name, set_value)
        response = self.rpc_client.settings_service.get_settings()
        assert setting_name not in response


@pytest.mark.rpc
class TestUserStatus(MessengerSteps):

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        """Initialize two unprivileged backends (sender and receiver) for each test function"""
        self.sender = backend_new_profile("sender")
        self.receiver = backend_new_profile("receiver")

    def test_status_updates(self):
        self.make_contacts(sender=self.sender, receiver=self.receiver)

        statuses = [[1, "text_1"], [2, "text_2"], [3, "text_3"], [4, "text_4"]]

        for new_status, custom_text in statuses:
            response = self.sender.wakuext_service.set_user_status(new_status, custom_text)
            # TODO: Add more assertions on response

            self.receiver.find_signal_containing_pattern(
                SignalType.MESSAGES_NEW.value,
                event_pattern=custom_text,
                timeout=10,
            )

            response = self.receiver.wakuext_service.status_updates()
            # TODO: Add more assertions on response

            statusUpdate = response.get("statusUpdates", [])[0]
            assert statusUpdate.get("statusType", -1) == new_status
            assert statusUpdate.get("text", "") == custom_text
