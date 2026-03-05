import logging
import time
import uuid
import json
from typing import Optional, List


import pytest

from clients.api import ApiResponseError
from clients.services.wakuext import CommunityPermissionsAccess, CommunityTokenPermissionType, CommunityTokenType, CommunityRoles
from clients.signals import SignalType
from clients.status_backend import StatusBackend
from resources.constants import user_1
from steps.messenger import MessengerSteps
from utils import fake
from utils.keys import change_community_key_compression

logger = logging.getLogger(__name__)

COMMUNITY_DEPLOY_OWNER_TOKEN = 12


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
    def setup_tokens(self, snt_addresses):
        self.snt_address = snt_addresses["snt"]
        self.snt_controller_address = snt_addresses["controller"]

    @pytest.fixture(autouse=True)
    def get_communities_contracts(self, communities_addresses):
        self.community_token_deployer = next(
            info["value"] for info in communities_addresses.values() if info["internal_type"] == "contract CommunityTokenDeployer"
        )

    def _communities_list(self, communities_response):
        if isinstance(communities_response, dict):
            return communities_response.get("communities", []) or []
        return communities_response or []

    def _spectate_and_fetch_community(self, backend, community_id, attempts=10, delay=5):
        # Fetch first so community exists in local DB before spectating.
        # `wakuext_spectateCommunity` returns "community not found" when called before fetch/sync.
        community = None
        for attempt in range(attempts):
            community = self.fetch_community(backend, community_id)
            if community:
                break
            logging.info(f"Community {community_id} not found yet (attempt {attempt + 1}/{attempts})")
            time.sleep(delay)

        if not community:
            return None

        try:
            backend.wakuext_service.spectate_community(community_id)
        except ApiResponseError as exc:
            logging.warning(f"Spectate community failed for {community_id}: {exc}")

        return community

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

        else:
            raise ValueError(f"Unsupported token_type: {token_type}. Supported types: ERC20")

        return member_address

    def verify_token_balance(self, foundry_client, token_type: CommunityTokenType, contract_address, owner_address, min_balance=1, token_id=0):
        """Verify token balance using foundry client. Supports ERC20 and ERC721."""
        if token_type == CommunityTokenType.ERC20:
            balance_result = foundry_client.get_erc20_balance(contract_address, owner_address)
            assert balance_result.exit_code == 0, "Balance check command failed"
            balance = int(balance_result.output.decode().strip(), 16)
            assert balance >= min_balance, f"Insufficient {token_type.name} balance: {balance}, expected at least {min_balance}"

        else:
            raise ValueError(f"Unsupported token_type: {token_type}. Supported types: ERC20")

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

    def deploy_owner_token(self, owner_backend, community_id):
        """Deploy and mint owner token for the community"""

        accounts = owner_backend.accounts_service.get_accounts()
        assert accounts, "No accounts found"
        wallet_account = next(a for a in accounts if not a.get("chat"))
        assert wallet_account, "No wallet account found"
        address_from = wallet_account["address"]

        # CommunityTokenDeployer contract address
        contract_address = self.community_token_deployer

        # Generate deployment signature
        chain_id = owner_backend.network_id
        signature = owner_backend.wakuext_service.create_community_token_deployment_signature(chain_id, address_from, community_id)
        logger.info(f"Community token deployment signature: {signature}")

        token_uri = change_community_key_compression(community_id) + "/"

        # Owner and master token parameters for deployment
        owner_token_parameters = {
            "name": "TestOwnerToken",
            "symbol": "TOT",
            "tokenUri": token_uri,
            "supply": 1,
            "infiniteSupply": False,
            "decimals": 0,
            "transferable": True,
            "remoteSelfDestruct": False,
            "base64image": "data:image/gif;base64,R0lGODlhAQABAIAAAP///wAAACwAAAAAAQABAAACAkQBADs=",
        }
        master_token_parameters = {
            "name": "TestMasterToken",
            "symbol": "TMT",
            "tokenUri": token_uri,
            "infiniteSupply": True,
            "decimals": 0,
            "transferable": True,
            "remoteSelfDestruct": True,
            "base64image": "data:image/gif;base64,R0lGODlhAQABAIAAAP///wAAACwAAAAAAQABAAACAkQBADs=",
        }

        # Generate UUID for the transaction
        transaction_uuid = str(uuid.uuid4())

        # Fetch balances
        owner_backend.wallet_service.get_balances_at_by_chain([chain_id], [address_from], [])

        # Get suggested routes for deploying tokens
        signer_pub_key = owner_backend.public_key

        routes_result = owner_backend.wallet_service.suggested_community_routes(
            uuid=transaction_uuid,
            send_type=COMMUNITY_DEPLOY_OWNER_TOKEN,
            chain_id=chain_id,
            address_from=address_from,
            addr_to=contract_address,
            community_id=community_id,
            signer_pub_key=signer_pub_key,
            token_ids=[],
            wallet_addresses=[],
            transfer_details=[],
            signature=signature,
            owner_token_parameters=owner_token_parameters,
            master_token_parameters=master_token_parameters,
        )

        assert routes_result.get("Uuid") == transaction_uuid, f"Expected UUID {transaction_uuid}, got {routes_result.get('uuid')}"

        # Build transactions from route (async, sends SignRouterTransactions signal)
        with owner_backend.expect_signal(
            SignalType.WALLET_ROUTER_SIGN_TRANSACTIONS,
            predicate=lambda s: s.get("event", {}).get("sendDetails", {}).get("uuid") == transaction_uuid,
            timeout=60,
        ) as sign_exp:
            owner_backend.wallet_service.build_transactions_from_route(transaction_uuid)

        sign_signal = sign_exp.result

        signing_details = sign_signal["event"]["signingDetails"]
        signatures = {}
        for tx_hash in signing_details["hashes"]:
            sig_hex = owner_backend.wallet_service.sign_message(tx_hash, address_from, owner_backend.password)
            assert sig_hex and sig_hex.startswith("0x"), f"Invalid transaction signature for hash {tx_hash}: {sig_hex}"
            tx_signature = sig_hex[2:]
            signatures[tx_hash] = {
                "r": tx_signature[:64],
                "s": tx_signature[64:128],
                "v": tx_signature[128:],
            }

        # Send the signed transactions
        with owner_backend.expect_signal(
            SignalType.WALLET_ROUTER_TRANSACTIONS_SENT,
            timeout=60,
        ) as sent_exp:
            owner_backend.wallet_service.send_router_transactions_with_signatures(transaction_uuid, signatures)

        sent_signal = sent_exp.result

        logger.debug(f"Sent router transactions for UUID {transaction_uuid}: {sent_signal}")

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

        member_community = self._spectate_and_fetch_community(member_with_snt_backend, community_id)
        assert member_community, "Community not found on member"

        req_id = None
        # Member tries to join with their wallet address (should have tokens)
        with owner_backend.expect_signal(
            SignalType.MESSAGES_NEW,
            predicate=lambda s: ((s.get("event", {}).get("requestsToJoinCommunity") or [{}])[0].get("id") == req_id),
            timeout=60,
        ):
            join_req = request_to_join_with_signatures(member_with_snt_backend, community_id, [member_address])
            requests = join_req.get("requestsToJoinCommunity", [])
            assert requests, "No requests to join community"
            assert len(requests) == 1, "Unexpected multiple requests to join community"

            req_id = requests[0].get("id")
            logger.info(f"Sent request to join community {community_id} with id {req_id}")

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
        response = self._spectate_and_fetch_community(member_with_snt_backend, community_id)
        assert response, "Community not found"
        assert response["tokenPermissions"], "No token permissions found"
        assert len(response["tokenPermissions"]) == 2, "Unexpected number of token permissions"

        # Wait for balance to be fetched (request to join uses cached balances)
        # Trigger explicit refresh and wait until permission check sees the balance
        member_with_snt_backend.wallet_service.fetch_or_get_cached_wallet_balances([member_address], True)
        permissions_resp = None
        for attempt in range(3):
            time.sleep(2)
            permissions_resp = member_with_snt_backend.wakuext_service.check_permissions_to_join_community(community_id)
            if permissions_resp and permissions_resp.get("satisfied"):
                break

            # Retry by forcing refresh again (balance update is async)
            member_with_snt_backend.wallet_service.fetch_or_get_cached_wallet_balances([member_address], True)

        assert permissions_resp, "Failed to check permissions to join community"
        assert permissions_resp.get("satisfied"), "Permissions to join are not satisfied"

        req_id = None
        # Member with tokens requests to join community
        with owner_backend.expect_signal(
            SignalType.MESSAGES_NEW,
            predicate=lambda signal: any(r.get("id") == req_id for r in (signal.get("event", {}).get("requestsToJoinCommunity") or [])),
            timeout=60,
        ):
            join_req = request_to_join_with_signatures(member_with_snt_backend, community_id, [member_address])
            requests = join_req.get("requestsToJoinCommunity", [])
            assert requests, "No requests to join community"
            assert len(requests) == 1, "Unexpected multiple requests to join community"

            req_id = requests[0].get("id")
            logger.info(f"Sent request to join community {community_id} with id {req_id}")

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

    def test_owner_edits_visible_before_and_after_minting_owner_token(self, owner_backend, member_backend, foundry_client, anvil_client):
        """Test that owner edits are visible before and after minting the owner token"""

        # Owner creates a token-gated community
        community_id = self.create_token_gated_community(
            owner_backend,
            permission_types=[CommunityTokenPermissionType.BECOME_MEMBER],
            membership=CommunityPermissionsAccess.MANUAL_ACCEPT,
        )

        # Fetch/spectate community as member and wait until it is available with propagated metadata
        time.sleep(2)
        response = self._spectate_and_fetch_community(member_backend, community_id)
        assert response, "Community not found"

        # Fund member and verify token balance so member satisfies token-gated permission
        member_address = self.fund_backend_account_with_tokens(member_backend, foundry_client)
        self.verify_token_balance(foundry_client, CommunityTokenType.ERC20, self.snt_address, member_address)

        # Balance and permission checks are async/cached, force refresh and retry.
        # `community.memberReevaluationStatus` is not reliably emitted in this path.
        member_backend.wallet_service.fetch_or_get_cached_wallet_balances([member_address], True)
        permissions_resp = None
        for _ in range(10):
            time.sleep(2)
            # Refresh community view too, token-permission propagation can lag behind fetchCommunity
            self.fetch_community(member_backend, community_id)
            permissions_resp = member_backend.wakuext_service.check_permissions_to_join_community(community_id)
            if permissions_resp and permissions_resp.get("satisfied"):
                break

            member_backend.wallet_service.fetch_or_get_cached_wallet_balances([member_address], True)

        assert permissions_resp, "Failed to check permissions to join community"
        assert permissions_resp.get("satisfied"), "Permissions to join are not satisfied"

        req_id = None

        # Wait for member validation
        with owner_backend.expect_signal(
            SignalType.MESSAGES_NEW,
            predicate=lambda signal: any(r.get("id") == req_id for r in (signal.get("event", {}).get("requestsToJoinCommunity") or [])),
            timeout=60,
        ):
            join_resp = request_to_join_with_signatures(member_backend, community_id, [member_address])
            requests = join_resp.get("requestsToJoinCommunity", [])

            assert requests, "No requests to join community"
            assert len(requests) == 1, "Unexpected multiple requests to join community"

            req_id = requests[0].get("id")

        time.sleep(2)

        accept_resp = owner_backend.wakuext_service.accept_request_to_join_community(req_id)
        assert accept_resp is not None, f"Failed to accept request: {accept_resp}"

        # Verify member is in community
        communities = owner_backend.wakuext_service.communities()
        owner_community = next((c for c in self._communities_list(communities) if c.get("id") == community_id), None)
        assert owner_community is not None
        assert member_backend.public_key in owner_community.get("members", {})

        # When the Owner edits the community
        with member_backend.expect_signal(
            SignalType.MESSAGES_NEW,
            predicate=lambda signal: community_id in json.dumps(signal),
            timeout=60,
        ):
            new_name, new_description = self.edit_community(owner_backend, community_id)

        # Then the Member sees the updated community
        assert self.check_member_community_updated(member_backend, community_id, new_name, new_description)

        # Fund owner wallet with native token on Anvil to cover gas for owner-token deployment
        owner_accounts = owner_backend.accounts_service.get_accounts()
        assert owner_accounts, "No owner accounts found"
        owner_wallet_account = next(a for a in owner_accounts if not a.get("chat"))
        assert owner_wallet_account, "No owner wallet account found"
        owner_address = owner_wallet_account["address"]

        ten_native_tokens_in_wei = 10 * 10**18
        anvil_client.set_balance(owner_address, ten_native_tokens_in_wei)

        # Refresh wallet balances cache after forcing Anvil balance
        owner_backend.wallet_service.fetch_or_get_cached_wallet_balances([owner_address], True)
        owner_backend.wallet_service.get_balances_at_by_chain([owner_backend.network_id], [owner_address], [])

        # When the Owner mints the owner token
        owner_backend.wallet_service.restart_wallet_reload_timer()
        time.sleep(2)  # Sync metadata
        self.deploy_owner_token(owner_backend, community_id)

        # And the Owner edits the community again
        with member_backend.expect_signal(
            SignalType.MESSAGES_NEW,
            predicate=lambda signal: community_id in json.dumps(signal),
            timeout=60,
        ):
            new_name2, new_description2 = self.edit_community(owner_backend, community_id)

        # Then the Member sees the updated community
        assert self.check_member_community_updated(member_backend, community_id, new_name2, new_description2)

        # When the Owner logouts and logs back in
        owner_backend.logout()
        owner_backend.login(owner_backend.key_uid, owner_backend.password)
        owner_backend.wait_for_login()
        owner_backend.wait_for_wakuext_ready(timeout=60, start_messenger=True)

        # And the Owner edits the community again
        with member_backend.expect_signal(
            SignalType.MESSAGES_NEW,
            predicate=lambda signal: community_id in json.dumps(signal),
            timeout=60,
        ):
            new_name3, new_description3 = self.edit_community(owner_backend, community_id)

        # Then the Member sees the updated community
        assert self.check_member_community_updated(member_backend, community_id, new_name3, new_description3)
