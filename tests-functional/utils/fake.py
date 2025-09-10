from faker import Faker

# Use a single English locale to minimize provider loading
_faker = Faker("en")


def community_name() -> str:
    ALLOWED_SPECIAL_CHARS = (".", "_", "-", " ")
    return _faker.word() + _faker.random_element(ALLOWED_SPECIAL_CHARS) + str(_faker.random_number())


def community_description() -> str:
    return _faker.sentence()


def profile_name() -> str:
    return _faker.user_name()


def profile_password(length: int = 8) -> str:
    # Letters + digits; no special characters to keep compatibility
    return _faker.password(length=length, special_chars=False)
