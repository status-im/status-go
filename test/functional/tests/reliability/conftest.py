import pytest

from steps import messenger


@pytest.fixture()
def community_admin(backend_new_profile):
    return backend_new_profile("community_admin", bridge_network=True)


@pytest.fixture()
def community_member(backend_new_profile):
    return backend_new_profile("community_member", bridge_network=True)


@pytest.fixture()
def community_chat(community_admin, community_member):
    community_id = messenger.create_community(community_admin)
    chat_id = messenger.join_community(member=community_member, admin=community_admin, community_id=community_id)
    return {
        "admin": community_admin,
        "member": community_member,
        "chat_id": chat_id,
        "community_id": community_id,
    }
