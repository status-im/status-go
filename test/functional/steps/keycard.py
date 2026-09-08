from eth_account import Account

PATH_EIP1581_CHAT = "m/43'/60'/1581'/0'/0"
PATH_EIP1581_ENCRYPTION = "m/43'/60'/1581'/1'/0"

Account.enable_unaudited_hdwallet_features()


def derive_keycard_password(backend, mnemonic: str) -> str:
    # A keycard profile's password is the encryption public key, the same value the mnemonic login path substitutes.
    derived = backend.wallet_service.get_derived_addresses_for_mnemonic(mnemonic, [PATH_EIP1581_ENCRYPTION])
    return derived[0]["public-key"]


def derive_chat_private_key(mnemonic: str) -> str:
    return Account.from_mnemonic(mnemonic, account_path=PATH_EIP1581_CHAT).key.hex()


def derive_chat_address(mnemonic: str) -> str:
    return Account.from_mnemonic(mnemonic, account_path=PATH_EIP1581_CHAT).address.lower()


def assert_keystore_state(backend, addresses, password: str, present: bool):
    for address in addresses:
        resp = backend.accounts_service.verify_keystore_file_for_account(address, password)
        expectation = "a keystore file" if present else "no keystore file"
        assert resp is present, f"Expected {expectation} for {address} with password {password!r}"
