import logging

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

        community_tokens.wait_until_member_sees_permissions(
            member_backend,
            community_id,
            CommunityTokenPermissionType.BECOME_MEMBER,
        )

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

        community_tokens.wait_until_member_sees_permissions(
            member_with_snt_backend,
            community_id,
            CommunityTokenPermissionType.BECOME_MEMBER,
        )

        member_with_snt_backend.wallet_service.restart_wallet_reload_timer()
        community_tokens.join_token_gated_community_as_member(owner_backend, member_with_snt_backend, community_id, member_address)

        community_tokens.sync_community_member_permissions(owner_backend, community_id)

        member_public_key = member_with_snt_backend.public_key
        communities = owner_backend.wakuext_service.communities()
        owner_community = next((c for c in messenger.communities_list(communities) if c.get("id") == community_id), None)
        assert owner_community is not None
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

        response = community_tokens.wait_until_member_sees_permissions(
            member_with_snt_backend,
            community_id,
            CommunityTokenPermissionType.BECOME_MEMBER,
            CommunityTokenPermissionType.BECOME_ADMIN,
        )
        assert response["tokenPermissions"], "No token permissions found"
        assert len(response["tokenPermissions"]) == 2, "Unexpected number of token permissions"
        perm_types = community_tokens.community_permission_types(response)
        assert CommunityTokenPermissionType.BECOME_MEMBER.value in perm_types
        assert CommunityTokenPermissionType.BECOME_ADMIN.value in perm_types

        member_with_snt_backend.wallet_service.restart_wallet_reload_timer()
        community_tokens.join_token_gated_community_as_member(owner_backend, member_with_snt_backend, community_id, member_address)

        community_tokens.sync_community_member_permissions(owner_backend, community_id)

        member_key = member_with_snt_backend.public_key
        owner_key = owner_backend.public_key

        owner_community = community_tokens.wait_for_member_role(
            owner_backend,
            community_id,
            member_key,
            CommunityRoles.ROLE_ADMIN.value,
            attempts=15,
            delay=2,
        )
        assert owner_community

        assert member_key in owner_community.get("members", {})

        assert owner_key in owner_community.get("members", {})
        assert CommunityRoles.ROLE_OWNER.value in owner_community["members"][owner_key].get("roles", [])
