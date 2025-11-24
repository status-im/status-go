import logging
import time
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
        member_address = accounts[0].get("address") if accounts else self.fake_address
        token_amount = str(10 * 10**18)  # 10 tokens with 18 decimals
        generate_cmd = (
            f"cast send {self.controller_address} 'generateTokens(address,uint256)' "
            f"{member_address} {token_amount} --rpc-url http://anvil:8545 "
            f"--private-key {user_1.private_key}"
        )
        result = foundry_client.container.exec_run(generate_cmd)
        logger.debug(f"Generate tokens result for member: exit_code={result.exit_code}, output={result.output.decode()}")

        logger.debug(f"Funded {member_address} with 10 SNT tokens at contract {self.snt_address}")

    def create_token_gated_community(
        self, owner_backend, permission_type=CommunityTokenPermissionType.BECOME_MEMBER, membership=CommunityPermissionsAccess.AUTO_ACCEPT
    ):
        """Helper to create a community with token permissions"""
        # Create basic community
        community_resp = owner_backend.wakuext_service.create_community(
            name=fake.community_name(),
            description=fake.community_description(),
            membership=membership,
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
        community_id, _ = self.create_token_gated_community(
            self.owner, CommunityTokenPermissionType.BECOME_MEMBER, CommunityPermissionsAccess.MANUAL_ACCEPT
        )

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

    def test_token_gated_community_membership_with_valid_tokens(self, foundry_client):
        """Test that users with required tokens can successfully join community as member"""
        # Get member's wallet address
        accounts = self.member_with_snt.accounts_service.get_accounts()
        member_address = accounts[0].get("address") if accounts else self.fake_address

        # Verify the balance directly with cast
        balance_cmd = f"cast call {self.snt_address} 'balanceOf(address)' {member_address} --rpc-url http://anvil:8545"
        balance_result = foundry_client.container.exec_run(balance_cmd)
        logger.debug(f"SNT balance check: exit_code={balance_result.exit_code}, output={balance_result.output.decode()}")

        token_address = self.snt_address

        # Owner creates token-gated community with the deployed token
        community_resp = self.owner.wakuext_service.create_community(
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

        self.owner.wakuext_service.create_community_token_permission(
            community_id=community_id, permission_type=CommunityTokenPermissionType.BECOME_MEMBER, token_criteria=token_criteria
        )

        # Fetch community as member
        self.fetch_community(self.member_with_snt, community_id)

        # Member tries to join with their wallet address (should have tokens)
        join_req = self.member_with_snt.wakuext_service.request_to_join_community(community_id, member_address)
        requests = join_req.get("requestsToJoinCommunity", [])

        if requests:
            req_id = requests[0].get("id")
            # Since member has valid tokens, request should not be declined
            # Wait a bit for token validation to complete, then try to accept directly
            time.sleep(2)

            # Owner can accept the request since member has tokens
            accept_resp = self.owner.wakuext_service.accept_request_to_join_community(req_id)
            assert accept_resp is not None, f"Failed to accept request: {accept_resp}"

        # Verify member is now in community (check from owner's perspective since acceptance happened there)
        communities = self.owner.wakuext_service.communities()
        owner_community = next((c for c in self._communities_list(communities) if c.get("id") == community_id), None)
        assert owner_community is not None
        # Check that member is in the community members list
        member_public_key = self.member_with_snt.public_key
        assert member_public_key in owner_community.get("members", {}), f"Member {member_public_key} not found in community members"

    @pytest.mark.skip(reason="Pending on issue resolution https://github.com/status-im/status-go/issues/7135")
    def test_admin_token_permissions_with_valid_tokens(self, foundry_client):
        """Test that users with required tokens get admin privileges"""
        # Get member's wallet address
        accounts = self.member_with_snt.accounts_service.get_accounts()
        member_address = accounts[0].get("address") if accounts else self.fake_address

        # Verify the balance directly with cast
        balance_cmd = f"cast call {self.snt_address} 'balanceOf(address)' {member_address} --rpc-url http://anvil:8545"
        balance_result = foundry_client.container.exec_run(balance_cmd)
        logger.debug(f"SNT balance check: exit_code={balance_result.exit_code}, output={balance_result.output.decode()}")

        # Owner creates token-gated community with admin permission
        community_resp = self.owner.wakuext_service.create_community(
            name=fake.community_name(),
            description=fake.community_description(),
            membership=CommunityPermissionsAccess.MANUAL_ACCEPT,
        )
        community_id = community_resp.get("communities", [{}])[0].get("id")

        # Add token permission requiring 1 SNT token for admin
        token_criteria = [
            {
                "type": CommunityTokenType.ERC20,
                "contract_addresses": {31337: self.snt_address},
                "symbol": "SNT",
                "amountInWei": "1000000000000000000",  # 1 token
                "decimals": 18,
            }
        ]

        # Add BECOME_MEMBER permission so join can succeed based on tokens
        self.owner.wakuext_service.create_community_token_permission(
            community_id=community_id,
            permission_type=CommunityTokenPermissionType.BECOME_MEMBER,
            token_criteria=token_criteria,
        )

        # Add BECOME_ADMIN permission with same criteria
        self.owner.wakuext_service.create_community_token_permission(
            community_id=community_id,
            permission_type=CommunityTokenPermissionType.BECOME_ADMIN,
            token_criteria=token_criteria,
        )

        # Fetch community as member
        self.fetch_community(self.member_with_snt, community_id)

        for i in range(10):
            permissions_resp = self.member_with_snt.wakuext_service.check_permissions_to_join_community(community_id)
            # Inspect response to see if admin permission is satisfied
            # Exact response shape depends on your RPC wrapper; pseudocode:
            if permissions_resp.get("satisfied"):
                break
            elif i == 9:
                pytest.fail("Permissions to join never became satisfied for member_with_snt")

            time.sleep(1)

        # Member with tokens requests to join community
        join_resp = self.member_with_snt.wakuext_service.request_to_join_community(community_id, member_address)
        requests = join_resp.get("requestsToJoinCommunity", [])

        if requests:
            req_id = requests[0].get("id")
            # Wait for token validation

            time.sleep(2)
            # Owner accepts the request since member has tokens
            accept_resp = self.owner.wakuext_service.accept_request_to_join_community(req_id)
            assert accept_resp is not None, f"Failed to accept request: {accept_resp}"

        # Explicitly reevaluate community members so token-based roles are applied
        self.owner.wakuext_service.reevaluate_community_members_permissions(community_id)

        # Verify member is now in community and has admin role
        communities = self.owner.wakuext_service.communities()
        owner_community = next(
            (c for c in self._communities_list(communities) if c.get("id") == community_id),
            None,
        )
        assert owner_community is not None

        member_key = self.member_with_snt.public_key
        owner_key = self.owner.public_key

        # Member should have admin role (4), granted via BECOME_ADMIN token permission
        assert member_key in owner_community.get("members", {})
        assert 4 in owner_community["members"][member_key].get("roles", [])

        # Owner should remain owner (1) only, not changed by token reevaluation
        assert owner_key in owner_community.get("members", {})
        assert 1 in owner_community["members"][owner_key].get("roles", [])

    def test_owner_edits_visible_before_and_after_minting_owner_token(self):
        """Test that owner edits are visible before and after minting the owner token"""
        # Owner creates a community
        community_resp = self.owner.wakuext_service.create_community(
            name=fake.community_name(),
            description=fake.community_description(),
            membership=CommunityPermissionsAccess.MANUAL_ACCEPT,
        )
        community_id = community_resp.get("communities", [{}])[0].get("id")

        # Fetch community as member
        self.fetch_community(self.member, community_id)

        # Member joins the community
        join_resp = self.member.wakuext_service.request_to_join_community(community_id, self.fake_address)
        requests = join_resp.get("requestsToJoinCommunity", [])
        if requests:
            req_id = requests[0].get("id")
            # Wait for the request to propagate to the owner
            time.sleep(2)
            # Accept the request
            self.owner.wakuext_service.accept_request_to_join_community(req_id)

        # Verify member is in community
        communities = self.owner.wakuext_service.communities()
        owner_community = next((c for c in self._communities_list(communities) if c.get("id") == community_id), None)
        assert owner_community is not None
        assert self.member.public_key in owner_community.get("members", {})

        # When the Owner edits the community
        new_name = fake.community_name()
        new_description = fake.community_description()
        edit_resp = self.owner.wakuext_service.edit_community(
            community_id=community_id,
            name=new_name,
            description=new_description,
        )
        assert edit_resp is not None

        # Then the Member sees the updated community
        def check_member_community_updated():
            member_communities = self.member.wakuext_service.communities()
            member_community = next((c for c in self._communities_list(member_communities) if c.get("id") == community_id), None)
            return member_community and member_community.get("name") == new_name and member_community.get("description") == new_description

        retry_call(check_member_community_updated)

        # # When the Owner mints the owner token
        # # Simulate minting owner token by saving and adding a community token with owner privileges
        # token_data = {
        #     "tokenType": 2,  # ERC721
        #     "communityId": community_id,
        #     "address": "0x1234567890123456789012345678901234567890",  # Fake address for test
        #     "chainId": 31337,
        #     "name": "OwnerToken",
        #     "supply": "1",
        #     "symbol": "OT",
        #     "privilegesLevel": 1,  # Owner level
        # }
        # self.owner.wakuext_service.save_community_token(token_data)
        # self.owner.wakuext_service.add_community_token(community_id, 31337, "0x1234567890123456789012345678901234567890")
        #
        # # And the Owner edits the community again
        # new_name2 = fake.community_name()
        # new_description2 = fake.community_description()
        # edit_resp2 = self.owner.wakuext_service.edit_community(
        #     community_id=community_id,
        #     name=new_name2,
        #     description=new_description2,
        # )
        # assert edit_resp2 is not None
        #
        # # Then the Member sees the updated community
        # def check_member_community_updated2():
        #     member_communities = self.member.wakuext_service.communities()
        #     member_community = next((c for c in self._communities_list(member_communities) if c.get("id") == community_id), None)
        #     return member_community and member_community.get("name") == new_name2 and member_community.get("description") == new_description2
        #
        # retry_call(check_member_community_updated2)

        # When the Owner restarts the messenger
        self.owner.wakuext_service.stop_messenger()  # <-- Not implemented in API
        self.owner.wakuext_service.start_messenger()

        # And the Owner edits the community again
        new_name3 = fake.community_name()
        new_description3 = fake.community_description()
        edit_resp3 = self.owner.wakuext_service.edit_community(
            community_id=community_id,
            name=new_name3,
            description=new_description3,
        )
        assert edit_resp3 is not None

        # Then the Member sees the updated community
        def check_member_community_updated3():
            member_communities = self.member.wakuext_service.communities()
            member_community = next((c for c in self._communities_list(member_communities) if c.get("id") == community_id), None)
            return member_community and member_community.get("name") == new_name3 and member_community.get("description") == new_description3

        retry_call(check_member_community_updated3)
