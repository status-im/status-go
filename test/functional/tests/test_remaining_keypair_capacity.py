import copy
import re
import pytest
from clients.api import ApiResponseError
from resources.constants import (
    user_1,
    new_account_data_1,
    wallet_account_details_root,
    wallet_account_details_derivation,
    keypair_name,
)
import secrets


@pytest.mark.rpc
class TestRemainingKeypairCapacity:

    @pytest.fixture()
    def backend(self, backend_new_profile):
        return backend_new_profile("account-backend")

    @pytest.fixture()
    def account_data(self):
        return copy.deepcopy(new_account_data_1)

    def test_remaining_keypair_capacity_decreases_on_add(self, backend):
        # 1) Query remaining capacity
        initial_capacity = backend.accounts_service.remaining_keypair_capacity()
        assert initial_capacity == 4

        # 2) Add a keypair via private key
        backend.accounts_service.add_keypair_via_private_key(
            user_1.private_key,
            backend.password,
            keypair_name,
            wallet_account_details_root,
        )

        # 3) Query remaining capacity again and assert it decreased by exactly 1
        after_capacity = backend.accounts_service.remaining_keypair_capacity()
        assert after_capacity == initial_capacity - 1

        # 4) Add a keypair via seed phrase
        backend.accounts_service.add_keypair_via_seed_phrase(
            user_1.passphrase,
            backend.password,
            keypair_name,
            "",
            wallet_account_details_derivation,
        )

        # 5) Query remaining capacity again and assert it decreased by exactly 1
        after_capacity2 = backend.accounts_service.remaining_keypair_capacity()
        assert after_capacity2 == after_capacity - 1

    def test_no_more_keypairs_can_be_added(self, backend):
        initial_capacity = backend.accounts_service.remaining_keypair_capacity()
        words = ["alpha", "bravo", "zulu", "test", "key"]
        for i in range(initial_capacity):
            keypair_name = f"stress-kp-{i}-{secrets.token_hex(4)}"
            passphrase = " ".join(secrets.choice(words) for _ in range(12))
            backend.accounts_service.add_keypair_via_seed_phrase(
                passphrase,
                backend.password,
                keypair_name,
                "",
                {
                    "name": keypair_name,
                    "path": f"m/44'/60'/0'/0/{i}",
                    "emoji": "🔑",
                    "colorId": "primary",
                },
            )

        get_keypairs_response = backend.accounts_service.get_account_keypairs()
        assert len(get_keypairs_response) == 5

        with pytest.raises(ApiResponseError, match=re.escape("no more keypairs can be added")):
            backend.accounts_service.remaining_keypair_capacity()
