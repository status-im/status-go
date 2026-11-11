from resources.constants import keypair_name, user_1, wallet_account_details_derivation


def add_seed_keypair(backend, user=user_1, name=keypair_name):
    response = backend.accounts_service.add_keypair_via_seed_phrase(user.passphrase, backend.password, name, "", wallet_account_details_derivation)
    assert response.get("key-uid"), "Expected addKeypairViaSeedPhrase to return the created keypair"
    assert response.get("cold-wallet", "") == "", "Expected a seed-imported keypair to start off any cold wallet"
    return response


def keystore_present(backend, keypair):
    addresses = [a["address"] for a in keypair["accounts"]] + [keypair["derived-from"]]
    return {address: backend.accounts_service.verify_keystore_file_for_account(address, backend.password) for address in addresses}
