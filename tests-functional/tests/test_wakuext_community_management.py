import logging

import pytest

from clients.services.wakuext import (
    CommunityPermissionsAccess,
    CommunityTokenPermissionType,
    CommunityTokenType,
    CommunityRoles,
)
from steps import community_tokens, community_token_deploy, messenger
from steps.community_token_deploy import CommunityTokenDeployState
from steps.community_tokens import NATIVE_TOKEN_ADDRESS
from utils import fake

logger = logging.getLogger(__name__)


@pytest.mark.rpc
@pytest.mark.usefixtures("community_token_deploy_context")
class TestCommunityTokenGatedManagement:
    snt_address: str
    snt_controller_address: str
    community_token_deployer: str
    deploy_state: CommunityTokenDeployState

    def test_owner_edits_visible_before_and_after_minting_owner_token(self, owner_backend, member_backend, foundry_client, anvil_client):
        """Test that owner edits are visible before and after minting the owner token"""
        community_id = community_tokens.create_token_gated_community(
            owner_backend,
            self.snt_address,
            permission_types=[CommunityTokenPermissionType.BECOME_MEMBER],
            membership=CommunityPermissionsAccess.MANUAL_ACCEPT,
        )

        response = messenger.spectate_and_fetch_community(member_backend, community_id)
        assert response, "Community not found"

        member_address = community_tokens.fund_backend_account_with_tokens(
            member_backend, foundry_client, self.snt_controller_address, self.snt_address
        )
        community_tokens.verify_token_balance(foundry_client, CommunityTokenType.ERC20, self.snt_address, member_address)

        community_tokens.join_token_gated_community_as_member(owner_backend, member_backend, community_id, member_address)

        communities = owner_backend.wakuext_service.communities()
        owner_community = next((c for c in messenger.communities_list(communities) if c.get("id") == community_id), None)
        assert owner_community is not None
        assert member_backend.public_key in owner_community.get("members", {})

        community_tokens.edit_community_and_wait_until_observer_sees_update(owner_backend, member_backend, community_id)

        owner_accounts = owner_backend.accounts_service.get_accounts()
        assert owner_accounts, "No owner accounts found"
        owner_wallet_account = next(a for a in owner_accounts if not a.get("chat"))
        assert owner_wallet_account, "No owner wallet account found"
        owner_address = owner_wallet_account["address"]

        anvil_client.set_balance(owner_address, 10 * 10**18)
        owner_backend.wallet_service.fetch_or_get_cached_wallet_balances([owner_address], True)
        owner_backend.wallet_service.get_balances_at_by_chain([owner_address], [f"{owner_backend.network_id}-{NATIVE_TOKEN_ADDRESS}"])

        owner_backend.wallet_service.restart_wallet_reload_timer()
        community_token_deploy.deploy_owner_token(
            owner_backend, community_id, self.community_token_deployer, self.deploy_state, anvil_client=anvil_client
        )

        community_tokens.edit_community_and_wait_until_observer_sees_update(
            owner_backend, member_backend, community_id, attempts=45, spectate_before_wait=True
        )

        owner_backend.logout()
        owner_backend.login(owner_backend.key_uid, owner_backend.password)
        owner_backend.wait_for_login()
        owner_backend.wait_for_wakuext_ready(timeout=60, start_messenger=True)
        messenger.spectate_and_fetch_community(member_backend, community_id)

        community_tokens.edit_community_and_wait_until_observer_sees_update(
            owner_backend,
            member_backend,
            community_id,
            attempts=60,
            spectate_before_wait=True,
        )

    @pytest.mark.parametrize(
        "admin_permission_token_type",
        [CommunityTokenType.ERC20, CommunityTokenType.ERC721],
        ids=["erc20", "erc721"],
    )
    def test_community_admin_can_edit_community(
        self,
        admin_permission_token_type,
        owner_backend,
        member_backend,
        backend_new_profile,
        snt_token_overrides,
        multicall3_deployer,
        foundry_client,
        anvil_client,
    ):
        """Admin member can edit the community; updates are visible to all members."""
        member_a = member_backend
        member_b = community_tokens.create_member_b_profile(
            backend_new_profile, snt_token_overrides, multicall3_deployer, self.community_token_deployer
        )

        community_resp = owner_backend.wakuext_service.create_community(
            name=fake.community_name(),
            description=fake.community_description(),
            membership=CommunityPermissionsAccess.MANUAL_ACCEPT,
        )
        community_id = community_resp.get("communities", [{}])[0].get("id")

        assert messenger.spectate_and_fetch_community(member_a, community_id), "Community not found for member A"
        assert messenger.spectate_and_fetch_community(member_b, community_id), "Community not found for member B"

        member_a_wallet = messenger.wallet_address(member_a)
        member_b_wallet = messenger.wallet_address(member_b)
        community_tokens.join_community_with_signatures_and_accept(owner_backend, member_a, community_id, member_a_wallet)
        community_tokens.join_community_with_signatures_and_accept(owner_backend, member_b, community_id, member_b_wallet)

        owner_community = next(
            (c for c in messenger.communities_list(owner_backend.wakuext_service.communities()) if c.get("id") == community_id),
            None,
        )
        assert owner_community is not None
        assert member_a.public_key in owner_community.get("members", {})
        assert member_b.public_key in owner_community.get("members", {})

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

        community_token_deploy.setup_admin_via_token(
            owner_backend,
            member_a,
            community_id,
            member_a_wallet,
            admin_permission_token_type,
            self.community_token_deployer,
            foundry_client,
            anvil_client,
            tokens.owner_token_address,
            tokens.master_token_address,
        )

        owner_community = next(
            (c for c in messenger.communities_list(owner_backend.wakuext_service.communities()) if c.get("id") == community_id),
            None,
        )
        assert owner_community, "Community not found on owner"
        assert CommunityRoles.ROLE_ADMIN.value not in owner_community.get("members", {}).get(member_b.public_key, {}).get("roles", [])

        new_name, new_description = community_tokens.edit_community_and_wait_until_observer_sees_update(
            member_a, owner_backend, community_id, attempts=45
        )
        for backend in (member_a, member_b):
            community_tokens.wait_until_member_sees_community_update(backend, community_id, new_name, new_description, attempts=45, spectate=True)
