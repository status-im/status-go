from faker import Faker

# Use a single English locale to minimize provider loading
_faker = Faker("en")


def CommunityName() -> str:
    return _faker.company()


def CommunityDescription() -> str:
    return _faker.sentence()


def ProfileName() -> str:
    return _faker.user_name()


def ProfilePassword(length: int = 8) -> str:
    # Letters + digits; no special characters to keep compatibility
    return _faker.password(length=length, special_chars=False)
