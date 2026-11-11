from eth_account import Account

PATH_EIP1581_CHAT = "m/43'/60'/1581'/0'/0"
PATH_EIP1581_ENCRYPTION = "m/43'/60'/1581'/1'/0"

KEYCARD_UID = "mock-keycard-uid"
KEYCARD_PAIRING_SECURE_CHANNEL_V2 = "secure-channel-v2"
KEYPAIR_FIELDS = ("key-uid", "type", "name", "derived-from", "xpub")

Account.enable_unaudited_hdwallet_features()


def derive_keycard_password(backend, mnemonic: str) -> str:
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


def mock_pairing(key_uid, suffix=""):
    return f"mock-pairing-{key_uid[:8]}{suffix}"


def keypair_addresses(keypair):
    return [a["address"] for a in keypair["accounts"]] + [keypair["derived-from"]]


def account_shape(keypair):
    return sorted((a["address"], a["path"], a["wallet"], a["chat"]) for a in keypair["accounts"])


def keycard_pairing_of(accounts, key_uid):
    entry = next((a for a in accounts if a.get("key-uid") == key_uid), None)
    assert entry is not None, f"Expected InitializeApplication to list the account {key_uid}"
    return entry.get("keycard-pairing", "")


def fresh_profile_keypair(backend):
    keypair = backend.accounts_service.get_keypair_by_key_uid(backend.key_uid)
    assert keypair["type"] == "profile"
    assert keypair.get("cold-wallet", "") == "", "Expected a fresh profile keypair to not be on a cold wallet"
    assert keypair.get("xpub", "").startswith("xpub"), "Expected a fresh profile keypair to carry its wallet xpub"
    assert keypair["derived-from"], "Expected the profile keypair to carry its master address"
    return keypair
