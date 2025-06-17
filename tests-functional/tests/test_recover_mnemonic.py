from clients.status_backend import StatusBackend
import pytest
from clients.signals import SignalType
import logging

from resources.constants import user_mnemonic_12, user_mnemonic_15, user_mnemonic_24
from resources.utils import assert_account_attributes


@pytest.mark.create_account
@pytest.mark.rpc
class TestRecoverMnemonic:
    @pytest.mark.parametrize(
        "user_mnemonic",
        [user_mnemonic_24, user_mnemonic_15, user_mnemonic_12],
        ids=["mnemonic_24", "mnemonic_15", "mnemonic_12"],
    )
    def test_recover_app_different_mnemonic(self, user_mnemonic):
        await_signals = [
            SignalType.MEDIASERVER_STARTED.value,
            SignalType.NODE_STARTED.value,
            SignalType.NODE_READY.value,
            SignalType.NODE_LOGIN.value,
        ]

        logging.info("Step: Initialize backend client and restore account using user_mnemonic.")
        backend_client = StatusBackend(await_signals)
        backend_client.init_status_backend()
        backend_client.restore_account_and_login(user=user_mnemonic)

        logging.info("Step: request getAccounts and check recovered accounts attributes.")
        response = backend_client.rpc_valid_request("accounts_getAccounts", [])
        assert_account_attributes(response.json()["result"], user_mnemonic.accounts)

        logging.info("Step: request getSettings and check recovered profile data attributes.")
        response = backend_client.rpc_valid_request("settings_getSettings", [])
        assert_account_attributes(response.json()["result"], user_mnemonic.profile_data)
