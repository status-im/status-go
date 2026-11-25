import re

import pytest

from clients.api import ApiResponseError
from utils import fake


def _account_by_key_uid(accounts, key_uid):
    return next(a for a in accounts if a.get("key-uid") == key_uid)


@pytest.mark.rpc
class TestZeroOnboardedProfile:

    def test_zero_onboarded_profile_becomes_regular_after_set_password(self, backend_new_profile):
        backend = backend_new_profile("user", zero_onboarded=True)
        key_uid = backend.key_uid
        generated_password = backend.password

        assert backend.node_login_event.get("event", {}).get("account", {}).get("zeroOnboarded") is True
        account = _account_by_key_uid(backend.reinit_and_get_accounts(), key_uid)
        assert account.get("zeroOnboarded") is True

        backend.login(key_uid, generated_password)
        backend.wait_for_login()

        new_password = fake.profile_password(8)
        backend.set_zero_onboarded_profile_password(new_password)

        with pytest.raises(ApiResponseError, match=re.escape("profile is not zero-onboarded")):
            backend.set_zero_onboarded_profile_password(fake.profile_password(8))

        account = _account_by_key_uid(backend.reinit_and_get_accounts(), key_uid)
        assert not account.get("zeroOnboarded")

        backend.login(key_uid, new_password)
        backend.wait_for_login()

    def test_set_password_refused_for_regular_profile(self, backend_new_profile):
        backend = backend_new_profile("user")

        with pytest.raises(ApiResponseError, match=re.escape("profile is not zero-onboarded")):
            backend.set_zero_onboarded_profile_password(fake.profile_password(8))

    def test_zero_onboarded_creation_refused_when_a_profile_exists(self, backend_new_profile):
        backend = backend_new_profile("user")
        backend.logout()

        backend.create_account_and_login(password=fake.profile_password(), zero_onboarded=True)
        with pytest.raises(AssertionError, match=re.escape("zero-onboarded profile can only be created")):
            backend.wait_for_login()
