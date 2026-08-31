import pytest

from steps import messenger
from utils.config import Config


# vut mode, peer mode. The peer mode applies to both the peer and the third backend.
CLIENT_MODES = [(False, False), (False, True), (True, False), (True, True)]


def _mode(is_light: bool) -> str:
    return "light" if is_light else "full"


def _peer_label(image: str) -> str:
    return image.split(":")[-1].removeprefix("statusgo-peer-") or "peer"


def pytest_generate_tests(metafunc):
    """Expand each test into one case per peer backend, per waku mode pair.

    Peer backends are the released images the test runner resolved, plus the build
    under test itself ("self"). The image and the two modes are parametrized in a
    single call so that one id can name both sides of the pairing; separate
    parametrize calls would leave pytest to join their ids with a "-".
    """
    if "peer_image" not in metafunc.fixturenames:
        return

    peer_images = list(Config.peer_docker_images or [])
    if Config.docker_image and Config.docker_image not in peer_images:
        peer_images.append(Config.docker_image)

    cases = []
    for image in peer_images:
        peer = "self" if image == Config.docker_image else _peer_label(image)
        for vut_light, peer_light in CLIENT_MODES:
            case_id = f"self-{_mode(vut_light)}__{peer}-{_mode(peer_light)}"
            cases.append(pytest.param(image, vut_light, peer_light, id=case_id))

    metafunc.parametrize(("peer_image", "vut_light", "peer_light"), cases)


@pytest.mark.compatibility
class TestChatCompatibility:
    """Cross-version chat smoke: vut (build under test) talks to peer (a released backend).

    The peer set always includes the build under test itself (the "self" id), so the
    mode matrix below is exercised same-version as well as cross-version. Nothing else
    in the suite pairs a full node with a light one: the rpc tests parametrize a single
    waku_light_client value across both sides.

    Each test runs in a full/light waku mode matrix: vut runs in the first mode,
    peer and third run in the second.
    """

    @pytest.fixture()
    def vut(self, backend_new_profile, vut_light):
        return backend_new_profile("vut", waku_light_client=vut_light)

    @pytest.fixture()
    def peer(self, backend_new_profile, peer_image, peer_light):
        return backend_new_profile("peer", image=peer_image, waku_light_client=peer_light)

    @pytest.fixture()
    def third(self, backend_new_profile, peer_image, peer_light):
        return backend_new_profile("third", image=peer_image, waku_light_client=peer_light)

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

        # Pre-install the community's persistent filter on the joiner before joining.
        # On light clients this avoids status-im/status-go#7547: otherwise the join's
        # internal warm-up fetchCommunity is the one that creates (and later forgets)
        # the bare community filter, collaterally tearing down the aggregated filter
        # subscription so pushed community messages are rejected. Spectating first
        # makes the filter already exist, so the warm-up fetches reuse it.
        messenger.spectate_and_fetch_community(peer, community_id)
        chat_id = messenger.join_community(member=peer, admin=vut, community_id=community_id)
        messenger.community_messages(chat_id, 2, sender=vut, receiver=peer)

        messenger.spectate_and_fetch_community(third, community_id)
        messenger.join_community(member=third, admin=vut, community_id=community_id)
        messenger.community_messages(chat_id, 2, sender=vut, receiver=third)

        messenger.ban_community_member(vut, community_id, third)
        messenger.community_messages(chat_id, 2, sender=peer, receiver=vut)

        messenger.leave_the_community(peer, community_id)
