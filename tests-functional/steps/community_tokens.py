import logging
import time
from typing import Optional, List

from clients.services.wakuext import (
    CommunityPermissionsAccess,
    CommunityTokenPermissionType,
    CommunityTokenType,
)
from clients.signals import SignalType
from clients.status_backend import StatusBackend
from resources.constants import user_1
from steps import messenger
from utils import fake

logger = logging.getLogger(__name__)

NATIVE_TOKEN_ADDRESS = "0x0000000000000000000000000000000000000000"


def request_to_join_with_signatures(backend: StatusBackend, community_id: str, addresses: list[str]):
    """Generate signatures for joining a community and submit the request."""
    sign_params = backend.wakuext_service.generate_joining_community_requests_for_signing(backend.public_key, community_id, addresses)

    for i in range(len(sign_params)):
        sign_params[i]["password"] = backend.password

    signatures = backend.wakuext_service.sign_data(sign_params)

    return backend.wakuext_service.request_to_join_community(community_id, addresses, signatures)


def create_token_gated_community(
    owner_backend,
    snt_address: str,
    permission_types: Optional[List[CommunityTokenPermissionType]] = None,
    token_criteria: Optional[List[dict]] = None,
    membership: CommunityPermissionsAccess = CommunityPermissionsAccess.AUTO_ACCEPT,
):
    """Create a community and add token permissions."""
    if permission_types is None:
        permission_types = [CommunityTokenPermissionType.BECOME_MEMBER]
    if token_criteria is None:
        token_criteria = [
            {
                "type": CommunityTokenType.ERC20.value,
                "contract_addresses": {31337: snt_address},
                "symbol": "SNT",
                "amountInWei": "1",
                "decimals": 18,
            }
        ]

    community_resp = owner_backend.wakuext_service.create_community(
        name=fake.community_name(),
        description=fake.community_description(),
        membership=membership,
    )
    community_id = community_resp.get("communities", [{}])[0].get("id")

    for permission_type in permission_types:
        owner_backend.wakuext_service.create_community_token_permission(
            community_id=community_id, permission_type=permission_type, token_criteria=token_criteria
        )

    return community_id


def fund_backend_account_with_tokens(backend, foundry_client, snt_controller_address: str, snt_address: str, amount=10):
    """Fund the given backend's first wallet account with ERC20 (SNT) tokens."""
    accounts = backend.accounts_service.get_accounts()
    assert accounts, "No accounts found"
    assert len(accounts) == 2  # Chat and Wallet accounts

    member_address = messenger.wallet_address(backend)

    token_amount = str(amount * 10**18)
    gen_tokens_result = foundry_client.generate_tokens(snt_controller_address, member_address, token_amount, user_1.private_key)
    logging.debug(f"Generate tokens result: exit_code={gen_tokens_result.exit_code}, output={gen_tokens_result.output.decode()}")
    logger.debug(f"Funded {member_address} with {amount} SNT tokens at contract {snt_address}")

    return member_address


def fund_native_balance(backend: StatusBackend, anvil_client, amount_in_wei: int = 10 * 10**18):
    """Set native ETH balance for the backend's wallet account via anvil."""
    address = messenger.wallet_address(backend)
    anvil_client.set_balance(address, amount_in_wei)
    backend.wallet_service.fetch_or_get_cached_wallet_balances([address], True)
    backend.wallet_service.get_balances_at_by_chain([address], [f"{backend.network_id}-{NATIVE_TOKEN_ADDRESS}"])
    return address


def verify_token_balance(foundry_client, token_type: CommunityTokenType, contract_address, owner_address, min_balance=1):
    """Verify token balance using foundry client."""
    if token_type == CommunityTokenType.ERC20:
        balance_result = foundry_client.get_erc20_balance(contract_address, owner_address)
        assert balance_result.exit_code == 0, "Balance check command failed"
        balance = int(balance_result.output.decode().strip(), 16)
        assert balance >= min_balance, f"Insufficient {token_type.name} balance: {balance}, expected at least {min_balance}"
    else:
        raise ValueError(f"Unsupported token_type: {token_type}. Supported types: ERC20")


def join_community_with_signatures_and_accept(owner_backend, member_backend, community_id: str, member_wallet_address: str):
    """Join a token-gated community with signatures and have the owner accept the request."""
    req_id = None
    with owner_backend.expect_signal(
        SignalType.MESSAGES_NEW,
        predicate=lambda signal: any(r.get("id") == req_id for r in (signal.get("event", {}).get("requestsToJoinCommunity") or [])),
        timeout=60,
    ):
        join_resp = request_to_join_with_signatures(member_backend, community_id, [member_wallet_address])
        requests = join_resp.get("requestsToJoinCommunity", [])
        assert requests, "No requests to join community"
        assert len(requests) == 1, "Unexpected multiple requests to join community"
        req_id = requests[0].get("id")

    time.sleep(2)
    accept_resp = owner_backend.wakuext_service.accept_request_to_join_community(req_id)
    assert accept_resp is not None, f"Failed to accept request: {accept_resp}"


def wait_for_member_role(
    backend,
    community_id: str,
    member_key: str,
    role: int,
    attempts: int = 10,
    delay: int = 2,
    fetch_from_store: bool = False,
    required: bool = True,
):
    """Poll until *backend* sees *role* on the member identified by *member_key*.

    When *fetch_from_store* is True, a live store-node fetch is forced on
    each iteration so the backend's local state is updated with the latest
    community description.

    Returns the community dict when the role is found, or the last observed
    community dict when *required* is False and the role was not seen.
    Raises AssertionError when *required* is True and the role is missing
    after all attempts.
    """
    community = None
    for _ in range(attempts):
        if fetch_from_store:
            messenger.fetch_community(backend, community_id, try_database=False)
        communities = backend.wakuext_service.communities()
        community = next(
            (c for c in messenger.communities_list(communities) if c.get("id") == community_id),
            None,
        )
        roles = (community or {}).get("members", {}).get(member_key, {}).get("roles", [])
        if role in roles:
            return community
        time.sleep(delay)

    if required:
        assert community is not None, "Community not found"
        roles = community.get("members", {}).get(member_key, {}).get("roles", [])
        assert role in roles, f"Member {member_key} did not receive role {role} after {attempts} attempts"
    else:
        logger.warning(f"Backend local state may not yet reflect role {role} for {member_key}; proceeding")
    return community
