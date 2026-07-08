"""Control-node transfer steps (issue 7132): publish the owner token and promote a paired device to control node."""

import logging
import time
from typing import Optional

from clients.api import ApiResponseError
from clients.services.wakuext import CommunityTokenPermissionType
from clients.status_backend import StatusBackend
from steps import community_tokens, messenger

logger = logging.getLogger(__name__)

# Go iota enums (protocol/communities/token).
TOKEN_OWNER_PRIVILEGES_LEVEL = 0
TOKEN_DEPLOY_STATE_DEPLOYED = 2
PROTOBUF_COMMUNITY_TOKEN_ERC721 = 2

# ErrOwnerTokenNeeded: promote is retryable only while the owner-token permission is still syncing.
_PROMOTE_RETRYABLE_ERROR = "owner token is needed"


def register_owner_token_on_backend(
    backend: StatusBackend,
    community_id: str,
    owner_token_address: str,
    *,
    name: str = "TestOwnerToken",
    symbol: str = "TOT",
):
    # May already exist → ignore the UNIQUE constraint.
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
    # Adds the BECOME_TOKEN_OWNER permission that control-node transfer requires.
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


def promote_to_control_node(
    backend: StatusBackend,
    community_id: str,
    attempts: int = 30,
    delay: int = 2,
) -> dict:
    """Promote *backend* to control node, polling while the owner-token permission is still syncing.

    Retries only ErrOwnerTokenNeeded; any other error is a real bug and re-raises immediately."""
    last_exc: Optional[ApiResponseError] = None
    for attempt in range(attempts):
        try:
            response = backend.wakuext_service.promote_self_to_control_node(community_id)
        except ApiResponseError as exc:
            if _PROMOTE_RETRYABLE_ERROR not in str(exc):
                raise
            last_exc = exc
            logger.debug(f"promote_self_to_control_node waiting for owner-token sync (attempt {attempt + 1}/{attempts}): {exc}")
            time.sleep(delay)
            continue
        assert response is not None, "promote_self_to_control_node returned no response"
        backend.wakuext_service.register_received_ownership_notification(community_id)
        return response

    raise AssertionError(f"promote_self_to_control_node never succeeded after {attempts} attempts; last error={last_exc}")


def wait_until_is_control_node(
    backend: StatusBackend,
    community_id: str,
    expected: bool = True,
    attempts: int = 30,
    delay: int = 2,
    try_database: bool = True,
    pull_sync_every: int = 5,
) -> dict:
    """Poll until *backend* sees its own control-node status equal to *expected*.

    isControlNode flips only after the paired-device control-node sync is received, so periodically
    pull historic messages from the store to fetch it instead of waiting for passive delivery."""
    community: Optional[dict] = None
    observed: Optional[bool] = None
    for attempt in range(attempts):
        if pull_sync_every and attempt % pull_sync_every == 0:
            try:
                backend.wakuext_service.request_all_historic_messages()
            except ApiResponseError as exc:
                logger.warning(f"request_all_historic_messages failed (attempt {attempt + 1}): {exc}")
        try:
            community = messenger.fetch_community(backend, community_id, wait_for_response=True, try_database=try_database)
        except ApiResponseError as exc:
            logger.debug(f"fetch_community failed (attempt {attempt + 1}): {exc}")
            community = None
        observed = community.get("isControlNode") if isinstance(community, dict) else None
        logger.info(f"control-node scan {attempt + 1}/{attempts}: isControlNode={observed} (want {expected})")
        if isinstance(community, dict) and observed == expected:
            return community
        time.sleep(delay)

    raise AssertionError(f"Backend never reached isControlNode={expected} after {attempts} attempts; last={observed}")
