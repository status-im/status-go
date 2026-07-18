import json
import logging
import time
from typing import Optional, List

from clients.services.wakuext import (
    CommunityPermissionsAccess,
    CommunityTokenPermissionType,
    CommunityTokenType,
)
from clients.signals import CommunityMemberReevaluationStatus, SignalType
from clients.status_backend import StatusBackend
from resources.constants import user_1
from steps import messenger
from utils import fake

logger = logging.getLogger(__name__)

NATIVE_TOKEN_ADDRESS = "0x0000000000000000000000000000000000000000"


def community_permission_types(community: dict) -> set[int]:
    return {p.get("type") for p in (community.get("tokenPermissions") or {}).values()}


def community_has_expected_details(community: Optional[dict], expected_name: str, expected_description: str) -> bool:
    return bool(community and community.get("name") == expected_name and community.get("description") == expected_description)


def check_member_community_updated(
    backend: StatusBackend,
    community_id: str,
    expected_name: str,
    expected_description: str,
    community: Optional[dict] = None,
    include_live_fetch: bool = True,
) -> bool:
    """Return True when any fetched or cached community view matches the expected details.

    With include_live_fetch=False only the local DB is read (non-blocking). Use it where a live fetch
    (wait_for_response=True) can block indefinitely under store contention.
    """
    sources: List[Optional[dict]] = [community]

    # (wait_for_response, try_database) per fetch: live + db-with-fallback normally, else a non-blocking db read.
    fetch_variants = [(True, False), (True, True)] if include_live_fetch else [(False, True)]
    for wait_for_response, try_database in fetch_variants:
        try:
            sources.append(
                messenger.fetch_community(
                    backend,
                    community_id,
                    wait_for_response=wait_for_response,
                    try_database=try_database,
                )
            )
        except Exception as exc:
            logger.debug(f"fetch_community failed while checking update for {community_id} " f"(try_database={try_database}): {exc}")

    member_communities = backend.wakuext_service.communities()
    sources.append(next((c for c in messenger.communities_list(member_communities) if c.get("id") == community_id), None))

    return any(community_has_expected_details(source, expected_name, expected_description) for source in sources)


def wait_until_join_permissions_satisfied(
    backend: StatusBackend,
    community_id: str,
    wallet_address: str,
    attempts: int = 10,
    delay: int = 2,
    refresh_community: bool = True,
) -> dict:
    """Poll until token-gated join permissions are satisfied for the wallet."""
    backend.wallet_service.fetch_or_get_cached_wallet_balances([wallet_address], True)
    permissions_resp = None
    for _ in range(attempts):
        if refresh_community:
            messenger.fetch_community(backend, community_id)
        permissions_resp = backend.wakuext_service.check_permissions_to_join_community(community_id)
        if permissions_resp and permissions_resp.get("satisfied"):
            return permissions_resp
        backend.wallet_service.fetch_or_get_cached_wallet_balances([wallet_address], True)
        time.sleep(delay)

    assert permissions_resp, "Failed to check permissions to join community"
    assert permissions_resp.get("satisfied"), "Permissions to join are not satisfied"
    return permissions_resp


def wait_until_member_sees_community_update(
    backend: StatusBackend,
    community_id: str,
    expected_name: str,
    expected_description: str,
    attempts: int = 30,
    delay: int = 2,
    spectate: bool = False,
    fetch_live: bool = True,
):
    """Poll until backend sees the expected community name and description.

    fetch_live=False reads only the local DB (non-blocking) — the update lands there once the backend
    processes the message, so it avoids a live fetch that can hang under store contention.
    """
    last_community = None
    for _ in range(attempts):
        # Re-spectate each iteration to survive a subscription that raced a device reconnect.
        if spectate:
            try:
                backend.wakuext_service.spectate_community(community_id)
            except Exception as exc:
                logger.debug(f"spectate_community failed for {community_id}: {exc}")

        if check_member_community_updated(backend, community_id, expected_name, expected_description, include_live_fetch=fetch_live):
            return
        member_communities = backend.wakuext_service.communities()
        last_community = next(
            (c for c in messenger.communities_list(member_communities) if c.get("id") == community_id),
            None,
        )
        time.sleep(delay)

    assert check_member_community_updated(backend, community_id, expected_name, expected_description, include_live_fetch=fetch_live), (
        f"{backend} did not see the updated community name/description; "
        f"expected name={expected_name!r} description={expected_description!r}, "
        f"last observed name={(last_community or {}).get('name')!r} "
        f"description={(last_community or {}).get('description')!r}"
    )


def edit_community_and_wait_until_observer_sees_update(
    editor_backend,
    observer_backend,
    community_id: str,
    attempts: int = 30,
    spectate_before_wait: bool = False,
    wait_for_message_signal: bool = True,
):
    """Edit a community and wait until *observer_backend* sees the new name/description."""
    edit_result: dict[str, str] = {}

    def _edit():
        edit_result["name"], edit_result["description"] = messenger.edit_community(editor_backend, community_id)

    if wait_for_message_signal:
        try:
            with observer_backend.expect_signal(
                SignalType.MESSAGES_NEW,
                predicate=lambda signal: community_id in json.dumps(signal),
                timeout=60,
            ):
                _edit()
        except TimeoutError:
            logger.warning(f"MESSAGES_NEW not received for community {community_id}; polling for update")
            if "name" not in edit_result:
                _edit()
    else:
        _edit()

    wait_until_member_sees_community_update(
        observer_backend,
        community_id,
        edit_result["name"],
        edit_result["description"],
        attempts=attempts,
        spectate=spectate_before_wait,
    )
    return edit_result["name"], edit_result["description"]


def join_token_gated_community_as_member(
    owner_backend,
    member_backend,
    community_id: str,
    wallet_address: str,
    attempts: int = 10,
):
    """Wait for join permissions and accept the member into a token-gated community."""
    wait_until_join_permissions_satisfied(member_backend, community_id, wallet_address, attempts=attempts)
    join_community_with_signatures_and_accept(owner_backend, member_backend, community_id, wallet_address)


def create_member_b_profile(
    backend_new_profile,
    snt_token_overrides,
    multicall3_deployer,
    community_token_deployer: str,
    name: str = "member_b",
):
    """Create a second member profile with the same local test-network overrides as owner/member."""
    return backend_new_profile(
        name=name,
        token_overrides=snt_token_overrides,
        multicall_contract_address=multicall3_deployer.contract_address,
        community_token_deployer_contract_address=community_token_deployer,
    )


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
        # Values must match protobuf CommunityTokenPermission.Type in protocol/protobuf/communities.proto
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
    spectate: bool = False,
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
        if spectate:
            try:
                backend.wakuext_service.spectate_community(community_id)
            except Exception as exc:
                logger.debug(f"spectate_community failed while waiting for role {role}: {exc}")
        if fetch_from_store:
            messenger.fetch_community(backend, community_id, try_database=False)
        communities = backend.wakuext_service.communities()
        community = next(
            (c for c in messenger.communities_list(communities) if c.get("id") == community_id),
            None,
        )
        roles = (community or {}).get("members", {}).get(member_key, {}).get("roles", [])
        if role in roles:
            assert community is not None, "Community not found"
            return community
        time.sleep(delay)

    if required:
        assert community is not None, "Community not found"
        roles = community.get("members", {}).get(member_key, {}).get("roles", [])
        assert role in roles, f"Member {member_key} did not receive role {role} after {attempts} attempts; roles={roles}"
        return community
    else:
        logger.warning(f"Backend local state may not yet reflect role {role} for {member_key}; proceeding")
    return community


def _wait_for_community_member_reevaluation_done(
    backend: StatusBackend,
    community_id: str,
    timeout: float = 60,
):
    """Wait until member permission reevaluation completes for *community_id*."""

    def _predicate(signal: dict) -> bool:
        event = signal.get("event") or {}
        if event.get("communityId") != community_id:
            return False
        return event.get("status") == CommunityMemberReevaluationStatus.DONE

    with backend.expect_signal(
        SignalType.COMMUNITY_MEMBER_REEVALUATION_STATUS,
        predicate=_predicate,
        timeout=timeout,
    ):
        pass


def sync_community_member_permissions(
    owner_backend: StatusBackend,
    community_id: str,
    *,
    timeout: float = 60,
) -> None:
    """Force permission reevaluation on the control node and wait until it completes."""
    owner_backend.wakuext_service.reevaluate_community_members_permissions(community_id, force=True)
    _wait_for_community_member_reevaluation_done(owner_backend, community_id, timeout=timeout)


def wait_until_member_sees_permissions(
    backend: StatusBackend,
    community_id: str,
    *permission_types: CommunityTokenPermissionType,
    attempts: int = 15,
    delay: int = 2,
) -> dict:
    """Poll until *backend* sees the community with the given token permission types."""
    expected = {permission_type.value for permission_type in permission_types}
    return wait_until_community_has_permission_types(
        backend,
        community_id,
        expected,
        attempts=attempts,
        delay=delay,
        fetch=True,
        spectate=True,
    )


def wait_until_community_has_permission_types(
    backend: StatusBackend,
    community_id: str,
    expected_types: set[int],
    attempts: int = 15,
    delay: int = 2,
    *,
    fetch: bool = False,
    spectate: bool = False,
) -> dict:
    """Poll until the community on *backend* includes all expected token permission types."""
    community = None
    for _ in range(attempts):
        if spectate:
            try:
                backend.wakuext_service.spectate_community(community_id)
            except Exception as exc:
                logger.debug(f"spectate_community failed for {community_id}: {exc}")
        if fetch:
            try:
                community = messenger.fetch_community(backend, community_id)
            except Exception as exc:
                logger.debug(f"fetch_community failed for {community_id}: {exc}")
        if not community:
            community = next(
                (c for c in messenger.communities_list(backend.wakuext_service.communities()) if c.get("id") == community_id),
                None,
            )
        if community and expected_types.issubset(community_permission_types(community)):
            return community
        community = None
        time.sleep(delay)

    assert community, f"Community {community_id} not found on {backend}"
    observed = community_permission_types(community)
    assert expected_types.issubset(observed), f"Expected permission types {expected_types}, got {observed}"
    return community
