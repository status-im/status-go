"""wakuext_signData refuses accounts whose key has moved to a cold wallet, because the card has to sign.

The guard sits between the profile/watch-only refusal and the password check
(internal/protocol/messenger_communities.go:1424-1437), and the suite only ever drove this call with
regular accounts. These tests pin the refusal, its position in that order, and that one bad account
fails the whole batch.
"""

import pytest
from clients.api import ApiResponseError
from resources.constants import (
    keypair_name,
    user_1,
    wallet_account_details_derivation,
)

COLD_WALLET_ERROR = "signing a joining community request for accounts migrated to a cold wallet must be done with the cold wallet"
PROFILE_OR_WATCH_ERROR = "cannot join a community using profile chat or watch-only account"
# Any hex payload will do — SignData text-hashes it before signing.
DATA = "0x" + "11" * 32


@pytest.mark.rpc
class TestSignDataColdWalletGuard:

    @pytest.fixture()
    def backend(self, backend_new_profile):
        return backend_new_profile("sign-data-cold")

    def _add_seed_keypair(self, backend):
        response = backend.accounts_service.add_keypair_via_seed_phrase(
            user_1.passphrase,
            backend.password,
            keypair_name,
            "",
            wallet_account_details_derivation,
        )
        key_uid = response.get("key-uid")
        assert key_uid, "Expected addKeypairViaSeedPhrase to return the created keypair"
        return key_uid, response["accounts"][0]["address"]

    def _profile_account(self, backend, chat):
        kp = backend.accounts_service.get_keypair_by_key_uid(backend.key_uid)
        account = next((a for a in kp["accounts"] if bool(a.get("chat")) is chat), None)
        assert account is not None, f"Expected the profile keypair to own a chat={chat} account"
        return account["address"]

    def _sign(self, backend, addresses, password=None):
        params = [{"data": DATA, "account": address, "password": backend.password if password is None else password} for address in addresses]
        return backend.wakuext_service.sign_data(params)

    def test_sign_data_refuses_a_cold_wallet_account(self, backend):
        key_uid, address = self._add_seed_keypair(backend)

        before = self._sign(backend, [address])
        assert len(before) == 1 and before[0].startswith("0x"), "Expected a regular keypair account to sign"

        backend.accounts_service.migrate_non_profile_keypair_to_cold_wallet(key_uid, backend.password, "status-keycard")

        with pytest.raises(ApiResponseError, match=COLD_WALLET_ERROR):
            self._sign(backend, [address])

        backend.accounts_service.migrate_non_profile_cold_wallet_keypair_to_app(user_1.passphrase, backend.password)

        after = self._sign(backend, [address])
        assert after == before, "Expected the same key to sign the same payload identically once it is back in the app keystore"

    def test_the_cold_wallet_refusal_precedes_the_password_check(self, backend):
        key_uid, address = self._add_seed_keypair(backend)
        backend.accounts_service.migrate_non_profile_keypair_to_cold_wallet(key_uid, backend.password, "status-keycard")

        # A wrong password must still surface the cold-wallet refusal: the guard runs first, so the
        # caller is told to use the card rather than that their password was wrong.
        with pytest.raises(ApiResponseError, match=COLD_WALLET_ERROR):
            self._sign(backend, [address], password="definitely-wrong")

    def test_sign_data_refuses_the_profile_chat_account(self, backend):
        # The sibling guard, one branch earlier — a different refusal reached through the same call.
        with pytest.raises(ApiResponseError, match=PROFILE_OR_WATCH_ERROR):
            self._sign(backend, [self._profile_account(backend, chat=True)])

    def test_one_cold_wallet_account_fails_the_whole_batch(self, backend):
        key_uid, cold_address = self._add_seed_keypair(backend)
        profile_address = self._profile_account(backend, chat=False)

        assert len(self._sign(backend, [profile_address, cold_address])) == 2, "Expected both accounts to sign before the migration"

        backend.accounts_service.migrate_non_profile_keypair_to_cold_wallet(key_uid, backend.password, "status-keycard")

        with pytest.raises(ApiResponseError, match=COLD_WALLET_ERROR):
            self._sign(backend, [profile_address, cold_address])

        assert len(self._sign(backend, [profile_address])) == 1, "Expected the untouched account to keep signing on its own"
