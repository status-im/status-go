import logging
import random
import string

import pytest

from clients.signals import SignalType
from resources.constants import Account


@pytest.mark.rpc
class TestPassword:

    await_signals = [
        SignalType.NODE_LOGIN.value,
        SignalType.DB_REENCRYPTION_STARTED.value,
        SignalType.DB_REENCRYPTION_FINISHED.value,
        SignalType.NODE_STARTED.value,
        SignalType.NODE_READY.value,
        SignalType.NODE_STOPPED.value,
    ]

    def test_verify_correct_password(self, backend_new_profile):
        backend = backend_new_profile("sender")
        response = backend.accounts_service.verify_password(backend.password)
        assert response.get("result") is True

    @pytest.mark.parametrize("password", ["testpassword", ""])
    def test_verify_wrong_password(self, password, backend_new_profile):
        backend = backend_new_profile("sender")
        response = backend.accounts_service.verify_password(password)
        assert response.get("result") is False

    def test_change_password(self, backend_new_profile):
        new_password = "".join(random.choices(string.ascii_letters + string.digits, k=8))
        backend = backend_new_profile("user")

        # Try a wrong password
        response = backend.change_database_password(backend.password + "-wrong", new_password)
        assert response.get("error") is not None
        assert response.get("error") == "incorrect current password"

        # Try a correct password
        response = backend.change_database_password(backend.password, new_password)
        assert response.get("error") == ""
        backend.wait_for_signal(SignalType.DB_REENCRYPTION_STARTED.value)
        backend.wait_for_signal(SignalType.DB_REENCRYPTION_FINISHED.value)
        backend.wait_for_signal(SignalType.NODE_STOPPED.value)
        backend.wait_for_signal(SignalType.NODE_STARTED.value)
        backend.wait_for_signal(SignalType.NODE_READY.value)

        backend.logout()
        backend.wait_for_signal(SignalType.NODE_STOPPED.value)

        # Try login with the old password
        account = Account(
            password=backend.password,
            address="",
            private_key="",
            passphrase="",
        )
        logging.info(f"Logging in with old password: {backend.password}, key uid: {backend.key_uid}")
        backend.login(backend.key_uid, account)
        signal = backend.wait_for_signal(SignalType.NODE_LOGIN.value)
        event = signal.get("event")
        assert "error" in event
        assert "failed to open database" in event.get("error")

        # Login with the new password
        backend.password = new_password
        account = Account(
            password=new_password,
            address="",
            private_key="",
            passphrase="",
        )
        backend.login(backend.key_uid, account)
        backend.wait_for_login()
