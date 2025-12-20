import logging
import re

import pytest

from clients.api import ApiResponseError
from clients.signals import SignalType
from utils import fake


@pytest.mark.rpc
class TestPassword:

    def test_verify_correct_password(self, backend_new_profile):
        backend = backend_new_profile("sender")
        response = backend.accounts_service.verify_password(backend.password)
        assert response is True

    @pytest.mark.parametrize("password", ["testpassword", ""])
    def test_verify_wrong_password(self, password, backend_new_profile):
        backend = backend_new_profile("sender")
        response = backend.accounts_service.verify_password(password)
        assert response is False

    def test_change_password(self, backend_new_profile):
        new_password = fake.profile_password(8)
        backend = backend_new_profile("user")

        # Try a wrong password
        with pytest.raises(ApiResponseError, match=re.escape("incorrect current password")):
            backend.change_database_password(backend.password + "-wrong", new_password)

        # Try a correct password
        with backend.expect_signals_sequence(
            [
                SignalType.DB_REENCRYPTION_STARTED,
                SignalType.DB_REENCRYPTION_FINISHED,
                SignalType.NODE_STOPPED,
                SignalType.NODE_STARTED,
                SignalType.NODE_READY,
            ]
        ):
            backend.change_database_password(backend.password, new_password)

        with backend.expect_signal(SignalType.NODE_STOPPED):
            backend.logout()

        # Try login with the old password
        logging.info(f"Logging in with old password: {backend.password}, key uid: {backend.key_uid}")
        with backend.expect_signal(SignalType.NODE_LOGIN) as exp:
            backend.login(backend.key_uid, backend.password)
        signal = exp.result
        event = signal.get("event")
        assert "error" in event
        assert "failed to open database" in event.get("error")

        # Login with the new password
        with backend.expect_signal(SignalType.NODE_LOGIN) as exp:
            backend.login(backend.key_uid, new_password)
        signal = exp.result
        # Verify successful login (no error)
        event = signal.get("event")
        assert "error" not in event or not event.get("error")
