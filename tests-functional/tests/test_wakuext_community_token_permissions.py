import logging
import pytest
from typing import Optional, List

from clients.contract_deployers.snt import SNTDeployer
from clients.services.wakuext import CommunityPermissionsAccess, CommunityTokenPermissionType, CommunityTokenType, CommunityRoles
from steps.messenger import MessengerSteps
from utils.retry_utils import retry_call
from utils import fake
from resources.constants import user_1

logger = logging.getLogger(__name__)


@pytest.mark.rpc
class TestCommunityTokenPermissions(MessengerSteps):
    def _communities_list(self, communities_response):
        if isinstance(communities_response, dict):
            return communities_response.get("communities", []) or []
        return communities_response or []

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile, foundry_client):
        """Initialize backends for token permission tests"""
        self.owner = backend_new_profile("owner")
        self.member = backend_new_profile("member")
        self.non_member = backend_new_profile("non_member")
        self.fake_address = "0x" + "0" * 40  # Fake address for testing

        # Deploy SNT token for tests that need it
        self.snt_deployer = SNTDeployer(foundry_client)
        self.snt_address = self.snt_deployer.snt_contract_address
        self.controller_address = self.snt_deployer.snt_token_controller_address

        # Create token overrides for wallet service
        token_overrides = [{"symbol": "SNT", "address": self.snt_address}]

        self.member_with_snt = backend_new_profile("member_with_snt", token_overrides=token_overrides)

        # Fund the member_with_snt with 10 SNT tokens
        accounts = self.member_with_snt.accounts_service.get_accounts()
        member_address = accounts[0]["address"] if accounts else self.fake_address
        token_amount = str(10 * 10**18)  # 10 tokens with 18 decimals
        gen_tokens_result = foundry_client.generate_tokens(self.controller_address, member_address, token_amount, user_1.private_key)
        logging.debug(f"Generate tokens result: exit_code={gen_tokens_result.exit_code}, output={gen_tokens_result.output.decode()}")
        logger.debug(f"Funded {member_address} with 10 SNT tokens at contract {self.snt_address}")

    def create_token_gated_community(
        self,
        owner_backend,
        permission_types: Optional[List[CommunityTokenPermissionType]] = None,
        token_criteria: Optional[List[dict]] = None,
        membership: CommunityPermissionsAccess = CommunityPermissionsAccess.AUTO_ACCEPT,
    ):
        """Helper to create a community and add token permissions"""
        if permission_types is None:
            permission_types = [CommunityTokenPermissionType.BECOME_MEMBER]
        if token_criteria is None:
            token_criteria = [
                {
                    "type": CommunityTokenType.ERC20.value,
                    "contract_addresses": {31337: self.snt_address},
                    "symbol": "SNT",
                    "amountInWei": "1",  # 1 wei required
                    "decimals": 18,
                }
            ]

        # Create basic community
        community_resp = owner_backend.wakuext_service.create_community(
            name=fake.community_name(),
            description=fake.community_description(),
            membership=membership,
        )
        community_id = community_resp.get("communities", [{}])[0].get("id")

        # Add token permissions
        for permission_type in permission_types:
            owner_backend.wakuext_service.create_community_token_permission(
                community_id=community_id, permission_type=permission_type, token_criteria=token_criteria
            )

        return community_id

    @pytest.mark.skip(reason="Pending on issue https://github.com/status-im/status-go/issues/7114")
    def test_membership_no_valid_tokens(self):
        """Test that users must hold required tokens to join community"""
        # Owner creates token-gated community
        community_id = self.create_token_gated_community(self.owner, membership=CommunityPermissionsAccess.MANUAL_ACCEPT)

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

    def test_membership_with_valid_tokens(self, foundry_client):
        """Test that users with required tokens can successfully join community as member"""
        # Get member's wallet address
        accounts = self.member_with_snt.accounts_service.get_accounts()
        member_address = accounts[0]["address"] if accounts else self.fake_address

        # Verify the balance directly with cast
        balance_result = foundry_client.get_erc20_balance(self.snt_address, member_address)
        assert balance_result.exit_code == 0, "Balance check command failed"
        balance = int(balance_result.output.decode().strip(), 16)
        assert balance >= 1000000000000000000, f"Insufficient SNT balance: {balance}, expected at least 1 token"

        # Owner creates token-gated community with the deployed token
        community_id = self.create_token_gated_community(
            self.owner,
            permission_types=[CommunityTokenPermissionType.BECOME_MEMBER],
            membership=CommunityPermissionsAccess.MANUAL_ACCEPT,
        )

        # Fetch community as member
        self.fetch_community(self.member_with_snt, community_id)

        # Member tries to join with their wallet address (should have tokens)
        join_req = self.member_with_snt.wakuext_service.request_to_join_community(community_id, member_address)
        requests = join_req.get("requestsToJoinCommunity", [])

        if requests:
            req_id = requests[0].get("id")
            # Since member has valid tokens, request should not be declined
            # Wait for token validation to complete, then try to accept directly

            def try_accept_request(req_id):
                resp = self.owner.wakuext_service.accept_request_to_join_community(req_id)
                return resp if resp is not None else None

            accept_resp = retry_call(try_accept_request, req_id)
            assert accept_resp is not None, f"Failed to accept request: {accept_resp}"

        # Verify member is now in community (check from owner's perspective since acceptance happened there)
        communities = self.owner.wakuext_service.communities()
        owner_community = next((c for c in self._communities_list(communities) if c.get("id") == community_id), None)
        assert owner_community is not None
        # Check that member is in the community members list
        member_public_key = self.member_with_snt.public_key
        assert member_public_key in owner_community.get("members", {}), f"Member {member_public_key} not found in community members"

    @pytest.mark.skip(reason="Pending on issue https://github.com/status-im/status-go/issues/7135")
    def test_admin_token_permissions_with_valid_tokens(self, foundry_client):
        """Test that users with required tokens get admin privileges"""
        # Get member's wallet address
        accounts = self.member_with_snt.accounts_service.get_accounts()
        member_address = accounts[0]["address"] if accounts else self.fake_address

        # Verify the balance directly with cast
        balance_result = foundry_client.get_erc20_balance(self.snt_address, member_address)
        assert balance_result.exit_code == 0, "Balance check command failed"
        balance = int(balance_result.output.decode().strip(), 16)
        assert balance >= 1000000000000000000, f"Insufficient SNT balance: {balance}, expected at 1 token"

        # Owner creates token-gated community with member and admin permissions
        community_id = self.create_token_gated_community(
            self.owner,
            permission_types=[
                CommunityTokenPermissionType.BECOME_MEMBER,
                CommunityTokenPermissionType.BECOME_ADMIN,
            ],
            membership=CommunityPermissionsAccess.MANUAL_ACCEPT,
        )

        # Fetch community as member
        self.fetch_community(self.member_with_snt, community_id)

        def check_permissions_satisfied(community_id):
            resp = self.member_with_snt.wakuext_service.check_permissions_to_join_community(community_id)
            return resp if resp.get("satisfied") else None

        permissions_resp = retry_call(check_permissions_satisfied, community_id)
        if not permissions_resp:
            pytest.fail("Permissions to join never became satisfied for member_with_snt")

        # Member with tokens requests to join community
        join_resp = self.member_with_snt.wakuext_service.request_to_join_community(community_id, member_address)
        requests = join_resp.get("requestsToJoinCommunity", [])

        if requests:
            req_id = requests[0].get("id")
            # Wait for token validation

            def try_accept_request(req_id):
                resp = self.owner.wakuext_service.accept_request_to_join_community(req_id)
                return resp if resp is not None else None

            accept_resp = retry_call(try_accept_request, req_id)
            assert accept_resp is not None, f"Failed to accept request: {accept_resp}"

        # Verify member is now in community and has admin role
        communities = self.owner.wakuext_service.communities()
        owner_community = next(
            (c for c in self._communities_list(communities) if c.get("id") == community_id),
            None,
        )
        assert owner_community is not None

        member_key = self.member_with_snt.public_key
        owner_key = self.owner.public_key

        # Member should have admin role, granted via BECOME_ADMIN token permission
        assert member_key in owner_community.get("members", {})
        assert CommunityRoles.ROLE_ADMIN.value in owner_community["members"][member_key].get("roles", [])

        # Owner should remain owner only, not changed by token reevaluation
        assert owner_key in owner_community.get("members", {})
        assert CommunityRoles.ROLE_OWNER.value in owner_community["members"][owner_key].get("roles", [])
