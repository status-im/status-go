import logging
import time
import uuid
from typing import Optional, List

import pytest

from clients.services.wakuext import CommunityPermissionsAccess, CommunityTokenPermissionType, CommunityTokenType, CommunityRoles
from clients.signals import SignalType, WalletEventType
from clients.status_backend import StatusBackend
from resources.constants import user_1
from resources.enums import RequestToJoinState
from steps.messenger import MessengerSteps
from utils import fake
from utils.retry_utils import retry_call

logger = logging.getLogger(__name__)


def request_to_join_with_signatures(backend: StatusBackend, community_id: str, addresses: list[str]):
    # Generate signatures for joining community with selected addresses to reveal
    # Address must correspond to a non-chat and non-watch account
    sign_params = backend.wakuext_service.generate_joining_community_requests_for_signing(backend.public_key, community_id, addresses)

    # Set password for each sign parameter
    for i in range(len(sign_params)):
        sign_params[i]["password"] = backend.password

    # Sign addresses to reveal
    signatures = backend.wakuext_service.sign_data(sign_params)

    # Send request to join with addresses to reveal and signatures
    return backend.wakuext_service.request_to_join_community(community_id, addresses, signatures)


@pytest.mark.rpc
class TestCommunityTokenPermissions(MessengerSteps):
    @pytest.fixture(autouse=True)
    def setup_tokens(self, snt_addresses, erc721_addresses):
        self.snt_address = snt_addresses["snt"]
        self.snt_controller_address = snt_addresses["controller"]
        self.erc721_address = erc721_addresses["erc721"]

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

    def fund_backend_account_with_tokens(self, backend, foundry_client, amount=10, token_type=CommunityTokenType.ERC20, token_id=0):
        """Fund the given backend's first account with the specified amount of SNT tokens or mint an ERC721 token."""
        accounts = backend.accounts_service.get_accounts()
        assert accounts, "No accounts found"
        assert len(accounts) == 2  # Chat and Wallet accounts

        # Select non-chat account
        wallet_account = next(a for a in accounts if not a.get("chat"))
        assert wallet_account, "No wallet account found"

        member_address = wallet_account["address"]

        if token_type == CommunityTokenType.ERC20:
            token_amount = str(amount * 10**18)  # amount tokens with 18 decimals
            gen_tokens_result = foundry_client.generate_tokens(self.snt_controller_address, member_address, token_amount, user_1.private_key)
            logging.debug(f"Generate tokens result: exit_code={gen_tokens_result.exit_code}, output={gen_tokens_result.output.decode()}")
            logger.debug(f"Funded {member_address} with {amount} SNT tokens at contract {self.snt_address}")

        elif token_type == CommunityTokenType.ERC721:
            mint_result = foundry_client.generate_token_erc721(self.erc721_address, member_address, token_id, user_1.private_key)
            logging.debug(f"Mint ERC721 result: exit_code={mint_result.exit_code}, output={mint_result.output.decode()}")
            logger.debug(f"Minted ERC721 token #{token_id} to {member_address} at contract {self.erc721_address}")

        else:
            raise ValueError(f"Unsupported token_type: {token_type}. Supported types: ERC20, ERC721")

        return member_address

    def verify_token_balance(self, foundry_client, token_type: CommunityTokenType, contract_address, owner_address, min_balance=1, token_id=0):
        """Verify token balance using foundry client. Supports ERC20 and ERC721."""
        if token_type == CommunityTokenType.ERC20:
            balance_result = foundry_client.get_erc20_balance(contract_address, owner_address)
            assert balance_result.exit_code == 0, "Balance check command failed"
            balance = int(balance_result.output.decode().strip(), 16)
            assert balance >= min_balance, f"Insufficient {token_type.name} balance: {balance}, expected at least {min_balance}"

        elif token_type == CommunityTokenType.ERC721:
            assert token_id is not None, "token_id is required for ERC721 verification"
            owner_result = foundry_client.get_erc721_owner(contract_address, token_id)
            assert owner_result.exit_code == 0, "Owner check command failed"
            output = owner_result.output.decode().strip()
            if output.startswith("0x") and len(output) == 66:
                actual_owner = "0x" + output[26:]
            else:
                actual_owner = output
            actual_owner = actual_owner.lower()
            expected_owner = owner_address.lower()
            assert actual_owner == expected_owner, f"Address {owner_address} does not own ERC721 token #{token_id}, owner is {actual_owner}"

        else:
            raise ValueError(f"Unsupported token_type: {token_type}. Supported types: ERC20, ERC721")

    def edit_community(self, owner_backend, community_id):
        new_name = fake.community_name()
        new_description = fake.community_description()
        edit_resp = owner_backend.wakuext_service.edit_community(
            community_id=community_id,
            name=new_name,
            description=new_description,
        )
        assert edit_resp is not None
        return new_name, new_description

    def check_member_community_updated(self, member_backend, community_id: str, expected_name: str, expected_description: str) -> bool:
        member_communities = member_backend.wakuext_service.communities()
        member_community = next((c for c in self._communities_list(member_communities) if c.get("id") == community_id), None)
        return bool(
            member_community and member_community.get("name") == expected_name and member_community.get("description") == expected_description
        )

    def deploy_owner_token(self, owner_backend, community_id, chain_id=11155111):
        """Deploy owner and master tokens for the community, similar to computeDeployOwnerContractsFee logic"""
        accounts = owner_backend.accounts_service.get_accounts()
        wallet_account = next(a for a in accounts if not a.get("chat"))
        address_from = wallet_account["address"]

        # Owner token deployment params
        owner_deployment_params = {
            "name": "Owner Token",
            "symbol": "OT",
            "tokenType": 2,  # ERC721
            "communityId": community_id,
            "supply": "1",
            "decimals": 0,
            "privilegesLevel": 1,  # Owner
            "baseTokenURI": "",
            "receiver": address_from,
            "signerPublicKey": owner_backend.public_key,
        }

        # Master token deployment params
        master_deployment_params = {
            "name": "Master Token",
            "symbol": "MT",
            "tokenType": 2,  # ERC721
            "communityId": community_id,
            "supply": "1",
            "decimals": 0,
            "privilegesLevel": 2,  # Master
            "baseTokenURI": "",
        }

        # Create deployment signature
        signature_result = owner_backend.wakuext_service.create_community_token_deployment_signature(chain_id, address_from, community_id)
        signature = signature_result["signature"]

        # Generate UUID for the transaction
        transaction_uuid = str(uuid.uuid4())

        # Get suggested routes for deploying owner token
        owner_backend.wallet_service.suggested_community_routes(
            uuid=transaction_uuid,
            send_type="CommunityDeployOwnerToken",
            chain_id=chain_id,
            address_from=address_from,
            community_id=community_id,
            signer_pub_key=owner_backend.public_key,
            token_ids=[],
            wallet_addresses=[],
            transfer_details=[],
            signature=signature,
            owner_token_parameters=owner_deployment_params,
            master_token_parameters=master_deployment_params,
        )

        # Build transactions from route
        transaction_data = owner_backend.wallet_service.build_transactions_from_route(transaction_uuid)

        # Sign the transaction
        signatures = owner_backend.wallet_service.sign_message(address_from, owner_backend.password, transaction_data["message"])

        # Send the transaction
        owner_backend.wallet_service.send_router_transactions_with_signatures(transaction_uuid, signatures)

    @pytest.mark.skip(reason="Pending on issue https://github.com/status-im/status-go/issues/7114")
    def test_membership_no_valid_tokens_fake_address(self, owner_backend, member_backend):
        """Test that join request with no tokens/fake address fails permission check (no request created)"""

        # Owner creates token-gated community
        community_id = self.create_token_gated_community(owner_backend, membership=CommunityPermissionsAccess.MANUAL_ACCEPT)

        time.sleep(2)

        # Fetch community as member
        self.fetch_community(member_backend, community_id)

        # Member tries to join without tokens and with fake address - should fail permission check
        fake_address = "0x" + "0" * 40

        # Verify request got rejected right away - no request created
        join_req = request_to_join_with_signatures(member_backend, community_id, [fake_address])
        requests = join_req.get("requestsToJoinCommunity", [])
        assert len(requests) == 0, "No request should get accepted"

        # Verify no declined requests created
        declined_reqs = owner_backend.wakuext_service.declined_requests_to_join_for_community(community_id)
        assert len(declined_reqs) == 0

        # Verify member is not in community
        communities = member_backend.wakuext_service.communities()
        member_community = next((c for c in self._communities_list(communities) if c.get("id") == community_id), None)
        assert member_community is None or not member_community.get("joined", False)

    def test_membership_with_valid_tokens(self, owner_backend, member_with_snt_backend, foundry_client):
        """Test that users with required tokens can successfully join community as member"""

        # Fund the member_with_snt with 10 SNT tokens
        member_address = self.fund_backend_account_with_tokens(member_with_snt_backend, foundry_client)
        self.verify_token_balance(foundry_client, CommunityTokenType.ERC20, self.snt_address, member_address)

        # Owner creates token-gated community with the deployed token
        community_id = self.create_token_gated_community(
            owner_backend,
            permission_types=[CommunityTokenPermissionType.BECOME_MEMBER],
            membership=CommunityPermissionsAccess.MANUAL_ACCEPT,
        )

        # Wait shortly to ensure that we fetch community with token permissions
        # TODO: Remove this sleep somehow
        time.sleep(2)

        # Fetch community as member
        self.fetch_community(member_with_snt_backend, community_id)

        # Member tries to join with their wallet address (should have tokens)
        join_req = request_to_join_with_signatures(member_with_snt_backend, community_id, [member_address])
        requests = join_req.get("requestsToJoinCommunity", [])
        assert requests, "No requests to join community"
        assert len(requests) == 1, "Unexpected multiple requests to join community"

        req_id = requests[0].get("id")
        logger.info(f"Sent request to join community {community_id} with id {req_id}")

        # Wait for the request to join to be received
        owner_backend.wait_for_signal_predicate(
            signal_type=SignalType.MESSAGES_NEW,
            predicate=lambda s: (s.get("event", {})["requestsToJoinCommunity"][0]["id"] == req_id),
        )

        # Accept request to join
        owner_backend.wakuext_service.accept_request_to_join_community(req_id)

        # Verify member is now in community (check from owner's perspective since acceptance happened there)
        communities = owner_backend.wakuext_service.communities()
        owner_community = next((c for c in self._communities_list(communities) if c.get("id") == community_id), None)
        assert owner_community is not None

        # Check that member is in the community members list
        member_public_key = member_with_snt_backend.public_key
        assert member_public_key in owner_community.get("members", {}), f"Member {member_public_key} not found in community members"

    def test_admin_token_permissions_with_valid_tokens(self, owner_backend, member_with_snt_backend, foundry_client):
        """Test that users with required tokens get admin privileges"""

        # Fund the member_with_snt with 10 SNT tokens
        member_address = self.fund_backend_account_with_tokens(member_with_snt_backend, foundry_client)
        self.verify_token_balance(foundry_client, CommunityTokenType.ERC20, self.snt_address, member_address)

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
        time.sleep(2)

        # Fetch community as member
        response = self.fetch_community(member_with_snt_backend, community_id)
        assert response, "Community not found"
        assert response["tokenPermissions"], "No token permissions found"
        assert len(response["tokenPermissions"]) == 2, "Unexpected number of token permissions"

        # Wait for balance to be fetched (request to join uses cached balances)
        member_with_snt_backend.wallet_service.restart_wallet_reload_timer()  # Force balance fetch
        member_with_snt_backend.wait_for_signal_predicate(
            SignalType.WALLET,
            lambda signal: signal["event"]["type"] == WalletEventType.WALLET_TICK_RELOAD.value,
            timeout=20,  # 10 seconds backoff + timeout
        )

        permissions_resp = member_with_snt_backend.wakuext_service.check_permissions_to_join_community(community_id)
        assert permissions_resp, "Failed to check permissions to join community"
        assert permissions_resp.get("satisfied"), "Permissions to join are not satisfied"

        # Member with tokens requests to join community
        join_req = request_to_join_with_signatures(member_with_snt_backend, community_id, [member_address])
        requests = join_req.get("requestsToJoinCommunity", [])
        assert requests, "No requests to join community"
        assert len(requests) == 1, "Unexpected multiple requests to join community"

        req_id = requests[0].get("id")
        logger.info(f"Sent request to join community {community_id} with id {req_id}")

        # Wait for request to join to get received
        owner_backend.wait_for_signal_predicate(
            SignalType.MESSAGES_NEW,
            lambda signal: (signal.get("event", {})["requestsToJoinCommunity"][0]["id"] == req_id),
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

    # @pytest.mark.skip(reason="Pending on issue https://github.com/status-im/status-go/issues/7167")
    def test_owner_edits_visible_before_and_after_minting_owner_token(self, sepolia_owner_backend, sepolia_member_backend):
        """Test that owner edits are visible before and after minting the owner token"""

        # Owner creates a community
        community_id = self.create_token_gated_community(
            sepolia_owner_backend,
            permission_types=[CommunityTokenPermissionType.BECOME_MEMBER],
            token_criteria=[],
            membership=CommunityPermissionsAccess.MANUAL_ACCEPT,
        )

        # Fetch community as member
        response = self.fetch_community(sepolia_member_backend, community_id)
        assert response, "Community not found"

        permissions_resp = sepolia_member_backend.wakuext_service.check_permissions_to_join_community(community_id)
        assert permissions_resp, "Failed to check permissions to join community"
        assert permissions_resp.get("satisfied"), "Permissions to join are not satisfied"

        # Member requests to join community
        join_resp = sepolia_member_backend.wakuext_service.request_to_join_community(community_id, [fake.address()])
        requests = join_resp.get("requestsToJoinCommunity", [])
        assert requests, "No requests to join community"
        assert len(requests) == 1, "Unexpected multiple requests to join community"

        req_id = requests[0].get("id")
        # Wait for member validation
        sepolia_owner_backend.wait_for_signal(
            SignalType.MESSAGES_NEW,
            lambda signal: signal.get("event", {}).get("requestsToJoinCommunity")[0].get("state")
            == RequestToJoinState.RequestToJoinStatePending.value,
        )

        time.sleep(2)

        accept_resp = sepolia_owner_backend.wakuext_service.accept_request_to_join_community(req_id)
        assert accept_resp is not None, f"Failed to accept request: {accept_resp}"

        # Verify member is in community
        communities = sepolia_owner_backend.wakuext_service.communities()
        owner_community = next((c for c in self._communities_list(communities) if c.get("id") == community_id), None)
        assert owner_community is not None
        assert sepolia_member_backend.public_key in owner_community.get("members", {})

        # When the Owner edits the community
        new_name, new_description = self.edit_community(sepolia_owner_backend, community_id)
        logger.info(f"New name: {new_name}, new description2: {new_description}")

        # Then the Member sees the updated community
        retry_call(self.check_member_community_updated, sepolia_member_backend, community_id, new_name, new_description)

        # When the Owner mints the owner token
        self.deploy_owner_token(sepolia_owner_backend, community_id)

        # And the Owner edits the community again
        new_name2, new_description2 = self.edit_community(sepolia_owner_backend, community_id)
        logger.info(f"New name2: {new_name2}, new description2: {new_description2}")

        # Then the Member sees the updated community
        retry_call(self.check_member_community_updated, sepolia_member_backend, community_id, new_name2, new_description2)

        # When the Owner logouts and logs back in
        sepolia_owner_backend.logout()
        sepolia_owner_backend.login(sepolia_owner_backend.key_uid, sepolia_owner_backend.password)
        sepolia_owner_backend.wait_for_login()
        sepolia_owner_backend.wakuext_service.start_messenger()

        # And the Owner edits the community again
        new_name3, new_description3 = self.edit_community(sepolia_owner_backend, community_id)

        # Then the Member sees the updated community
        retry_call(self.check_member_community_updated, sepolia_member_backend, community_id, new_name3, new_description3)
