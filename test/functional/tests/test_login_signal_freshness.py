import pytest


@pytest.mark.rpc
class TestLoginSignalFreshness:

    @pytest.fixture()
    def backend(self, backend_new_profile):
        return backend_new_profile("account-backend")

    def test_wait_for_login_rejects_a_failed_relogin(self, backend):
        key_uid = backend.key_uid
        backend.logout()
        backend.login(key_uid, "not-the-password")

        with pytest.raises(AssertionError, match="Unexpected error during login"):
            backend.wait_for_login()


@pytest.mark.rpc
@pytest.mark.asyncio
class TestAsyncLoginSignalFreshness:

    async def test_wait_for_login_rejects_a_failed_relogin(self, async_backend_new_profile):
        device = await async_backend_new_profile("account-backend")
        key_uid, password = device.backend.key_uid, device.backend.password

        device.backend.logout()
        device.backend.login(key_uid, password)
        await device.wait_for_login()

        device.backend.logout()
        device.backend.login(key_uid, "not-the-password")

        with pytest.raises(AssertionError, match="Unexpected error during login"):
            await device.wait_for_login()
