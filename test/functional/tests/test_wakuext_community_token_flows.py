from concurrent.futures import ThreadPoolExecutor

import pytest

from clients.services.wakuext import (
    CommunityPermissionsAccess,
    CommunityTokenType,
)
from steps import community_token_deploy, community_tokens, messenger
from steps.community_token_deploy import CommunityTokenDeployState
from utils import fake

COLLECTIBLE_SUPPLY = 10
ASSET_SUPPLY = 10 * 10**18
ASSET_AIRDROP = 10**18


def _start_owner_and_member(backend_new_profile, snt_token_overrides, multicall3_deployer, community_token_deployer_contract_address):
    profile_kwargs = {
        "token_overrides": snt_token_overrides,
        "multicall_contract_address": multicall3_deployer.contract_address,
        "community_token_deployer_contract_address": community_token_deployer_contract_address,
    }

    def _start(name):
        return backend_new_profile(name, **profile_kwargs)

    with ThreadPoolExecutor(max_workers=2) as pool:
        owner_future = pool.submit(_start, "owner")
        member_future = pool.submit(_start, "member")
        return owner_future.result(), member_future.result()


def _create_community_with_member(owner_backend, member_backend):
    community_resp = owner_backend.wakuext_service.create_community(
        name=fake.community_name(),
        description=fake.community_description(),
        membership=CommunityPermissionsAccess.MANUAL_ACCEPT,
    )
    community_id = community_resp.get("communities", [{}])[0].get("id")
    assert community_id, "Community was not created"

    assert messenger.spectate_and_fetch_community(member_backend, community_id), "Community not found for member"
    member_wallet = messenger.wallet_address(member_backend)
    community_tokens.join_community_with_signatures_and_accept(owner_backend, member_backend, community_id, member_wallet)
    return community_id, member_wallet


def _deploy_owner_master(owner_backend, community_id, deployer, deploy_state, anvil_client):
    community_tokens.fund_native_balance(owner_backend, anvil_client)
    owner_backend.wallet_service.restart_wallet_reload_timer()
    tokens = community_token_deploy.deploy_owner_and_master_tokens(
        owner_backend,
        community_id,
        deployer,
        deploy_state,
        anvil_client=anvil_client,
        wait_for_deploy_status_signal=False,
        wait_for_backend_transition=False,
    )
    assert tokens.owner_token_address, "Owner token was not deployed"
    assert tokens.master_token_address, "Master token was not deployed"
    return tokens


def _airdrop_when_mintable(owner_backend, community_id, token_address, member_wallet, token_type):
    owner_wallet = messenger.wallet_address(owner_backend)
    community_token_deploy.wait_until_community_admin_token_is_mint_usable(
        backend=owner_backend,
        community_id=community_id,
        token_contract_address=token_address,
        sender_wallet_address=owner_wallet,
        recipient_wallet_addresses=[member_wallet],
        token_type=token_type,
        attempts=40,
        delay=1,
    )
    community_token_deploy.airdrop_community_admin_token(
        owner_backend,
        community_id,
        token_address,
        wallet_addresses=[member_wallet],
        token_type=token_type,
    )


@pytest.mark.rpc
@pytest.mark.usefixtures("community_token_deploy_context")
class TestCommunityTokenFlows:
    snt_address: str
    snt_controller_address: str
    community_token_deployer: str
    deploy_state: CommunityTokenDeployState

    def test_collectible_and_asset_mint_airdrop_and_burn(
        self,
        backend_new_profile,
        snt_token_overrides,
        multicall3_deployer,
        community_token_deployer_contract_address,
        foundry_client,
        anvil_client,
    ):
        owner_backend, member_backend = _start_owner_and_member(
            backend_new_profile, snt_token_overrides, multicall3_deployer, community_token_deployer_contract_address
        )
        community_id, member_wallet = _create_community_with_member(owner_backend, member_backend)
        tokens = _deploy_owner_master(owner_backend, community_id, self.community_token_deployer, self.deploy_state, anvil_client)

        collectible_address = community_token_deploy.deploy_community_erc721_collectible(
            owner_backend,
            community_id,
            tokens.owner_token_address,
            tokens.master_token_address,
            self.community_token_deployer,
            foundry_client,
            anvil_client=anvil_client,
            name="Collectible",
            symbol="COL",
            supply=str(COLLECTIBLE_SUPPLY),
            remote_self_destruct=True,
            description="Community collectible",
        )
        _airdrop_when_mintable(owner_backend, community_id, collectible_address, member_wallet, CommunityTokenType.ERC721)
        community_token_deploy.verify_member_holds_admin_token(foundry_client, CommunityTokenType.ERC721, collectible_address, member_wallet)
        assert community_token_deploy.remaining_collectibles_supply(anvil_client, collectible_address) == COLLECTIBLE_SUPPLY - 1

        community_token_deploy.remote_burn_community_token(
            owner_backend,
            community_id,
            collectible_address,
            token_ids=[0],
            wallet_addresses=[member_wallet],
            anvil_client=anvil_client,
        )
        with pytest.raises(RuntimeError, match="cast call failed"):
            foundry_client.get_erc721_owner(collectible_address, 0)

        remaining = community_token_deploy.remaining_collectibles_supply(anvil_client, collectible_address)
        assert remaining > 0, "Expected unminted supply after remote burn"
        community_token_deploy.burn_community_token(
            owner_backend,
            community_id,
            collectible_address,
            CommunityTokenType.ERC721,
            remaining,
            anvil_client=anvil_client,
        )
        assert community_token_deploy.remaining_collectibles_supply(anvil_client, collectible_address) == 0

        asset_address = community_token_deploy.deploy_community_erc20_asset(
            owner_backend,
            community_id,
            tokens.owner_token_address,
            tokens.master_token_address,
            self.community_token_deployer,
            foundry_client,
            anvil_client=anvil_client,
            name="Asset",
            symbol="AST",
            supply=str(ASSET_SUPPLY),
            description="Community asset",
        )
        _airdrop_when_mintable(owner_backend, community_id, asset_address, member_wallet, CommunityTokenType.ERC20)
        community_tokens.verify_token_balance(foundry_client, CommunityTokenType.ERC20, asset_address, member_wallet, min_balance=ASSET_AIRDROP)

        remaining = community_token_deploy.remaining_assets_supply(anvil_client, asset_address)
        assert remaining == ASSET_SUPPLY - ASSET_AIRDROP
        community_token_deploy.burn_community_token(
            owner_backend,
            community_id,
            asset_address,
            CommunityTokenType.ERC20,
            remaining,
            anvil_client=anvil_client,
        )
        assert community_token_deploy.remaining_assets_supply(anvil_client, asset_address) == 0
        community_tokens.verify_token_balance(foundry_client, CommunityTokenType.ERC20, asset_address, member_wallet, min_balance=ASSET_AIRDROP)
