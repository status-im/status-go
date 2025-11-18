import logging

import pytest
from clients.contract_deployers.snt import SNTDeployer
from clients.services.wakuext import CommunityPermissionsAccess
from steps.messenger import MessengerSteps
from utils.retry_utils import retry_call
from utils import fake
from resources.constants import user_1

logger = logging.getLogger(__name__)


class CommunityTokenPermissionType:
    BECOME_MEMBER = 1
    BECOME_ADMIN = 2
    BECOME_TOKEN_MASTER = 3
    CAN_VIEW_CHANNEL = 4
    CAN_VIEW_AND_POST_CHANNEL = 5


class CommunityTokenType:
    ERC20 = 1
    ERC721 = 2
    ENS = 3


class CommunityTokenPrivilegesLevel:
    OWNER_LEVEL = 1
    MASTER_LEVEL = 2
    COMMUNITY_LEVEL = 3


@pytest.mark.rpc
class TestCommunityTokenPermissions(MessengerSteps):
    def _communities_list(self, communities_response):
        if isinstance(communities_response, dict):
            return communities_response.get("communities", []) or []
        return communities_response or []

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        """Initialize backends for token permission tests"""
        self.owner = backend_new_profile("owner")
        self.member = backend_new_profile("member")
        self.non_member = backend_new_profile("non_member")
        self.fake_address = "0x" + "0" * 40  # Fake address for testing

    def create_token_gated_community(self, owner_backend, permission_type=CommunityTokenPermissionType.BECOME_MEMBER):
        """Helper to create a community with token permissions"""
        # Create basic community
        community_resp = owner_backend.wakuext_service.create_community(
            name=fake.community_name(),
            description=fake.community_description(),
            membership=CommunityPermissionsAccess.MANUAL_ACCEPT,
        )
        community_id = community_resp.get("communities", [{}])[0].get("id")

        # Add token permission
        token_criteria = [
            {
                "type": CommunityTokenType.ERC20,
                "contract_addresses": {31337: "0x1234567890123456789012345678901234567890"},
                "symbol": "TEST",
                "amountInWei": "1",  # 1 wei required
                "decimals": 18,
            }
        ]

        permission_resp = owner_backend.wakuext_service.create_community_token_permission(
            community_id=community_id, permission_type=permission_type, token_criteria=token_criteria
        )

        return community_id, permission_resp

    @pytest.mark.skip(reason="Pending on issue resolution https://github.com/status-im/status-go/issues/7114")
    def test_token_gated_community_membership_no_valid_tokens(self):
        """Test that users must hold required tokens to join community"""
        # Owner creates token-gated community
        community_id, _ = self.create_token_gated_community(self.owner, CommunityTokenPermissionType.BECOME_MEMBER)

        # Fetch community as member
        self.fetch_community(self.member, community_id)

        # Member tries to join without tokens - should fail at request time
        join_req = self.member.wakuext_service.request_to_join_community(community_id, self.fake_address)
        requests = join_req.get("requestsToJoinCommunity", [])
        if requests:
            # If request was created, check that it gets declined
            req_id = requests[0].get("id")
            # Check that request is declined due to insufficient permissions
            declined_reqs = retry_call(self.owner.wakuext_service.declined_requests_to_join_for_community, community_id)
            assert len(declined_reqs) == 1
            assert declined_reqs[0].get("id") == req_id
        else:
            # Request was rejected at creation time
            pass

        # Verify member is not in community
        communities = self.member.wakuext_service.communities()
        member_community = next((c for c in self._communities_list(communities) if c.get("id") == community_id), None)
        assert member_community is None or not member_community.get("joined", False)

    def test_token_gated_community_membership_with_valid_tokens(self, backend_new_profile, foundry_client):
        """Test that users with required tokens can successfully join community as member"""
        # Deploy SNT token
        snt_deployer = SNTDeployer(foundry_client)
        snt_address = snt_deployer.snt_contract_address
        controller_address = snt_deployer.snt_token_controller_address

        # Create token overrides for wallet service
        token_overrides = [{"symbol": "SNT", "address": snt_address}]

        # Create backends with token overrides
        owner = backend_new_profile("owner", token_overrides=token_overrides)
        member = backend_new_profile("member", token_overrides=token_overrides)

        # Get member's wallet address
        accounts = member.accounts_service.get_accounts()
        member_address = accounts[0].get("address") if accounts else self.fake_address

        # Generate 10 SNT tokens for member address using the controller
        token_amount = str(10 * 10**18)  # 10 tokens with 18 decimals
        generate_cmd = (
            f"cast send {controller_address} 'generateTokens(address,uint256)' "
            f"{member_address} {token_amount} --rpc-url http://anvil:8545 "
            f"--private-key {user_1.private_key}"
        )
        result = foundry_client.container.exec_run(generate_cmd)
        logger.debug(f"Generate tokens result: exit_code={result.exit_code}, output={result.output.decode()}")

        # Verify the balance directly with cast
        balance_cmd = f"cast call {snt_address} 'balanceOf(address)' {member_address} --rpc-url http://anvil:8545"
        balance_result = foundry_client.container.exec_run(balance_cmd)
        logger.debug(f"SNT balance check: exit_code={balance_result.exit_code}, output={balance_result.output.decode()}")

        logger.debug(f"Funded {member_address} with 10 SNT tokens at contract {snt_address}")
        token_address = snt_address

        # Owner creates token-gated community with the deployed token
        community_resp = owner.wakuext_service.create_community(
            name=fake.community_name(),
            description=fake.community_description(),
            membership=CommunityPermissionsAccess.MANUAL_ACCEPT,
        )
        community_id = community_resp.get("communities", [{}])[0].get("id")

        # Add token permission requiring 1 SNT token
        token_criteria = [
            {
                "type": CommunityTokenType.ERC20,
                "contract_addresses": {31337: token_address},
                "symbol": "SNT",
                "amountInWei": "1000000000000000000",  # 1 token
                "decimals": 18,
            }
        ]

        owner.wakuext_service.create_community_token_permission(
            community_id=community_id, permission_type=CommunityTokenPermissionType.BECOME_MEMBER, token_criteria=token_criteria
        )

        # Fetch community as member
        self.fetch_community(member, community_id)

        # Member tries to join with their wallet address (should have tokens)
        join_req = member.wakuext_service.request_to_join_community(community_id, member_address)
        requests = join_req.get("requestsToJoinCommunity", [])

        if requests:
            req_id = requests[0].get("id")
            # Since member has valid tokens, request should not be declined
            # Wait a bit for token validation to complete, then try to accept directly
            import time

            time.sleep(2)

            # Owner can accept the request since member has tokens
            accept_resp = owner.wakuext_service.accept_request_to_join_community(req_id)
            assert accept_resp is not None, f"Failed to accept request: {accept_resp}"

        # Verify member is now in community (check from owner's perspective since acceptance happened there)
        communities = owner.wakuext_service.communities()
        owner_community = next((c for c in self._communities_list(communities) if c.get("id") == community_id), None)
        assert owner_community is not None
        # Check that member is in the community members list
        member_public_key = member.public_key
        assert member_public_key in owner_community.get("members", {}), f"Member {member_public_key} not found in community members"
