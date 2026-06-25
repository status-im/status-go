"""Community ownership-transfer steps (issue 7131): transfer owner NFT, set signer, promote."""

import logging
import time
import uuid
from typing import Optional

from web3 import Web3

from clients.api import ApiResponseError
from clients.services.wakuext import (
    CommunityRoles,
    CommunityTokenPermissionType,
    CommunityTokenPrivilegesLevel,
    CommunityTokenType,
)
from clients.services.wallet_send_type import WalletSendType
from clients.signals import SignalType
from clients.status_backend import StatusBackend
from steps import community_token_deploy, community_tokens, messenger
from steps.community_tokens import NATIVE_TOKEN_ADDRESS

logger = logging.getLogger(__name__)

# Go iota enums (protocol/communities/token), distinct from the wallet-router enums.
TOKEN_OWNER_PRIVILEGES_LEVEL = 0  # token.OwnerLevel
TOKEN_DEPLOY_STATE_DEPLOYED = 2  # token.Deployed
PROTOBUF_COMMUNITY_TOKEN_ERC721 = 2  # protobuf.CommunityTokenType_ERC721

ERC721_OWNERSHIP_ABI = [
    {
        "inputs": [{"name": "owner", "type": "address"}],
        "name": "balanceOf",
        "outputs": [{"name": "", "type": "uint256"}],
        "stateMutability": "view",
        "type": "function",
    },
    {
        "inputs": [{"name": "owner", "type": "address"}, {"name": "index", "type": "uint256"}],
        "name": "tokenOfOwnerByIndex",
        "outputs": [{"name": "", "type": "uint256"}],
        "stateMutability": "view",
        "type": "function",
    },
    {
        "inputs": [{"name": "tokenId", "type": "uint256"}],
        "name": "ownerOf",
        "outputs": [{"name": "", "type": "address"}],
        "stateMutability": "view",
        "type": "function",
    },
    {
        "inputs": [{"name": "from", "type": "address"}, {"name": "to", "type": "address"}, {"name": "tokenId", "type": "uint256"}],
        "name": "safeTransferFrom",
        "outputs": [],
        "stateMutability": "nonpayable",
        "type": "function",
    },
]


def _anvil_rpc(anvil_client, method: str, params: list):
    from web3.types import RPCEndpoint

    return anvil_client.provider.make_request(RPCEndpoint(method), params)


def resolve_owner_token_id(anvil_client, owner_token_address: str, holder_address: str) -> int:
    contract = anvil_client.eth.contract(address=Web3.to_checksum_address(owner_token_address), abi=ERC721_OWNERSHIP_ABI)
    try:
        return contract.functions.tokenOfOwnerByIndex(Web3.to_checksum_address(holder_address), 0).call()
    except Exception as exc:
        logger.debug(f"tokenOfOwnerByIndex unavailable for {owner_token_address}: {exc}; defaulting to 0")
        return 0


def transfer_owner_token_to_new_owner(
    anvil_client,
    owner_token_address: str,
    from_wallet: str,
    to_wallet: str,
    token_id: Optional[int] = None,
) -> int:
    # Test-only: Anvil impersonation, no private key. App equivalent is an ERC721_TRANSFER route.
    from_wallet = Web3.to_checksum_address(from_wallet)
    to_wallet = Web3.to_checksum_address(to_wallet)
    if token_id is None:
        token_id = resolve_owner_token_id(anvil_client, owner_token_address, from_wallet)

    contract = anvil_client.eth.contract(address=Web3.to_checksum_address(owner_token_address), abi=ERC721_OWNERSHIP_ABI)
    _anvil_rpc(anvil_client, "anvil_impersonateAccount", [from_wallet])
    try:
        tx_hash = contract.functions.safeTransferFrom(from_wallet, to_wallet, token_id).transact({"from": from_wallet})
        receipt = anvil_client.eth.wait_for_transaction_receipt(tx_hash, timeout=60)
        assert receipt.get("status") == 1, f"Owner token transfer failed: {receipt}"
    finally:
        _anvil_rpc(anvil_client, "anvil_stopImpersonatingAccount", [from_wallet])

    new_owner = contract.functions.ownerOf(token_id).call()
    assert new_owner.lower() == to_wallet.lower(), f"Owner token still held by {new_owner}, expected {to_wallet}"
    logger.info(f"Transferred owner token {owner_token_address}#{token_id} from {from_wallet} to {to_wallet}")
    return token_id


def register_owner_token_on_backend(
    backend: StatusBackend,
    community_id: str,
    owner_token_address: str,
    *,
    name: str = "TestOwnerToken",
    symbol: str = "TOT",
):
    # Needed by the set-signer route; may already be stored → ignore the UNIQUE constraint.
    try:
        return backend.wakuext_service.save_community_token(
            {
                "tokenType": PROTOBUF_COMMUNITY_TOKEN_ERC721,
                "communityId": community_id,
                "address": owner_token_address,
                "name": name,
                "symbol": symbol,
                "supply": "1",
                "infiniteSupply": False,
                "transferable": True,
                "remoteSelfDestruct": False,
                "chainId": backend.network_id,
                "deployState": TOKEN_DEPLOY_STATE_DEPLOYED,
                "image": "",
                "decimals": 0,
                "privilegesLevel": TOKEN_OWNER_PRIVILEGES_LEVEL,
            }
        )
    except ApiResponseError as exc:
        if "UNIQUE constraint failed" not in str(exc):
            raise
        return None


def publish_owner_token_to_community(owner_backend: StatusBackend, community_id: str, owner_token_address: str):
    # AddCommunityToken adds the BECOME_TOKEN_OWNER permission.
    register_owner_token_on_backend(owner_backend, community_id, owner_token_address)

    owner_backend.wakuext_service.add_community_token(community_id, owner_backend.network_id, owner_token_address)

    owner_community = next(
        (c for c in messenger.communities_list(owner_backend.wakuext_service.communities()) if c.get("id") == community_id),
        None,
    )
    permission_types = community_tokens.community_permission_types(owner_community or {})
    assert (
        CommunityTokenPermissionType.BECOME_TOKEN_OWNER.value in permission_types
    ), f"Owner token did not create a BECOME_TOKEN_OWNER permission; observed={permission_types}"


def set_signer_pubkey(new_owner_backend: StatusBackend, community_id: str, owner_token_address: str) -> dict:
    address_from = messenger.wallet_address(new_owner_backend)
    chain_id = new_owner_backend.network_id
    new_owner_backend.wallet_service.get_balances_at_by_chain([address_from], [f"{chain_id}-{NATIVE_TOKEN_ADDRESS}"])

    # Route validation requires exactly one transfer detail with the owner-token address.
    transfer_details = [
        {
            "tokenType": CommunityTokenType.ERC721.value,
            "privilegeLevel": CommunityTokenPrivilegesLevel.OWNER_LEVEL.value,
            "tokenContractAddress": owner_token_address,
            "amount": hex(1),
        }
    ]

    transaction_uuid = str(uuid.uuid4())
    routes_result = new_owner_backend.wallet_service.suggested_community_routes(
        uuid=transaction_uuid,
        send_type=WalletSendType.COMMUNITY_SET_SIGNER_PUB_KEY,
        chain_id=chain_id,
        address_from=address_from,
        addr_to=owner_token_address,
        community_id=community_id,
        signer_pub_key=new_owner_backend.public_key,
        token_ids=[],
        wallet_addresses=[],
        transfer_details=transfer_details,
    )
    assert routes_result.get("Uuid") == transaction_uuid, f"Expected UUID {transaction_uuid}, got {routes_result.get('Uuid')}"

    with new_owner_backend.expect_signal(
        SignalType.COMMUNITY_TOKEN_TRANSACTION_STATUS_CHANGED,
        predicate=lambda s: community_token_deploy.is_community_token_tx_success(s, WalletSendType.COMMUNITY_SET_SIGNER_PUB_KEY),
        timeout=60,
    ):
        sent_signal, _ = community_token_deploy._sign_and_send_community_route(new_owner_backend, transaction_uuid, address_from)
    return sent_signal


def _permission_types(community: Optional[dict]) -> set:
    """Token-permission type ints present on a fetched community dict (e.g. {5, 6})."""
    return community_tokens.community_permission_types(community or {})


def _community_clock(backend: StatusBackend, community_id: str, try_database: bool = False) -> Optional[int]:
    """Community-level clock *backend* currently sees (live store fetch by default), or None."""
    try:
        community = messenger.fetch_community(backend, community_id, wait_for_response=True, try_database=try_database)
    except ApiResponseError as exc:
        logger.debug(f"_community_clock fetch failed: {exc}")
        return None
    return community.get("clock") if isinstance(community, dict) else None


def wait_until_member_sees_community_clock(
    backend: StatusBackend,
    community_id: str,
    min_clock: int,
    reference_backend: Optional[StatusBackend] = None,
    attempts: int = 45,
    delay: int = 2,
    spectate: bool = True,
) -> dict:
    """Poll a live store fetch until *backend* sees the community at clock >= *min_clock*; return it."""
    community: Optional[dict] = None
    member_clock: Optional[int] = None
    for attempt in range(attempts):
        if spectate:
            try:
                backend.wakuext_service.spectate_community(community_id)
            except ApiResponseError as exc:
                logger.debug(f"spectate_community failed (attempt {attempt + 1}): {exc}")

        try:
            community = messenger.fetch_community(backend, community_id, wait_for_response=True, try_database=False)
        except ApiResponseError as exc:
            logger.debug(f"fetch_community failed (attempt {attempt + 1}): {exc}")
            community = None

        member_clock = community.get("clock") if isinstance(community, dict) else None
        owner_clock = _community_clock(reference_backend, community_id, try_database=True) if reference_backend is not None else None
        logger.info(
            f"community-clock scan {attempt + 1}/{attempts}: member_clock={member_clock} "
            f"owner_clock={owner_clock} (waiting for member_clock >= {min_clock})"
        )
        if isinstance(community, dict) and member_clock is not None and member_clock >= min_clock:
            return community
        time.sleep(delay)

    raise AssertionError(f"Member never reached community clock >= {min_clock} after {attempts} attempts; " f"last member_clock={member_clock}.")


def assert_new_owner_receives_owner_permission(
    owner_backend: StatusBackend,
    new_owner_backend: StatusBackend,
    community_id: str,
    attempts: int = 45,
    delay: int = 2,
) -> None:
    """Assert the owner created the type-6 permission and the new owner receives it once synced."""
    want = CommunityTokenPermissionType.BECOME_TOKEN_OWNER.value

    owner_updated = messenger.fetch_community(owner_backend, community_id, wait_for_response=True, try_database=True)
    assert want in _permission_types(
        owner_updated
    ), f"Owner did not create BECOME_TOKEN_OWNER permission; observed={sorted(_permission_types(owner_updated))}"

    new_owner_updated = wait_until_member_sees_community_clock(
        new_owner_backend,
        community_id,
        min_clock=owner_updated["clock"],
        reference_backend=owner_backend,
        attempts=attempts,
        delay=delay,
    )
    assert want in _permission_types(new_owner_updated), (
        f"New owner caught up to clock {owner_updated['clock']} but has no BECOME_TOKEN_OWNER permission; "
        f"observed={sorted(_permission_types(new_owner_updated))}"
    )


def promote_to_control_node(new_owner_backend: StatusBackend, community_id: str) -> dict:
    response = new_owner_backend.wakuext_service.promote_self_to_control_node(community_id)
    assert response is not None, "promote_self_to_control_node returned no response"
    new_owner_backend.wakuext_service.register_received_ownership_notification(community_id)
    return response


def wait_until_member_is_owner(
    backend: StatusBackend,
    community_id: str,
    new_owner_public_key: str,
    attempts: int = 30,
    delay: int = 2,
    fetch_from_store: bool = False,
    spectate: bool = False,
    required: bool = True,
):
    return community_tokens.wait_for_member_role(
        backend,
        community_id,
        new_owner_public_key,
        CommunityRoles.ROLE_OWNER.value,
        attempts=attempts,
        delay=delay,
        fetch_from_store=fetch_from_store,
        spectate=spectate,
        required=required,
    )


def wait_until_owner_and_members_present(
    backend: StatusBackend,
    community_id: str,
    owner_public_key: str,
    expected_member_public_keys: list,
    attempts: int = 45,
    delay: int = 2,
    spectate: bool = True,
) -> dict:
    """Poll until *backend* sees *owner_public_key* with the OWNER role and every key in
    *expected_member_public_keys* in the member list (the full ticket end state)."""
    community: Optional[dict] = None
    missing: list = list(expected_member_public_keys)
    owner_roles: list = []
    for _ in range(attempts):
        if spectate:
            try:
                backend.wakuext_service.spectate_community(community_id)
            except ApiResponseError as exc:
                logger.debug(f"spectate_community failed: {exc}")
        try:
            messenger.fetch_community(backend, community_id, wait_for_response=True, try_database=False)
        except ApiResponseError as exc:
            logger.debug(f"fetch_community failed: {exc}")

        community = next(
            (c for c in messenger.communities_list(backend.wakuext_service.communities()) if c.get("id") == community_id),
            None,
        )
        members = (community or {}).get("members", {})
        owner_roles = members.get(owner_public_key, {}).get("roles", [])
        missing = [k for k in expected_member_public_keys if k not in members]
        if isinstance(community, dict) and CommunityRoles.ROLE_OWNER.value in owner_roles and not missing:
            return community
        logger.info(
            f"waiting for owner+members: owner_has_role={CommunityRoles.ROLE_OWNER.value in owner_roles}, missing={[k[:12] for k in missing]}"
        )
        time.sleep(delay)

    raise AssertionError(f"Did not reach owner+all-members state after {attempts} attempts; " f"missing members={missing}, owner_roles={owner_roles}")


def wait_until_kicked_from_community(
    backend: StatusBackend,
    community_id: str,
    attempts: int = 45,
    delay: int = 2,
) -> dict:
    """Poll until *backend* observes it has been kicked (no longer joined) after an ownership change.
    The ex-owner must wait for this before re-requesting to join, else it gets 'already joined'."""
    community: Optional[dict] = None
    for _ in range(attempts):
        try:
            messenger.fetch_community(backend, community_id, wait_for_response=True, try_database=False)
        except ApiResponseError as exc:
            logger.debug(f"fetch_community failed: {exc}")
        community = next(
            (c for c in messenger.communities_list(backend.wakuext_service.communities()) if c.get("id") == community_id),
            None,
        )
        if community is not None and not community.get("joined", False):
            return community
        time.sleep(delay)

    raise AssertionError(
        f"Backend was never kicked from community after {attempts} attempts; " f"joined={community.get('joined') if community else None}"
    )


def transfer_community_ownership(
    owner_backend: StatusBackend,
    new_owner_backend: StatusBackend,
    community_id: str,
    owner_token_address: str,
    anvil_client,
) -> dict:
    owner_wallet = messenger.wallet_address(owner_backend)
    new_owner_wallet = messenger.wallet_address(new_owner_backend)

    # Persist the type-6 permission while the old owner is still the verified signer, then flip it.
    assert_new_owner_receives_owner_permission(owner_backend, new_owner_backend, community_id)

    transfer_owner_token_to_new_owner(anvil_client, owner_token_address, owner_wallet, new_owner_wallet)
    register_owner_token_on_backend(new_owner_backend, community_id, owner_token_address)
    set_signer_pubkey(new_owner_backend, community_id, owner_token_address)
    time.sleep(2)  # let the new owner's wallet index the signer change

    return promote_to_control_node(new_owner_backend, community_id)
