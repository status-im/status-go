import pytest
from clients.services.wakuext import CommunityPermissionsAccess
from steps.messenger import MessengerSteps
from utils.retry_utils import retry_call
from utils import fake


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
                "contractAddresses": {1: "0x1234567890123456789012345678901234567890"},
                "symbol": "TEST",
                "amountInWei": "100000000000000000000",  # 100 tokens
                "decimals": 18,
            }
        ]

        permission_resp = owner_backend.wakuext_service.create_community_token_permission(
            community_id=community_id, permission_type=permission_type, token_criteria=token_criteria
        )

        return community_id, permission_resp

    def test_token_gated_community_membership(self):
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
