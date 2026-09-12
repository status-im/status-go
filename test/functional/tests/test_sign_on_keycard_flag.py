"""signOnKeycard tells the client to route signing to the cold wallet instead of asking for a password.

It is derived from the keypair, not the account, and it is set on two different signing surfaces with
two different wire shapes: buildTransaction omits it when false, the router's signingDetails always
sends it. These tests pin the true branch of buildTransaction, which nothing else drives.
"""

import json

import pytest
from clients.api import ApiResponseError
from resources.constants import (
    ANVIL_NETWORK_ID,
    keypair_name,
    user_1,
    wallet_account_details_derivation,
)

COLD_WALLET_TYPES = ("status-keycard", "ledger", "trezor")
RECIPIENT = "0x70997970c51812dc3a010c7d01b50e0d17dc79c8"
# Fully specified so validateAndBuildTransaction never needs a nonce or gas estimate from the chain.
TX_TEMPLATE = {
    "to": RECIPIENT,
    "value": "0x1",
    "nonce": "0x0",
    "gas": "0x5208",
    "gasPrice": "0x3b9aca00",
}


@pytest.mark.rpc
class TestSignOnKeycardFlag:

    @pytest.fixture()
    def backend(self, backend_new_profile):
        return backend_new_profile("sign-on-keycard")

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
        assert response.get("cold-wallet", "") == "", "Expected a seed-imported keypair to start off any cold wallet"
        address = response["accounts"][0]["address"]
        return key_uid, address

    def _profile_wallet_address(self, backend):
        kp = backend.accounts_service.get_keypair_by_key_uid(backend.key_uid)
        wallet = next((a for a in kp["accounts"] if a.get("wallet")), None)
        assert wallet is not None, "Expected the profile keypair to own a wallet account"
        return wallet["address"]

    def _build(self, backend, address):
        args = dict(TX_TEMPLATE, **{"from": address})
        response = backend.wallet_service.build_transaction(ANVIL_NETWORK_ID, json.dumps(args))
        assert response.get("messageToSign"), f"Expected buildTransaction to produce a hash to sign for {address}"
        return response

    def test_sign_on_keycard_follows_the_cold_wallet_round_trip(self, backend):
        key_uid, address = self._add_seed_keypair(backend)

        before = self._build(backend, address)
        # TxResponse.SignOnKeycard is `omitempty`, so a regular keypair sends no such key at all.
        assert "signOnKeycard" not in before, "Expected buildTransaction to omit signOnKeycard for a keypair in the app keystore"

        backend.accounts_service.migrate_non_profile_keypair_to_cold_wallet(key_uid, backend.password, "status-keycard")

        cold = self._build(backend, address)
        assert cold.get("signOnKeycard") is True, "Expected signOnKeycard once the keypair's key lives on the card"
        for field in ("keyUid", "address", "addressPath", "chainId", "messageToSign"):
            assert cold.get(field) == before.get(field), f"Expected {field} to be unchanged by the migration"

        backend.accounts_service.migrate_non_profile_cold_wallet_keypair_to_app(user_1.passphrase, backend.password)

        after = self._build(backend, address)
        assert "signOnKeycard" not in after, "Expected signOnKeycard to drop again once the keystore files are restored"
        assert after.get("messageToSign") == before.get("messageToSign")

    @pytest.mark.parametrize("cold_wallet_type", COLD_WALLET_TYPES)
    def test_sign_on_keycard_is_set_for_every_cold_wallet_type(self, backend, cold_wallet_type):
        # The field is named for the keycard but MigratedToColdWallet() is true for ledger and trezor too.
        key_uid, address = self._add_seed_keypair(backend)
        backend.accounts_service.migrate_non_profile_keypair_to_cold_wallet(key_uid, backend.password, cold_wallet_type)

        assert backend.accounts_service.get_keypair_by_key_uid(key_uid)["cold-wallet"] == cold_wallet_type
        assert self._build(backend, address).get("signOnKeycard") is True

    def test_sign_on_keycard_is_per_keypair(self, backend):
        key_uid, cold_address = self._add_seed_keypair(backend)
        profile_address = self._profile_wallet_address(backend)

        backend.accounts_service.migrate_non_profile_keypair_to_cold_wallet(key_uid, backend.password, "status-keycard")

        assert self._build(backend, cold_address).get("signOnKeycard") is True
        assert "signOnKeycard" not in self._build(
            backend, profile_address
        ), "Expected the profile account to stay password-signed while another keypair is on a card"

    def test_build_transaction_rejects_an_unknown_address(self, backend):
        # Pins that the flag is read off a resolved account, not defaulted for an address the DB never saw.
        with pytest.raises(ApiResponseError, match="failed to resolve account"):
            self._build(backend, "0x" + "ab" * 20)
