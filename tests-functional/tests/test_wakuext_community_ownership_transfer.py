"""Community ownership transfer scenario — github.com/status-im/status-go/issues/7131."""

import logging
from uuid import uuid4

import pytest

from clients.services.wakuext import CommunityPermissionsAccess
from steps import community_ownership, community_tokens, community_token_deploy, messenger
from steps.community_token_deploy import CommunityTokenDeployState
from utils import fake

logger = logging.getLogger(__name__)

SUBCHANNEL_NAME = "subchannel"


def _subchannel_payload():
    display_name = SUBCHANNEL_NAME + "_" + str(uuid4())
    return {
        "identity": {
            "displayName": display_name,
            "color": "#1f2c75",
            "description": SUBCHANNEL_NAME,
        },
        "viewersCanPostReactions": True,
        "hideIfPermissionsNotMet": False,
        "permissions": {"access": 1},
    }


def _community_chat_count(backend, community_id):
    community = next(
        (c for c in messenger.communities_list(backend.wakuext_service.communities()) if c.get("id") == community_id),
        None,
    )
    return len((community or {}).get("chats", {}) or {})


@pytest.mark.rpc
@pytest.mark.usefixtures("community_token_deploy_context")
class TestCommunityOwnershipTransfer:
    snt_address: str
    snt_controller_address: str
    community_token_deployer: str
    deploy_state: CommunityTokenDeployState

    def test_ownership_transfer_to_member(
        self,
        owner_backend,
        member_backend,
        backend_new_profile,
        snt_token_overrides,
        multicall3_deployer,
        anvil_client,
    ):
        """Owner transfers the owner token to Member A; Member A becomes the new owner."""
        member_a_backend = member_backend
        member_b_backend = community_tokens.create_member_b_profile(
            backend_new_profile, snt_token_overrides, multicall3_deployer, self.community_token_deployer
        )

        # Given the Owner has created a community
        community_resp = owner_backend.wakuext_service.create_community(
            name=fake.community_name(),
            description=fake.community_description(),
            membership=CommunityPermissionsAccess.MANUAL_ACCEPT,
        )
        community_id = community_resp.get("communities", [{}])[0].get("id")
        assert community_id, "Community was not created"

        # And the Owner has created a channel named "subchannel"
        create_chat_resp = owner_backend.wakuext_service.create_community_chat(community_id, _subchannel_payload())
        assert len(create_chat_resp.get("chats", [])) == 1, "subchannel was not created"
        assert _community_chat_count(owner_backend, community_id) == 2, "Owner should see 'general' and 'subchannel'"

        # And Member A / Member B have joined the community
        assert messenger.spectate_and_fetch_community(member_a_backend, community_id), "Community not found for Member A"
        assert messenger.spectate_and_fetch_community(member_b_backend, community_id), "Community not found for Member B"

        member_a_wallet = messenger.wallet_address(member_a_backend)
        member_b_wallet = messenger.wallet_address(member_b_backend)
        community_tokens.join_community_with_signatures_and_accept(owner_backend, member_a_backend, community_id, member_a_wallet)
        community_tokens.join_community_with_signatures_and_accept(owner_backend, member_b_backend, community_id, member_b_wallet)

        owner_community = next(
            (c for c in messenger.communities_list(owner_backend.wakuext_service.communities()) if c.get("id") == community_id),
            None,
        )
        assert owner_community is not None
        assert member_a_backend.public_key in owner_community.get("members", {}), "Member A did not join"
        assert member_b_backend.public_key in owner_community.get("members", {}), "Member B did not join"

        # When the Owner mints the owner token
        community_tokens.fund_native_balance(owner_backend, anvil_client)
        community_tokens.fund_native_balance(member_a_backend, anvil_client)
        owner_backend.wallet_service.restart_wallet_reload_timer()
        tokens = community_token_deploy.deploy_owner_and_master_tokens(
            owner_backend,
            community_id,
            self.community_token_deployer,
            self.deploy_state,
            anvil_client=anvil_client,
            wait_for_deploy_status_signal=False,
        )
        assert tokens.owner_token_address, "Owner token was not deployed"

        community_ownership.publish_owner_token_to_community(owner_backend, community_id, tokens.owner_token_address)

        # And the Owner transfers the owner token to Member A
        # And Member A approves the ownership transfer
        community_ownership.transfer_community_ownership(
            owner_backend=owner_backend,
            new_owner_backend=member_a_backend,
            community_id=community_id,
            owner_token_address=tokens.owner_token_address,
            anvil_client=anvil_client,
        )

        # Then Member A is displayed as the Owner
        community_ownership.wait_until_member_is_owner(
            member_a_backend,
            community_id,
            member_a_backend.public_key,
            attempts=45,
        )

        # On ownership change the new owner kicks all members. Member B (a regular member) auto-rejoins
        # via its revealed accounts; wait for Member A to be owner with Member B re-added.
        community_ownership.wait_until_owner_and_members_present(
            backend=member_a_backend,
            community_id=community_id,
            owner_public_key=member_a_backend.public_key,
            expected_member_public_keys=[member_a_backend.public_key, member_b_backend.public_key],
            attempts=45,
            delay=2,
        )

        # The ex-owner created the community and has no revealed accounts, so it can't auto-rejoin; it
        # must manually re-join once it has observed the kick.
        community_ownership.wait_until_kicked_from_community(owner_backend, community_id)
        community_tokens.join_community_with_signatures_and_accept(
            member_a_backend, owner_backend, community_id, messenger.wallet_address(owner_backend)
        )

        # Then the members list contains Owner, Member A, and Member B
        new_community = community_ownership.wait_until_owner_and_members_present(
            backend=member_a_backend,
            community_id=community_id,
            owner_public_key=member_a_backend.public_key,
            expected_member_public_keys=[
                owner_backend.public_key,
                member_a_backend.public_key,
                member_b_backend.public_key,
            ],
            attempts=45,
            delay=2,
        )
        assert new_community is not None, "New owner's community not found"

        # And all members see the channels "general" and "subchannel"
        for backend, label in ((owner_backend, "owner"), (member_a_backend, "memberA"), (member_b_backend, "memberB")):
            messenger.fetch_community(backend, community_id, try_database=False)
            assert _community_chat_count(backend, community_id) == 2, f"{label} should see 'general' and 'subchannel'"

        # When Member A edits the community
        # Then the Owner and Member B see the updated community
        new_name, new_description = community_tokens.edit_community_and_wait_until_observer_sees_update(
            member_a_backend, owner_backend, community_id, attempts=45
        )
        community_tokens.wait_until_member_sees_community_update(
            member_b_backend, community_id, new_name, new_description, attempts=45, spectate=True
        )
