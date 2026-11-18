"""Channel-level permissions scenario — github.com/status-im/status-go/issues/7134."""

import time

import pytest

from clients.api import ApiResponseError
from clients.signals import SignalType
from clients.services.wakuext import CommunityTokenPermissionType, CommunityTokenType
from steps import community_tokens, messenger

SNT_CHAIN_ID = 31337


def _snt_criteria(snt_address: str, amount_tokens: int) -> list[dict]:
    return [
        {
            "type": CommunityTokenType.ERC20.value,
            "contract_addresses": {SNT_CHAIN_ID: snt_address},
            "symbol": "SNT",
            "amountInWei": str(amount_tokens * 10**18),
            "decimals": 18,
        }
    ]


def _short_chat_id(community_id: str, chat_id: str) -> str:
    if chat_id.startswith(community_id):
        return chat_id[len(community_id) :]
    return chat_id


def _channel(backend, community_id: str, chat_id: str):
    """Return the channel from a store fetch, or from the local copy when the store has nothing newer.

    A store fetch only returns a description newer than the local clock. Once that
    description is stored, the next fetch is null rather than a community missing the chat.
    """
    short_chat_id = _short_chat_id(community_id, chat_id)
    for try_database in (False, True):
        try:
            community = messenger.fetch_community(backend, community_id, try_database=try_database)
        except ApiResponseError:
            community = None
        chat = ((community or {}).get("chats") or {}).get(short_chat_id)
        if chat:
            return chat
    return None


def _wait_until_channel(backend, community_id: str, chat_id: str, attempts: int = 15, delay: int = 2, **expected):
    chat = None
    for _ in range(attempts):
        chat = _channel(backend, community_id, chat_id)
        if chat and all(chat.get(key) is value for key, value in expected.items()):
            return
        time.sleep(delay)
    raise AssertionError(f"{backend} channel {chat_id} is {chat}, expected {expected}")


def _token_gated_channel():
    return {
        "identity": {
            "display_name": "token-gated",
            "emoji": "😀",
            "color": "#1f2c75",
            "description": "token-gated",
        },
        "viewersCanPostReactions": False,
        "hideIfPermissionsNotMet": False,
        "permissions": {"access": 1},
    }


@pytest.mark.rpc
@pytest.mark.usefixtures("community_token_snt_context")
class TestCommunityChannelPermissions:
    snt_address: str
    snt_controller_address: str

    def test_token_gated_channel_view_and_post(self, owner_backend, member_backend, member_with_snt_backend, foundry_client):
        member_a = member_backend
        member_b = member_with_snt_backend

        community_id = messenger.create_community(owner_backend)
        create_resp = owner_backend.wakuext_service.create_community_chat(community_id, _token_gated_channel())
        chat_id = create_resp.get("chats")[0].get("id")
        messenger.wait_for_community_chat_visible(member_a, community_id, chat_id)
        messenger.wait_for_community_chat_visible(member_b, community_id, chat_id)

        member_a_wallet = messenger.wallet_address(member_a)
        community_tokens.join_community_with_signatures_and_accept(owner_backend, member_a, community_id, member_a_wallet)
        member_b_wallet = community_tokens.fund_backend_account_with_tokens(
            member_b, foundry_client, self.snt_controller_address, self.snt_address, amount=1000
        )
        community_tokens.join_community_with_signatures_and_accept(owner_backend, member_b, community_id, member_b_wallet)

        owner_text = "owner message in token-gated"
        start_a = len(member_a.received_signals[SignalType.MESSAGES_NEW])
        start_b = len(member_b.received_signals[SignalType.MESSAGES_NEW])
        send_resp = owner_backend.wakuext_service.send_chat_message(chat_id, owner_text)
        owner_message_id = send_resp.get("messages", [])[0].get("id", "")
        with member_a.expect_signal(SignalType.MESSAGES_NEW, pattern=owner_message_id, timeout=60, start=start_a):
            pass
        with member_b.expect_signal(SignalType.MESSAGES_NEW, pattern=owner_message_id, timeout=60, start=start_b):
            pass

        owner_backend.wakuext_service.create_community_token_permission(
            community_id=community_id,
            permission_type=CommunityTokenPermissionType.CAN_VIEW_AND_POST_CHANNEL,
            token_criteria=_snt_criteria(self.snt_address, 1000),
            chat_ids=[chat_id],
        )
        community_tokens.sync_community_member_permissions(owner_backend, community_id)

        snapshot = owner_backend.wakuext_service.get_community_members(community_id) or {}
        channel_roles = (snapshot.get("channels") or {}).get(_short_chat_id(community_id, chat_id)) or {}
        assert member_a.public_key not in channel_roles
        assert member_b.public_key in channel_roles
        members = {member["pubKey"] for member in snapshot.get("members") or []}
        assert member_a.public_key in members
        assert member_b.public_key in members

        _wait_until_channel(member_b, community_id, chat_id, canPost=True)
        _wait_until_channel(member_a, community_id, chat_id, canView=False, canPost=False)
        member_text = "member b message in token-gated"
        start_owner = len(owner_backend.received_signals[SignalType.MESSAGES_NEW])
        send_resp = member_b.wakuext_service.send_chat_message(chat_id, member_text)
        member_message_id = send_resp.get("messages", [])[0].get("id", "")
        with owner_backend.expect_signal(SignalType.MESSAGES_NEW, pattern=member_message_id, timeout=60, start=start_owner):
            pass
