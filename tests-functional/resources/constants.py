from dataclasses import dataclass
import os
from typing import Optional


@dataclass
class Account:
    address: str
    private_key: str
    password: str
    passphrase: str
    accounts: Optional[list] = None  # Optional list of accounts
    profile_data: Optional[dict] = None  # Optional profile data


user_1 = Account(
    address="0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
    private_key="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
    password="Strong12345",
    passphrase="test test test test test test test test test test test junk",
)
user_2 = Account(
    address="0x70997970C51812dc3A010C7d01b50e0d17dc79C8",
    private_key="0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d",
    password="Strong12345",
    passphrase="test test test test test test test test test test nest junk",
)

user_mnemonic_12 = Account(
    address="0xC43f4Ab94eC965a3EE9815C5Df07383057d261A8",
    private_key="",
    password="Strong12345",
    passphrase="exhibit soldier miracle series edge atom daring alter absorb decide orphan addict",
    accounts=[
        {
            "address": "0xb01a7dbaaacc92581558a4d178289be7471ba0f4",
            "public-key": "0x047fd6e4384b764cb7782bf747ad3fb51e8c54c2b3c2be7029c72d85d16b07f4be6a8703cf0a1d97be4c8712ad"
            "52cd6a519efdc723faae4935bbaba5c320dde02b",
            "path": "m/43'/60'/1581'/0'/0",
            "prodPreferredChainIds": "1:10:42161:8453",
            "operable": "fully",
            "position": -1,
        },
        {
            "address": "0xc43f4ab94ec965a3ee9815c5df07383057d261a8",
            "public-key": "0x041d8f6ebfa662c506d3d11cd407373f8ff2eb4c9ddc61a834864150a908172efbaa37f0de7dbeeea51f02272"
            "74e6ccd78fa5a07de55716094dbba475ac9e9ab44",
            "path": "m/44'/60'/0'/0/0",
            "name": "Account 1",
            "colorId": "primary",
            "hidden": False,
            "prodPreferredChainIds": "1:10:42161:8453",
            "position": 0,
        },
    ],
    profile_data={
        "address": "0x5f8e02f9f52709b29c82bd893d9af0f83273a6e4",
        "currency": "usd",
        "networks/current-network": "",
        "dapps-address": "0xc43f4ab94ec965a3ee9815c5df07383057d261a8",
        "eip1581-address": "0x803102f704324d4a4f2ddb1c47f6ab31339d0f70",
        "key-uid": "0x3231d92c94548d14f097173765a50bebe28fbad8f2267c9e08cc4433a6f219a4",
        "latest-derived-path": 0,
        "link-preview-request-enabled": True,
        "messages-from-contacts-only": False,
        "mutual-contact-enabled?": False,
        "name": "Anchored Open Wirehair",
        "networks/networks": [],
        "news-feed-enabled?": True,
        "news-rss-enabled?": True,
        "photo-path": "",
        "preview-privacy?": False,
        "public-key": "0x047fd6e4384b764cb7782bf747ad3fb51e8c54c2b3c2be7029c72d85d16b07f4be6a8703cf0a1d97be4c8712ad52cd"
        "6a519efdc723faae4935bbaba5c320dde02b",
        "default-sync-period": 777600,
        "appearance": 0,
        "profile-pictures-show-to": 2,
        "profile-pictures-visibility": 2,
        "use-mailservers?": True,
        "wallet-root-address": "0x1846a7930d0ab03e5a120ebdd46eff3fe0365824",
        "send-status-updates?": True,
        "backup-enabled?": True,
        "show-community-asset-when-sending-tokens?": True,
        "display-assets-below-balance-threshold": 100000000,
        "url-unfurling-mode": 1,
        "compressedKey": "zQ3shoF8xQNaT44MWQxztUXK6DK9UU63PRgJhn1Zd7oYWXJ5K",
        "emojiHash": ["🔢", "🤞🏽", "🖍️", "🙇🏽‍♀️", "🙋🏼‍♀️", "🙌", "💎", "🎞️", "😊", "🦸🏼", "😤", "👵🏿", "🧑🏿‍🔧", "🤶🏿"],
    },
)

user_mnemonic_15 = Account(
    address="0x685d7ec8e08769ca7020a6b65709887e38e68e6d",
    private_key="",
    password="Strong12345",
    passphrase="category two chapter fame hunt horse huge rotate inner monkey affair champion mixed tail final",
    accounts=[
        {
            "address": "0x245b7438961de05444645898f9215e5cf0786891",
            "public-key": "0x04c898c7763afd577f10efdd9e5d607caafd6d708e6cad8cee1b6d822d6ab148eb4e76d1c8266c8b73c8ce1d76"
            "699e072ac5844a9a6934abb565044bf619336302",
            "path": "m/43'/60'/1581'/0'/0",
            "prodPreferredChainIds": "1:10:42161:8453",
            "operable": "fully",
            "position": -1,
        },
        {
            "address": "0x685d7ec8e08769ca7020a6b65709887e38e68e6d",
            "public-key": "0x0480acddfad1e73c3e70e8e50f82eb1566e3df125736e9fe9042c4df5022c825afe6234021ad8bbb43e0ab0196"
            "878c2d9e3c8b5a8f266aca72b0e23d1f84464c72",
            "path": "m/44'/60'/0'/0/0",
            "name": "Account 1",
            "colorId": "primary",
            "hidden": False,
            "prodPreferredChainIds": "1:10:42161:8453",
            "position": 0,
        },
    ],
    profile_data={
        "address": "0x3644a8cc3860606fdee3b95c8825e17933a91647",
        "dapps-address": "0x685d7ec8e08769ca7020a6b65709887e38e68e6d",
        "eip1581-address": "0xe98734898ff58ac33a1a9c28f732696ec3e6b580",
        "key-uid": "0x944c1ce03f83dd1750acee591745d6ef14da90723af86f97b2df7d7282e8dd97",
        "name": "Overcooked Lost Grayreefshark",
        "public-key": "0x04c898c7763afd577f10efdd9e5d607caafd6d708e6cad8cee1b6d822d6ab148eb4e76d1c8266c8b73c8ce1d76699e"
        "072ac5844a9a6934abb565044bf619336302",
        "wallet-root-address": "0xe89675c9be641ceeca9f250345dc58528c3de93b",
        "emojiHash": ["🏄🏾", "👒", "👨🏽‍🎓", "🅰️", "👩🏾‍🍳", "🪀", "📩", "🐄", "⏳", "👸🏻", "🚣🏾‍♂️", "👩🏽‍🤝‍👩🏻", "📀", "🌒"],
    },
)

user_mnemonic_24 = Account(
    address="0xf2d58ae5aa880f7c3f65d769296b1061c61e0955",
    private_key="",
    password="Strong12345",
    passphrase="border cabbage grape stage return enable bamboo main only voyage glad race patient stool drum sort "
    "army abandon elegant grit cinnamon endless rail drink",
    accounts=[
        {
            "address": "0x8a6d1f3b9f158f7274ca4be1a3f0056c86e2ccdb",
            "public-key": "0x04c25b359fab7fc8989d6325128b06dd9734b38d207dc2ab652e130a5d59852910fd6414694e1f5ce3a9cdd5c1"
            "f6bec9a425a57bae10c98ff4337adba3bf8c18bb",
            "path": "m/43'/60'/1581'/0'/0",
            "prodPreferredChainIds": "1:10:42161:8453",
            "operable": "fully",
            "position": -1,
        },
        {
            "address": "0xf2d58ae5aa880f7c3f65d769296b1061c61e0955",
            "public-key": "0x04218096ceb5420c9b4cfa9d0187a057099540edff0aa5882b0a16b76fc8f0056d1a01930db4981f8885d00137"
            "c535740c04eec7ebe8bae7ef9fd98338fba31e04",
            "path": "m/44'/60'/0'/0/0",
            "name": "Account 1",
            "colorId": "primary",
            "hidden": False,
            "prodPreferredChainIds": "1:10:42161:8453",
            "position": 0,
        },
    ],
    profile_data={
        "address": "0xb47386b0074a9ddfd979540f134915d1df8dc3d0",
        "dapps-address": "0xf2d58ae5aa880f7c3f65d769296b1061c61e0955",
        "eip1581-address": "0xf203f9c33afd10e2d3888289ad2cad81c4b017c4",
        "key-uid": "0xcf119f28496e4123dd6d5a4936c5f595ee1a873b11ead5f275098456eb8777c4",
        "name": "Selfassured Pesky Mayfly",
        "public-key": "0x04c25b359fab7fc8989d6325128b06dd9734b38d207dc2ab652e130a5d59852910fd6414694e1f5ce3a9cdd5c1f6be"
        "c9a425a57bae10c98ff4337adba3bf8c18bb",
        "wallet-root-address": "0x0410bd5715fdd8ccadede1d3131a9180a96e502c",
        "emojiHash": ["👦🏻", "🕔", "🧜", "👩🏽‍🤝‍👨🏼", "👩🏿‍🔧", "🉐", "🧝🏿‍♂️", "🚣🏾‍♀️", "🫀", "🏄🏿", "🌘", "🤵🏼‍♀️", "🏄🏿‍♀️", "🎴"],
    },
)

PROJECT_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../"))
TESTS_DIR = os.path.join(PROJECT_ROOT, "tests-functional")
SIGNALS_DIR = os.path.join(TESTS_DIR, "signals")
FORGE_OUTPUT_DIR = os.path.join(PROJECT_ROOT, "forge_output")
DEPLOYER_ACCOUNT = user_1
LOG_SIGNALS_TO_FILE = False  # used for debugging purposes
USE_IPV6 = os.getenv("USE_IPV6", "No")

gas_fee_mode_low = 0
gas_fee_mode_medium = 1
gas_fee_mode_high = 2
gas_fee_mode_custom = 3

processor_name_transfer = "Transfer"

ANVIL_NETWORK_ID = 31337
