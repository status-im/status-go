import logging

import pytest
from web3 import Web3

from clients.services.wakuext import (
    CommunityPermissionsAccess,
    CommunityTokenPrivilegesLevel,
    CommunityTokenType,
    CommunityRoles,
)
from clients.services.wallet_send_type import WalletSendType
from clients.signals import SignalType
from steps import community_tokens, community_token_deploy, messenger
from steps.community_token_deploy import CommunityTokenDeployState
from utils import fake

logger = logging.getLogger(__name__)


@pytest.mark.rpc
@pytest.mark.usefixtures("community_token_deploy_context")
class TestCommunityTokenMaster:
    snt_address: str
    snt_controller_address: str
    community_token_deployer: str
    deploy_state: CommunityTokenDeployState

    def test_master_token_holder_can_edit_and_mint_tokens(
        self,
        owner_backend,
        member_backend,
        backend_new_profile,
        snt_token_overrides,
        multicall3_deployer,
        anvil_client,
    ):
        """Test that a master token holder can edit community and mint/airdrop tokens"""
        member_b_backend = community_tokens.create_member_b_profile(
            backend_new_profile, snt_token_overrides, multicall3_deployer, self.community_token_deployer
        )

        community_resp = owner_backend.wakuext_service.create_community(
            name=fake.community_name(),
            description=fake.community_description(),
            membership=CommunityPermissionsAccess.MANUAL_ACCEPT,
        )
        community_id = community_resp.get("communities", [{}])[0].get("id")

        assert messenger.spectate_and_fetch_community(member_backend, community_id), "Community not found for member A"
        assert messenger.spectate_and_fetch_community(member_b_backend, community_id), "Community not found for member B"

        member_a_wallet = messenger.wallet_address(member_backend)
        member_b_wallet = messenger.wallet_address(member_b_backend)
        community_tokens.join_community_with_signatures_and_accept(owner_backend, member_backend, community_id, member_a_wallet)
        community_tokens.join_community_with_signatures_and_accept(owner_backend, member_b_backend, community_id, member_b_wallet)

        owner_community = next(
            (c for c in messenger.communities_list(owner_backend.wakuext_service.communities()) if c.get("id") == community_id),
            None,
        )
        assert owner_community is not None
        assert member_backend.public_key in owner_community.get("members", {})
        assert member_b_backend.public_key in owner_community.get("members", {})

        community_tokens.fund_native_balance(owner_backend, anvil_client)
        community_tokens.fund_native_balance(member_backend, anvil_client)

        owner_backend.wallet_service.restart_wallet_reload_timer()
        tokens = community_token_deploy.deploy_owner_and_master_tokens(
            owner_backend,
            community_id,
            self.community_token_deployer,
            self.deploy_state,
            anvil_client=anvil_client,
            wait_for_deploy_status_signal=False,
        )

        community_token_deploy.mint_master_to_member(
            owner_backend,
            community_id,
            tokens.master_token_address,
            member_a_wallet,
            tokens.owner_token_address,
            self.deploy_state,
            anvil_client=anvil_client,
        )

        owner_backend.wakuext_service.reevaluate_community_members_permissions(community_id)

        community_tokens.wait_for_member_role(
            owner_backend,
            community_id,
            member_backend.public_key,
            CommunityRoles.ROLE_TOKEN_MASTER.value,
            attempts=10,
            delay=2,
        )

        community_tokens.wait_for_member_role(
            member_backend,
            community_id,
            member_backend.public_key,
            CommunityRoles.ROLE_TOKEN_MASTER.value,
            attempts=30,
            delay=2,
            fetch_from_store=True,
            required=False,
        )

        new_name, new_description = community_tokens.edit_community_and_wait_until_observer_sees_update(
            member_backend, owner_backend, community_id, attempts=45
        )
        community_tokens.wait_until_member_sees_community_update(
            member_b_backend, community_id, new_name, new_description, attempts=45, spectate=True
        )

        with owner_backend.expect_signal(
            SignalType.COMMUNITY_TOKEN_ACTION,
            predicate=lambda s: s.get("event", {}).get("actionType") == 1,
            timeout=60,
        ):
            community_token_deploy.mint_community_token(
                sender_backend=member_backend,
                community_id=community_id,
                token_contract_address=tokens.master_token_address,
                wallet_addresses=[messenger.wallet_address(owner_backend)],
                token_type=CommunityTokenType.ERC721,
                privilege_level=CommunityTokenPrivilegesLevel.MASTER_LEVEL.value,
                amount=1,
            )

        owner_address = messenger.wallet_address(owner_backend)
        balance_of_abi = [
            {
                "inputs": [{"name": "owner", "type": "address"}],
                "name": "balanceOf",
                "outputs": [{"name": "", "type": "uint256"}],
                "stateMutability": "view",
                "type": "function",
            }
        ]
        master_token_contract = anvil_client.eth.contract(address=Web3.to_checksum_address(tokens.master_token_address), abi=balance_of_abi)
        owner_master_balance = master_token_contract.functions.balanceOf(Web3.to_checksum_address(owner_address)).call()
        assert owner_master_balance >= 1, f"Owner should hold at least 1 master token, got {owner_master_balance}"

        with member_backend.expect_signal(
            SignalType.COMMUNITY_TOKEN_TRANSACTION_STATUS_CHANGED,
            predicate=lambda s: community_token_deploy.is_community_token_tx_success(s, WalletSendType.COMMUNITY_MINT_TOKENS),
            timeout=60,
        ):
            community_token_deploy.mint_community_token(
                sender_backend=member_backend,
                community_id=community_id,
                token_contract_address=tokens.master_token_address,
                wallet_addresses=[member_b_wallet],
                token_type=CommunityTokenType.ERC721,
                privilege_level=CommunityTokenPrivilegesLevel.MASTER_LEVEL.value,
                amount=1,
            )

        member_b_master_balance = master_token_contract.functions.balanceOf(Web3.to_checksum_address(member_b_wallet)).call()
        assert member_b_master_balance >= 1, f"Member B should hold at least 1 master token, got {member_b_master_balance}"
