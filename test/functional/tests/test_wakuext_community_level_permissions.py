"""Community-level permissions scenario — github.com/status-im/status-go/issues/7133."""

import time

import pytest

from clients.api import ApiResponseError
from clients.services.wakuext import (
    CommunityPermissionsAccess,
    CommunityRoles,
    CommunityTokenPermissionType,
    CommunityTokenType,
)
from steps import community_token_deploy, community_tokens, messenger
from steps.community_token_deploy import CommunityTokenDeployState

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


def _become_member_permission_id(community: dict) -> str:
    for permission_id, permission in (community.get("tokenPermissions") or {}).items():
        if permission.get("type") == CommunityTokenPermissionType.BECOME_MEMBER.value:
            return permission_id
    raise AssertionError("BECOME_MEMBER permission not found")


def _wait_until_live_community_has_permission(
    backend, community_id: str, permission_type: CommunityTokenPermissionType, attempts: int = 15, delay: int = 2
):
    """Poll a store fetch. A database fetch keeps the copy from before this permission existed."""
    community = None
    for _ in range(attempts):
        try:
            community = messenger.fetch_community(backend, community_id, try_database=False)
        except Exception:
            community = None
        if community and permission_type.value in community_tokens.community_permission_types(community):
            return community
        time.sleep(delay)
    observed = community_tokens.community_permission_types(community or {})
    raise AssertionError(f"{backend} did not see permission {permission_type.name}; observed={observed}")


@pytest.mark.rpc
@pytest.mark.usefixtures("community_token_deploy_context")
class TestCommunityLevelPermissions:
    snt_address: str
    snt_controller_address: str
    community_token_deployer: str
    deploy_state: CommunityTokenDeployState

    def test_membership_gated_by_snt_and_admin_token(
        self,
        owner_backend,
        member_backend,
        backend_new_profile,
        snt_token_overrides,
        multicall3_deployer,
        foundry_client,
        anvil_client,
    ):
        member_a = member_backend
        member_b = community_tokens.create_member_b_profile(
            backend_new_profile, snt_token_overrides, multicall3_deployer, self.community_token_deployer
        )

        community_id = community_tokens.create_token_gated_community(
            owner_backend,
            self.snt_address,
            permission_types=[CommunityTokenPermissionType.BECOME_MEMBER],
            token_criteria=_snt_criteria(self.snt_address, 1000),
            membership=CommunityPermissionsAccess.MANUAL_ACCEPT,
        )
        community_tokens.wait_until_member_sees_permissions(member_a, community_id, CommunityTokenPermissionType.BECOME_MEMBER)
        community_tokens.wait_until_member_sees_permissions(member_b, community_id, CommunityTokenPermissionType.BECOME_MEMBER)

        member_a_wallet = messenger.wallet_address(member_a)
        permissions = member_a.wakuext_service.check_permissions_to_join_community(community_id)
        assert not permissions.get("satisfied")
        with pytest.raises(ApiResponseError, match="permission to join not satisfied"):
            community_tokens.request_to_join_with_signatures(member_a, community_id, [member_a_wallet])
        assert member_a.public_key not in messenger.community_members(owner_backend, community_id)

        member_b_wallet = community_tokens.fund_backend_account_with_tokens(
            member_b, foundry_client, self.snt_controller_address, self.snt_address, amount=1000
        )
        community_tokens.verify_token_balance(foundry_client, CommunityTokenType.ERC20, self.snt_address, member_b_wallet)
        community_tokens.join_token_gated_community_as_member(owner_backend, member_b, community_id, member_b_wallet)
        members = messenger.community_members(owner_backend, community_id)
        assert member_b.public_key in members
        assert CommunityRoles.ROLE_ADMIN.value not in members[member_b.public_key].get("roles", [])

        community_tokens.fund_native_balance(owner_backend, anvil_client)
        owner_backend.wallet_service.restart_wallet_reload_timer()
        tokens = community_token_deploy.deploy_owner_and_master_tokens(
            owner_backend,
            community_id,
            self.community_token_deployer,
            self.deploy_state,
            anvil_client=anvil_client,
            wait_for_deploy_status_signal=False,
        )
        admin_token_address = community_token_deploy.deploy_community_admin_permission_token(
            owner_backend,
            community_id,
            owner_token_address=tokens.owner_token_address,
            master_token_address=tokens.master_token_address,
            token_type=CommunityTokenType.ERC20,
            community_token_deployer=self.community_token_deployer,
            foundry_client=foundry_client,
            anvil_client=anvil_client,
        )
        owner_wallet = messenger.wallet_address(owner_backend)
        community_token_deploy.wait_until_community_admin_token_is_mint_usable(
            backend=owner_backend,
            community_id=community_id,
            token_contract_address=admin_token_address,
            sender_wallet_address=owner_wallet,
            recipient_wallet_addresses=[member_a_wallet],
            token_type=CommunityTokenType.ERC20,
        )
        community_token_deploy.airdrop_community_admin_token(
            owner_backend,
            community_id,
            admin_token_address,
            wallet_addresses=[member_a_wallet],
            token_type=CommunityTokenType.ERC20,
        )
        community_token_deploy.verify_member_holds_admin_token(foundry_client, CommunityTokenType.ERC20, admin_token_address, member_a_wallet)
        owner_backend.wakuext_service.create_community_token_permission(
            community_id=community_id,
            permission_type=CommunityTokenPermissionType.BECOME_ADMIN,
            token_criteria=community_token_deploy.admin_permission_token_criteria(
                CommunityTokenType.ERC20, owner_backend.network_id, admin_token_address
            ),
        )

        # The admin token arrives with community metadata asynchronously.
        _wait_until_live_community_has_permission(member_a, community_id, CommunityTokenPermissionType.BECOME_ADMIN)
        admin_token_key = f"{member_a.network_id}-{admin_token_address.lower()}"
        balance_error = None
        for _ in range(15):
            try:
                member_a.wallet_service.get_balances_at_by_chain([member_a_wallet], [admin_token_key])
                balance_error = None
                break
            except ApiResponseError as exc:
                balance_error = exc
                time.sleep(2)
        assert balance_error is None, f"Admin token balance was not readable: {balance_error}"
        member_a.wallet_service.restart_wallet_reload_timer()
        community_tokens.join_token_gated_community_as_member(owner_backend, member_a, community_id, member_a_wallet)
        community_tokens.sync_community_member_permissions(owner_backend, community_id)
        community_tokens.wait_for_member_role(
            owner_backend,
            community_id,
            member_a.public_key,
            CommunityRoles.ROLE_ADMIN.value,
        )

        owner_community = next(
            (c for c in messenger.communities_list(owner_backend.wakuext_service.communities()) if c.get("id") == community_id),
            None,
        )
        assert owner_community, "Community not found on owner"
        owner_backend.wakuext_service.edit_community_token_permission(
            permission_id=_become_member_permission_id(owner_community),
            community_id=community_id,
            permission_type=CommunityTokenPermissionType.BECOME_MEMBER,
            token_criteria=_snt_criteria(self.snt_address, 10000),
        )
        community_tokens.sync_community_member_permissions(owner_backend, community_id)

        members = messenger.community_members(owner_backend, community_id)
        assert member_b.public_key not in members
        assert member_a.public_key in members
