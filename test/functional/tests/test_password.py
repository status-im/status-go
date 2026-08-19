import logging
import re

import pytest

from clients.api import ApiResponseError
from clients.signals import SignalType
from utils import fake


@pytest.mark.rpc
@pytest.mark.asyncio
class TestPassword:

    async def test_verify_correct_password(self, async_backend_new_profile):
        backend = await async_backend_new_profile("sender")
        response = backend.backend.accounts_service.verify_password(backend.backend.password)
        assert response is True

    @pytest.mark.parametrize("password", ["testpassword", ""])
    async def test_verify_wrong_password(self, password, async_backend_new_profile):
        backend = await async_backend_new_profile("sender")
        response = backend.backend.accounts_service.verify_password(password)
        assert response is False

    async def test_change_password(self, async_backend_new_profile):
        new_password = fake.profile_password(8)
        backend = await async_backend_new_profile("user")

        # Try a wrong password
        with pytest.raises(ApiResponseError, match=re.escape("incorrect current password")):
            backend.backend.change_database_password(backend.backend.password + "-wrong", new_password)

        # Try a correct password - profiles are on the DEK scheme, so this is a fast re-wrap of the profile
        # envelope: no re-encryption signals and no node restart are expected
        backend.backend.change_database_password(backend.backend.password, new_password)

        # Logout
        backend.backend.logout()
        await backend.wait_for_signal(SignalType.NODE_STOPPED, timeout=60, check_buffer=True)

        # Try login with the old password
        logging.info(f"Logging in with old password: {backend.backend.password}, key uid: {backend.backend.key_uid}")
        backend.backend.login(backend.backend.key_uid, backend.backend.password)
        signal = await backend.wait_for_signal(SignalType.NODE_LOGIN, timeout=60, check_buffer=True)
        event = signal.raw.get("event")
        assert "error" in event
        assert "failed to open database" in event.get("error")

        # Login with the new password - use after_seq to skip the previous signal
        backend.backend.login(backend.backend.key_uid, new_password)
        signal = await backend.wait_for_signal(SignalType.NODE_LOGIN, timeout=60, check_buffer=True, after_seq=signal.seq)
        # Verify successful login (no error)
        event = signal.raw.get("event")
        assert "error" not in event or not event.get("error")

    async def test_change_password_with_rekey(self, async_backend_new_profile):
        new_password = fake.profile_password(8)
        backend = await async_backend_new_profile("user")

        # A deep rekey generates a fresh data-encryption key and re-encrypts the databases and
        # keystore with it. The node is logged out mid-rekey (to release the database files before
        # swapping them) and restarted once the rekey is done, hence NODE_STOPPED arrives before
        # DB_REENCRYPTION_FINISHED.
        backend.backend.change_database_password(backend.backend.password, new_password, rekey=True)

        # Wait for all signals in sequence (check_buffer=True to find signals that arrived during RPC)
        signals = await backend.wait_for_signals_sequence(
            [
                SignalType.DB_REENCRYPTION_STARTED,
                SignalType.NODE_STOPPED,
                SignalType.DB_REENCRYPTION_FINISHED,
                SignalType.NODE_STARTED,
                SignalType.NODE_READY,
            ],
            timeout=120,
        )

        # Logout - use after_seq to skip the signals emitted by the rekey restart (including its
        # successful node.login)
        backend.backend.logout()
        signal = await backend.wait_for_signal(SignalType.NODE_STOPPED, timeout=60, check_buffer=True, after_seq=signals[-1].seq)

        # Try login with the old password
        backend.backend.login(backend.backend.key_uid, backend.backend.password)
        signal = await backend.wait_for_signal(SignalType.NODE_LOGIN, timeout=60, check_buffer=True, after_seq=signal.seq)
        event = signal.raw.get("event")
        assert "error" in event
        assert "failed to open database" in event.get("error")

        # Login with the new password - use after_seq to skip the previous signal
        backend.backend.login(backend.backend.key_uid, new_password)
        signal = await backend.wait_for_signal(SignalType.NODE_LOGIN, timeout=60, check_buffer=True, after_seq=signal.seq)
        # Verify successful login (no error)
        event = signal.raw.get("event")
        assert "error" not in event or not event.get("error")
