import logging
import time
from typing import Optional, List

import pytest

from clients.services.wakuext import CommunityPermissionsAccess, CommunityTokenPermissionType, CommunityTokenType, CommunityRoles
from clients.signals import SignalType
from resources.constants import user_1
from resources.enums import RequestToJoinState
from steps.messenger import MessengerSteps
from utils import fake
from utils.retry_utils import retry_call

logger = logging.getLogger(__name__)


@pytest.mark.rpc
@pytest.mark.usefixtures("snt_deployment")
class TestCommunityTokenPermissions(MessengerSteps):
    snt_address: Optional[str] = None
    snt_controller_address: Optional[str] = None

    def _communities_list(self, communities_response):
        if isinstance(communities_response, dict):
            return communities_response.get("communities", []) or []
        return communities_response or []

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

    def fund_backend_account_with_snt(self, backend, foundry_client, amount=10):
        """Fund the given backend's first account with the specified amount of SNT tokens."""
        accounts = backend.accounts_service.get_accounts()
        member_address = accounts[0]["address"]
        token_amount = str(amount * 10**18)  # amount tokens with 18 decimals
        gen_tokens_result = foundry_client.generate_tokens(self.snt_controller_address, member_address, token_amount, user_1.private_key)
        logging.debug(f"Generate tokens result: exit_code={gen_tokens_result.exit_code}, output={gen_tokens_result.output.decode()}")
        logger.debug(f"Funded {member_address} with {amount} SNT tokens at contract {self.snt_address}")

        return member_address

    def verify_snt_balance(self, foundry_client, member_address, min_wei: int = 1000000000000000000):
        """Verify the SNT balance using foundry client."""
        balance_result = foundry_client.get_erc20_balance(self.snt_address, member_address)
        assert balance_result.exit_code == 0, "Balance check command failed"
        balance = int(balance_result.output.decode().strip(), 16)
        assert balance >= min_wei, f"Insufficient SNT balance: {balance}, expected at least 1 token"

    @pytest.mark.skip(reason="Pending on issue https://github.com/status-im/status-go/issues/7114")
    def test_membership_no_valid_tokens(self, owner_backend, member_backend):
        """Test that users must hold required tokens to join community"""

        # Owner creates token-gated community
        community_id = self.create_token_gated_community(owner_backend, membership=CommunityPermissionsAccess.MANUAL_ACCEPT)

        # Fetch community as member
        self.fetch_community(member_backend, community_id)

        # Member tries to join without tokens - should fail at request time
        fake_address = "0x" + "0" * 40
        join_req = member_backend.wakuext_service.request_to_join_community(community_id, fake_address)
        requests = join_req.get("requestsToJoinCommunity", [])
        if requests:
            # If request was created, check that it gets declined
            req_id = requests[0].get("id")
            # Check that request is declined due to insufficient permissions
            owner_backend.wait_for_signal(SignalType.MESSAGES_NEW)
            declined_reqs = owner_backend.wakuext_service.declined_requests_to_join_for_community(community_id)
            assert len(declined_reqs) == 1
            assert declined_reqs[0].get("id") == req_id
        else:
            # Request was rejected at creation time
            pass

        # Verify member is not in community
        communities = member_backend.wakuext_service.communities()
        member_community = next((c for c in self._communities_list(communities) if c.get("id") == community_id), None)
        assert member_community is None or not member_community.get("joined", False)

    @pytest.mark.skip(reason="Pending on issue https://github.com/status-im/status-go/issues/7161")
    def test_membership_with_valid_tokens(self, owner_backend, member_with_snt_backend, foundry_client):
        """Test that users with required tokens can successfully join community as member"""

        # Fund the member_with_snt with 10 SNT tokens
        member_address = self.fund_backend_account_with_snt(member_with_snt_backend, foundry_client)
        self.verify_snt_balance(foundry_client, member_address)

        # Owner creates token-gated community with the deployed token
        community_id = self.create_token_gated_community(
            owner_backend,
            permission_types=[CommunityTokenPermissionType.BECOME_MEMBER],
            membership=CommunityPermissionsAccess.MANUAL_ACCEPT,
        )

        # Fetch community as member
        self.fetch_community(member_with_snt_backend, community_id)

        # Member tries to join with their wallet address (should have tokens)
        join_req = member_with_snt_backend.wakuext_service.request_to_join_community(community_id, member_address)
        requests = join_req.get("requestsToJoinCommunity", [])

        if requests:
            req_id = requests[0].get("id")
            # Since member has valid tokens, request should not be declined
            # Wait for token validation to complete, then try to accept directly

            received_signal = owner_backend.wait_for_signal(SignalType.COMMUNITY_MEMBER_REEVALUATION_STATUS)
            logger.info(f"Received signal: {received_signal}")

            def try_accept_request(req_id):
                resp = owner_backend.wakuext_service.accept_request_to_join_community(req_id)
                return resp if resp is not None else None

            accept_resp = retry_call(try_accept_request, req_id)

            # accept_resp = owner_backend.wakuext_service.accept_request_to_join_community(req_id)
            assert accept_resp is not None, f"Failed to accept request: {accept_resp}"

        # Verify member is now in community (check from owner's perspective since acceptance happened there)
        communities = owner_backend.wakuext_service.communities()
        owner_community = next((c for c in self._communities_list(communities) if c.get("id") == community_id), None)
        assert owner_community is not None
        # Check that member is in the community members list
        member_public_key = member_with_snt_backend.public_key
        assert member_public_key in owner_community.get("members", {}), f"Member {member_public_key} not found in community members"

    @pytest.mark.skip(reason="Pending on issue https://github.com/status-im/status-go/issues/7135")
    def test_admin_token_permissions_with_valid_tokens(self, owner_backend, member_with_snt_backend, foundry_client):
        """Test that users with required tokens get admin privileges"""

        # Fund the member_with_snt with 10 SNT tokens
        member_address = self.fund_backend_account_with_snt(member_with_snt_backend, foundry_client)
        self.verify_snt_balance(foundry_client, member_address)

        # Owner creates token-gated community with member and admin permissions
        community_id = self.create_token_gated_community(
            owner_backend,
            permission_types=[
                CommunityTokenPermissionType.BECOME_MEMBER,
                CommunityTokenPermissionType.BECOME_ADMIN,
            ],
            membership=CommunityPermissionsAccess.MANUAL_ACCEPT,
        )

        # We should give some time for the token permissions to be distributed over network to store nodes.
        # If we don't wait enough, community can be found without token permissions (or with only 1).
        # Proper way would be this:
        # 1. wakuext_SpectateCommunity - this makes us subscribe to community updates
        # 2. if token permissions are not present,
        #    wait for `messages.new` with community update and token permissions. (might need skip a few signals)
        time.sleep(5)

        # Fetch community as member
        response = self.fetch_community(member_with_snt_backend, community_id)
        assert response, "Community not found"
        assert response["tokenPermissions"], "No token permissions found"
        assert len(response["tokenPermissions"]) == 2, "Unexpected number of token permissions"  # FIXME: Fails here

        # member_with_snt_backend.wait_for_signal(SignalType.COMMUNITY_MEMBER_REEVALUATION_STATUS)
        permissions_resp = member_with_snt_backend.wakuext_service.check_permissions_to_join_community(community_id)
        assert permissions_resp, "Failed to check permissions to join community"
        assert permissions_resp.get("satisfied"), "Permissions to join are not satisfied"

        # Member with tokens requests to join community
        join_resp = member_with_snt_backend.wakuext_service.request_to_join_community(community_id, member_address)
        requests = join_resp.get("requestsToJoinCommunity", [])
        assert requests, "No requests to join community"
        assert len(requests) == 1, "Unexpected multiple requests to join community"

        req_id = requests[0].get("id")
        # Wait for token validation
        owner_backend.wait_for_signal(
            SignalType.MESSAGES_NEW,
            lambda signal: signal.get("event", {}).get("requestsToJoinCommunity")[0].get("state")
            == RequestToJoinState.RequestToJoinStatePending.value,
        )

        accept_resp = owner_backend.wakuext_service.accept_request_to_join_community(req_id)
        assert accept_resp is not None, f"Failed to accept request: {accept_resp}"

        # Verify member is now in community and has admin role
        communities = owner_backend.wakuext_service.communities()
        owner_community = next(
            (c for c in self._communities_list(communities) if c.get("id") == community_id),
            None,
        )
        assert owner_community is not None

        member_key = member_with_snt_backend.public_key
        owner_key = owner_backend.public_key

        # Member should have admin role, granted via BECOME_ADMIN token permission
        assert member_key in owner_community.get("members", {})
        assert CommunityRoles.ROLE_ADMIN.value in owner_community["members"][member_key].get("roles", [])

        # Owner should remain owner only, not changed by token reevaluation
        assert owner_key in owner_community.get("members", {})
        assert CommunityRoles.ROLE_OWNER.value in owner_community["members"][owner_key].get("roles", [])
