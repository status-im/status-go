"""The three operability RPCs that decide whether a keypair's keys are on disk, all of which branch on cold-wallet state:
makeSeedPhraseKeypairFullyOperable writes keystore files back from a mnemonic, makePartiallyOperableAccoutsFullyOperable
sweeps accounts that were added without a password, and cleanKeystoreFiles deletes files again."""

import copy

import pytest
from clients.api import ApiResponseError
from resources.constants import user_1, user_mnemonic_12
from steps.cold_wallet import add_seed_keypair, keystore_present

NEXT_PATH = "m/44'/60'/0'/0/1"


@pytest.mark.rpc
class TestColdKeypairOperability:

    @pytest.fixture()
    def backend(self, backend_new_profile):
        return backend_new_profile("operability")

    def _keystore_present(self, backend, key_uid):
        return keystore_present(backend, backend.accounts_service.get_keypair_by_key_uid(key_uid))

    def _add_account_without_a_password(self, backend, key_uid, mnemonic, path=NEXT_PATH):
        """An account added with no password is derived from the stored xpub, so it lands partially operable."""
        keypair = backend.accounts_service.get_keypair_by_key_uid(key_uid)
        derived = backend.wallet_service.get_derived_addresses_for_mnemonic(mnemonic, [path])[0]
        template = copy.deepcopy(keypair["accounts"][0])
        template.update(
            {
                "address": derived["address"],
                "public-key": derived["public-key"],
                "path": path,
                "name": "added-without-password",
                "wallet": False,
                "chat": False,
            }
        )
        backend.accounts_service.add_account("", template)
        return derived["address"]

    def _account(self, backend, key_uid, address):
        keypair = backend.accounts_service.get_keypair_by_key_uid(key_uid)
        account = next((a for a in keypair["accounts"] if a["address"].lower() == address.lower()), None)
        assert account is not None, f"Expected {address} to be an account of {key_uid}"
        return account

    def test_the_sweep_promotes_an_account_added_without_a_password(self, backend):
        key_uid = add_seed_keypair(backend)["key-uid"]
        address = self._add_account_without_a_password(backend, key_uid, user_1.passphrase)

        assert self._account(backend, key_uid, address)["operable"] == "partially", "Expected an xpub-derived account to start partially operable"
        assert (
            backend.accounts_service.verify_keystore_file_for_account(address, backend.password) is False
        ), "Expected no keystore file, which is what partially operable means"

        promoted = backend.accounts_service.make_partially_operable_accouts_fully_operable(backend.password)

        assert [a.lower() for a in promoted] == [address.lower()], "Expected the sweep to report exactly the account it promoted"
        assert self._account(backend, key_uid, address)["operable"] == "fully"
        assert (
            backend.accounts_service.verify_keystore_file_for_account(address, backend.password) is True
        ), "Expected the sweep to write the keystore file it derived"

    def test_the_sweep_skips_cold_wallet_keypairs(self, backend):
        app_key_uid = add_seed_keypair(backend)["key-uid"]
        cold_key_uid = add_seed_keypair(backend, user_mnemonic_12, name="cold-keypair")["key-uid"]
        app_address = self._add_account_without_a_password(backend, app_key_uid, user_1.passphrase)

        backend.accounts_service.migrate_non_profile_keypair_to_cold_wallet(cold_key_uid, backend.password, "status-keycard")
        assert not any(self._keystore_present(backend, cold_key_uid).values()), "Expected migration to have deleted the cold keypair's files"

        promoted = backend.accounts_service.make_partially_operable_accouts_fully_operable(backend.password)

        assert [a.lower() for a in promoted] == [app_address.lower()], "Expected only the app-side account to be promoted"
        assert not any(
            self._keystore_present(backend, cold_key_uid).values()
        ), "Expected the sweep to leave the cold keypair alone, because its key is on the card"

    def test_the_sweep_requires_a_password(self, backend):
        key_uid = add_seed_keypair(backend)["key-uid"]
        address = self._add_account_without_a_password(backend, key_uid, user_1.passphrase)

        with pytest.raises(ApiResponseError, match="no password provided"):
            backend.accounts_service.make_partially_operable_accouts_fully_operable("")

        assert self._account(backend, key_uid, address)["operable"] == "partially", "Expected the rejected sweep to promote nothing"

    def test_make_seed_phrase_keypair_fully_operable_restores_files_but_leaves_the_keypair_cold(self, backend):
        key_uid = add_seed_keypair(backend)["key-uid"]

        backend.accounts_service.migrate_non_profile_keypair_to_cold_wallet(key_uid, backend.password, "status-keycard")
        assert not any(self._keystore_present(backend, key_uid).values())

        backend.accounts_service.make_seed_phrase_keypair_fully_operable(user_1.passphrase, backend.password)

        present = self._keystore_present(backend, key_uid)
        assert all(present.values()), f"Expected every file back, the master address included: {present}"
        # Unlike migrateNonProfileColdWalletKeypairToApp this call does not clear cold_wallet, so the
        # keypair now claims to be on a card while its keys sit on disk.
        assert (
            backend.accounts_service.get_keypair_by_key_uid(key_uid)["cold-wallet"] == "status-keycard"
        ), "Characterises current behaviour: this call writes keystore files without clearing cold-wallet"

    def test_clean_keystore_files_empties_a_cold_keypair_and_spares_a_regular_one(self, backend):
        cold_key_uid = add_seed_keypair(backend)["key-uid"]
        app_key_uid = add_seed_keypair(backend, user_mnemonic_12, name="app-keypair")["key-uid"]

        backend.accounts_service.migrate_non_profile_keypair_to_cold_wallet(cold_key_uid, backend.password, "status-keycard")
        backend.accounts_service.make_seed_phrase_keypair_fully_operable(user_1.passphrase, backend.password)
        assert any(self._keystore_present(backend, cold_key_uid).values()), "Expected files to exist before cleaning"

        backend.accounts_service.clean_keystore_files(backend.password)

        assert not any(
            self._keystore_present(backend, cold_key_uid).values()
        ), "Expected every file of a cold keypair to go, the master address included"
        assert all(
            self._keystore_present(backend, app_key_uid).values()
        ), "Expected a keypair that is not cold and has no removed accounts to be untouched"
