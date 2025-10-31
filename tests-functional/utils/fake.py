import random

from faker import Faker

# Use a single English locale to minimize provider loading
_faker = Faker("en")


def community_name() -> str:
    ALLOWED_SPECIAL_CHARS = (".", "_", "-", " ")
    return _faker.word() + _faker.random_element(ALLOWED_SPECIAL_CHARS) + str(_faker.random_number())


def community_channel_name() -> str:
    return _faker.word()


def emoji() -> str:
    return _faker.emoji()


def color() -> str:
    return _faker.hex_color()


def community_description() -> str:
    return _faker.sentence()


def profile_name() -> str:
    length = random.randint(5, 24)
    return _faker.pystr(min_chars=length, max_chars=length)


def emoji() -> str:
    return _faker.emoji()


def account_name() -> str:
    return _faker.word()


def profile_password(length: int = 8) -> str:
    # Letters + digits; no special characters to keep compatibility
    return _faker.password(length=length, special_chars=False)


def community_channel_identity() -> dict:
    return {
        "displayName": community_channel_name(),
        "emoji": emoji(),
        "color": color(),
        "description": community_description(),
    }
