import logging
import time

import pytest

from clients.services.wakuext import (
    CommunityPermissionsAccess,
    CommunityTokenPermissionType,
    CommunityTokenType,
    CommunityRoles,
)
from steps import community_tokens, messenger

logger = logging.getLogger(__name__)


@pytest.mark.rpc
@pytest.mark.usefixtures("community_token_snt_context")
class TestCommunityTokenMembership:
    snt_address: str
    snt_controller_address: str

    @pytest.mark.skip(reason="Pending on issue https://github.com/status-im/status-go/issues/7114")
    def test_membership_no_valid_tokens_fake_address(self, owner_backend, member_backend):
        """Test that join request with no tokens/fake address fails permission check (no request created)"""
        community_id = community_tokens.create_token_gated_community(
            owner_backend, self.snt_address, membership=CommunityPermissionsAccess.MANUAL_ACCEPT
        )

        time.sleep(2)
        messenger.fetch_community(member_backend, community_id)

        fake_address = "0x" + "0" * 40
        join_req = community_tokens.request_to_join_with_signatures(member_backend, community_id, [fake_address])
        requests = join_req.get("requestsToJoinCommunity", [])
        assert len(requests) == 0, "No request should get accepted"

        declined_reqs = owner_backend.wakuext_service.declined_requests_to_join_for_community(community_id)
        assert len(declined_reqs) == 0

        communities = member_backend.wakuext_service.communities()
        member_community = next((c for c in messenger.communities_list(communities) if c.get("id") == community_id), None)
        assert member_community is None or not member_community.get("joined", False)

    def test_membership_with_valid_tokens(self, owner_backend, member_with_snt_backend, foundry_client):
        """Test that users with required tokens can successfully join community as member"""
        member_address = community_tokens.fund_backend_account_with_tokens(
            member_with_snt_backend, foundry_client, self.snt_controller_address, self.snt_address
        )
        community_tokens.verify_token_balance(foundry_client, CommunityTokenType.ERC20, self.snt_address, member_address)

        community_id = community_tokens.create_token_gated_community(
            owner_backend,
            self.snt_address,
            permission_types=[CommunityTokenPermissionType.BECOME_MEMBER],
            membership=CommunityPermissionsAccess.MANUAL_ACCEPT,
        )

        time.sleep(2)

        member_community = messenger.spectate_and_fetch_community(member_with_snt_backend, community_id)
        assert member_community, "Community not found on member"
        assert CommunityTokenPermissionType.BECOME_MEMBER.value in community_tokens.community_permission_types(
            member_community
        ), f"Expected BECOME_MEMBER permission (type=2), got {community_tokens.community_permission_types(member_community)}"

        member_with_snt_backend.wallet_service.restart_wallet_reload_timer()
        time.sleep(2)
        community_tokens.join_token_gated_community_as_member(owner_backend, member_with_snt_backend, community_id, member_address)

        communities = owner_backend.wakuext_service.communities()
        owner_community = next((c for c in messenger.communities_list(communities) if c.get("id") == community_id), None)
        assert owner_community is not None

        member_public_key = member_with_snt_backend.public_key
        assert member_public_key in owner_community.get("members", {}), f"Member {member_public_key} not found in community members"
        member_roles = owner_community["members"][member_public_key].get("roles", [])
        assert CommunityRoles.ROLE_ADMIN.value not in member_roles, "Member should join without admin role"

    def test_admin_token_permissions_with_valid_tokens(self, owner_backend, member_with_snt_backend, foundry_client):
        """Test that users with required tokens get admin privileges"""
        member_address = community_tokens.fund_backend_account_with_tokens(
            member_with_snt_backend, foundry_client, self.snt_controller_address, self.snt_address
        )
        community_tokens.verify_token_balance(foundry_client, CommunityTokenType.ERC20, self.snt_address, member_address)

        community_id = community_tokens.create_token_gated_community(
            owner_backend,
            self.snt_address,
            permission_types=[
                CommunityTokenPermissionType.BECOME_MEMBER,
                CommunityTokenPermissionType.BECOME_ADMIN,
            ],
            membership=CommunityPermissionsAccess.MANUAL_ACCEPT,
        )

        time.sleep(2)

        response = messenger.spectate_and_fetch_community(member_with_snt_backend, community_id)
        assert response, "Community not found"
        assert response["tokenPermissions"], "No token permissions found"
        assert len(response["tokenPermissions"]) == 2, "Unexpected number of token permissions"
        perm_types = community_tokens.community_permission_types(response)
        assert CommunityTokenPermissionType.BECOME_MEMBER.value in perm_types
        assert CommunityTokenPermissionType.BECOME_ADMIN.value in perm_types

        community_tokens.join_token_gated_community_as_member(owner_backend, member_with_snt_backend, community_id, member_address, attempts=3)

        communities = owner_backend.wakuext_service.communities()
        owner_community = next(
            (c for c in messenger.communities_list(communities) if c.get("id") == community_id),
            None,
        )
        assert owner_community is not None

        member_key = member_with_snt_backend.public_key
        owner_key = owner_backend.public_key

        assert member_key in owner_community.get("members", {})
        assert CommunityRoles.ROLE_ADMIN.value in owner_community["members"][member_key].get("roles", [])

        assert owner_key in owner_community.get("members", {})
        assert CommunityRoles.ROLE_OWNER.value in owner_community["members"][owner_key].get("roles", [])
