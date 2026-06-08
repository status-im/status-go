import pytest

from steps import messenger
from utils.config import Config


def pytest_generate_tests(metafunc):
    if "peer_image" in metafunc.fixturenames:
        images = Config.peer_docker_images or ([Config.docker_image] if Config.docker_image else [])
        ids = [(image.split(":")[-1] or "self") for image in images]
        metafunc.parametrize("peer_image", images, ids=ids)


@pytest.mark.compatibility
class TestChatCompatibility:
    """Cross-version chat smoke: vut (build under test) talks to peer (a released backend)."""

    @pytest.fixture()
    def vut(self, backend_new_profile):
        return backend_new_profile("vut")

    @pytest.fixture()
    def peer(self, backend_new_profile, peer_image):
        return backend_new_profile("peer", image=peer_image)

    @pytest.fixture()
    def third(self, backend_new_profile, peer_image):
        return backend_new_profile("third", image=peer_image)

    def test_one_to_one_compatibility(self, vut, peer):
        messenger.make_contacts(vut, peer)
        messenger.one_to_one_message(1, sender=vut, receiver=peer)
        messenger.one_to_one_message(1, sender=peer, receiver=vut)

    def test_group_chat_compatibility(self, vut, peer, third):
        messenger.make_contacts(vut, peer)
        messenger.make_contacts(vut, third)

        group_id = messenger.join_private_group(admin=vut, member=peer)
        messenger.private_group_message(2, group_id, sender=vut, receiver=peer)

        messenger.add_group_member(vut, group_id, third, observers=[peer, third])
        messenger.private_group_message(2, group_id, sender=vut, receiver=third)

        messenger.remove_group_member(vut, group_id, third, observers=[peer])
        messenger.private_group_message(2, group_id, sender=peer, receiver=vut)

        messenger.leave_group(peer, group_id, observers=[vut])

    def test_community_compatibility(self, vut, peer, third):
        community_id = messenger.create_community(vut)

        chat_id = messenger.join_community(member=peer, admin=vut, community_id=community_id)
        messenger.community_messages(chat_id, 2, sender=vut, receiver=peer)

        messenger.join_community(member=third, admin=vut, community_id=community_id)
        messenger.community_messages(chat_id, 2, sender=vut, receiver=third)

        messenger.ban_community_member(vut, community_id, third)
        messenger.community_messages(chat_id, 2, sender=peer, receiver=vut)

        messenger.leave_the_community(peer, community_id)
