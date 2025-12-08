import logging
import pytest
from clients.signals import SignalType
from steps.messenger import MessengerSteps


class TestProfile:

    @pytest.fixture()
    def rpc_client(self, backend_new_profile):
        return backend_new_profile("rpc-client-backend")

    def test_set_display_name(self, rpc_client):
        rpc_client.wakuext_service.set_display_name("new valid username")
        result = rpc_client.settings_service.get_settings()
        assert result.get("display-name") == "new valid username"

    def test_set_bio(self, rpc_client):
        rpc_client.wakuext_service.set_bio("some valid bio")
        result = rpc_client.settings_service.get_settings()
        assert result.get("bio") == "some valid bio"

    def test_set_customization_color(self, rpc_client):
        result = rpc_client.wakuext_service.set_customization_color("magenta", "0xea42dd9a4e668b0b76c7a5210ca81576d51cd19cdd0f6a0c22196219dc423f29")
        assert result is None

    def test_set_user_status(self, rpc_client):
        status_type = 3
        status_text = "test"
        rpc_client.wakuext_service.set_user_status(status_type, status_text)
        result = rpc_client.settings_service.get_settings()
        assert result.get("current-user-status").get("statusType") == status_type
        assert result.get("current-user-status").get("text") == status_text

    def test_set_syncing_on_mobile_network(self, rpc_client):
        rpc_client.wakuext_service.set_syncing_on_mobile_network(False)

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
    def test_settings_(self, rpc_client, setting_name, default_value, changed_value):
        logging.info("Step: check that %s is %s by default " % (setting_name, default_value))
        response = rpc_client.settings_service.get_settings()
        assert response[setting_name] == default_value

        logging.info("Step: change %s to %s and check it is updated" % (setting_name, changed_value))
        # settings_saveSetting -> settings_service.saveSetting
        rpc_client.settings_service.save_setting(setting_name, changed_value)
        response = rpc_client.settings_service.get_settings()
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
    def test_omitempty_false_(self, rpc_client, setting_name, set_value):
        logging.info("Step: assert that %s is not retrieved in settings before setting" % setting_name)
        response = rpc_client.settings_service.get_settings()
        assert setting_name not in response

        logging.info("Step: change %s to %s and check it is updated" % (setting_name, set_value))
        # settings_saveSetting -> settings_service.saveSetting
        rpc_client.settings_service.save_setting(setting_name, set_value)
        response = rpc_client.settings_service.get_settings()
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
    def test_omitempty_true_(self, rpc_client, setting_name, set_value):
        logging.info("Step: assert that %s is  retrieved in settings before unsetting" % setting_name)
        response = rpc_client.settings_service.get_settings()
        assert setting_name in response

        logging.info("Step: change %s to %s and check it is updated and does not retrieve anymore" % (setting_name, set_value))
        # settings_saveSetting -> settings_service.saveSetting
        rpc_client.settings_service.save_setting(setting_name, set_value)
        response = rpc_client.settings_service.get_settings()
        assert setting_name not in response


@pytest.mark.rpc
class TestUserStatus(MessengerSteps):

    @pytest.fixture()
    def sender(self, backend_new_profile):
        return backend_new_profile("sender")

    @pytest.fixture()
    def receiver(self, backend_new_profile):
        return backend_new_profile("receiver")

    def test_status_updates(self, sender, receiver):
        self.make_contacts(sender=sender, receiver=receiver)

        statuses = [[1, "text_1"], [2, "text_2"], [3, "text_3"], [4, "text_4"]]

        for new_status, custom_text in statuses:
            response = sender.wakuext_service.set_user_status(new_status, custom_text)
            # TODO: Add more assertions on response

            receiver.find_signal_containing_pattern(
                SignalType.MESSAGES_NEW.value,
                event_pattern=custom_text,
                timeout=10,
            )

            response = receiver.wakuext_service.status_updates()
            # TODO: Add more assertions on response

            status_update = response.get("statusUpdates", [])[0]
            assert status_update.get("statusType", -1) == new_status
            assert status_update.get("text", "") == custom_text
