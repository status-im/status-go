import asyncio
import logging
from uuid import uuid4

from clients.signals import SignalType
from resources.enums import MessageContentType
from utils import fake

from steps.messenger import (  # noqa: F401
    get_message_by_content_type,
    get_message_id,
    get_message_by_message_id,
    validate_signal_event_against_response,
    send_multiple_one_to_one_messages,
)


async def send_contact_request_and_wait(sender, receiver, message_text: str = "contact_request") -> str:
    response = sender.wakuext_service.send_contact_request(receiver.public_key, message_text)
    expected_message = get_message_by_content_type(response, MessageContentType.CONTACT_REQUEST.value)[0]
    message_id = expected_message.get("id")
    assert message_id, "Message ID should not be empty"

    await receiver.wait_for_signal(SignalType.MESSAGES_NEW, pattern=message_id, timeout=60, check_buffer=True)
    return message_id


async def accept_contact_request_and_wait(message_id: str, sender, receiver) -> None:
    accepted_signal = f"@{receiver.public_key} accepted your contact request"
    receiver.wakuext_service.accept_contact_request(message_id, sender.public_key)
    await sender.wait_for_signal(SignalType.MESSAGES_NEW, pattern=accepted_signal, timeout=60, check_buffer=True)


async def make_contacts(sender, receiver) -> str:
    existing_contacts = receiver.wakuext_service.get_contacts()
    if sender.public_key in str(existing_contacts):
        return ""

    message_id = await send_contact_request_and_wait(sender, receiver)
    await accept_contact_request_and_wait(message_id, sender, receiver)
    return message_id


def create_community(node) -> str:
    wakuext = getattr(node, "wakuext_service", None)
    if wakuext is None and hasattr(node, "backend"):
        wakuext = node.backend.wakuext_service
    assert wakuext is not None, "Node must have wakuext_service attribute"
    response = wakuext.create_community(fake.community_name(), fake.community_description())
    community_id = response.get("communities", [{}])[0].get("id")
    return community_id


def fetch_community(node, community_id: str) -> dict:
    wakuext = getattr(node, "wakuext_service", None)
    if wakuext is None and hasattr(node, "backend"):
        wakuext = node.backend.wakuext_service
    if wakuext is None:
        raise ValueError("Node must have wakuext_service attribute")
    return wakuext.fetch_community(community_id)


async def join_private_group(admin, member) -> str:
    private_group_name = f"private_group_{uuid4()}"
    response = admin.wakuext_service.create_group_chat_with_members([member.public_key], private_group_name)
    expected_group_creation_msg = f"@{admin.public_key} created the group {private_group_name}"
    expected_message = get_message_by_content_type(
        response,
        content_type=MessageContentType.SYSTEM_MESSAGE_CONTENT_PRIVATE_GROUP.value,
        message_pattern=expected_group_creation_msg,
    )[0]
    await member.wait_for_signal(SignalType.MESSAGES_NEW, pattern=expected_message.get("id"), timeout=60, check_buffer=True)
    return response.get("chats", [])[0].get("id")


async def _async_retry_call(func, *args, max_attempts: int = 40, retry_interval: float = 0.5):
    for attempt in range(max_attempts):
        try:
            response = func(*args)
            if response is not None:
                return response
        except Exception as e:
            logging.debug(f"Attempt {attempt + 1}/{max_attempts} for {func.__name__} failed: {e}")
        await asyncio.sleep(retry_interval)
    raise Exception(f"Failed to execute {func.__name__} within {max_attempts * retry_interval}s")


def _pick_chat_id(chats: dict) -> str | None:
    if not chats:
        return None
    try:
        items = list(chats.items())
        items.sort(key=lambda kv: (kv[1] or {}).get("position", 1_000_000))
        for key, value in items:
            if isinstance(value, dict):
                chat_id = value.get("id") or key
            else:
                chat_id = key
            if chat_id:
                return chat_id
        return items[0][0] if items else None
    except Exception:
        return next(iter(chats.keys()), None)


async def join_community(member, admin, community_id: str) -> str:
    # Fetch community with retry for light client (may need time to sync)
    community = None
    for attempt in range(12):
        community = fetch_community(member, community_id)
        if community and community.get("chats"):
            break
        logging.debug(f"Community {community_id} not ready on member (attempt {attempt + 1}/12)")
        await asyncio.sleep(5)

    if not community:
        raise Exception(f"Community {community_id} not visible to member after retries")

    # Ensure admin has community data (triggers filter subscriptions for community topics)
    fetch_community(admin, community_id)
    # In light client mode, filter subscriptions are async and take several seconds.
    # If the member sends the join request before the admin subscribes,
    # the message is lost (filter protocol doesn't deliver historical messages).
    if getattr(admin, "waku_light_client", False):
        await asyncio.sleep(10)

    # Request to join
    response_to_join = member.wakuext_service.request_to_join_community(community_id)
    join_req = (response_to_join.get("requestsToJoinCommunity") or [{}])[0]
    join_id = join_req.get("id")
    join_state = join_req.get("state")
    assert join_id, f"Failed to request to join community: {response_to_join}"

    def _is_accepted(state) -> bool:
        return state == 3 or str(state) == "3"

    # Auto-accept: if state is already accepted, skip admin accept phase
    if _is_accepted(join_state):
        community = fetch_community(member, community_id)
        chats = community.get("chats", {}) if community else {}
        chat_id = _pick_chat_id(chats)
        if chat_id:
            return community_id + chat_id

    # Phase 1: Wait for admin to see the pending request.
    accept_id = None
    last_pending = None
    last_latest = None
    for attempt in range(120):
        admin_observed_ids: set[str] = set()
        try:
            last_pending = admin.wakuext_service.pending_requests_to_join_for_community(community_id) or []
            for req in last_pending:
                if req.get("id"):
                    admin_observed_ids.add(req["id"])
        except Exception as e:
            logging.debug(f"Attempt {attempt + 1}/120 pending_requests check failed: {e}")

        try:
            last_latest = admin.wakuext_service.latest_request_to_join_for_community(community_id)
            if last_latest and last_latest.get("id"):
                admin_observed_ids.add(last_latest["id"])
        except Exception as e:
            logging.debug(f"Attempt {attempt + 1}/120 latest_request check failed: {e}")

        try:
            all_non_approved = admin.wakuext_service.all_non_approved_communities_requests_to_join() or []
            for r in all_non_approved:
                if r.get("communityId") == community_id and r.get("id"):
                    admin_observed_ids.add(r["id"])
        except Exception as e:
            logging.debug(f"Attempt {attempt + 1}/120 all_non_approved check failed: {e}")

        if join_id in admin_observed_ids:
            accept_id = join_id
            break
        elif len(admin_observed_ids) == 1:
            accept_id = next(iter(admin_observed_ids))
            break

        # Re-send the join request periodically in case the original was lost
        if attempt > 0 and attempt % 30 == 0:
            logging.debug(f"Re-sending join request (attempt {attempt}/120)")
            try:
                retry_resp = member.wakuext_service.request_to_join_community(community_id)
                retry_req = (retry_resp.get("requestsToJoinCommunity") or [{}])[0]
                if retry_req.get("id"):
                    join_id = retry_req["id"]
            except Exception as e:
                logging.debug(f"Re-send join request failed: {e}")

        await asyncio.sleep(1)

    if not accept_id:
        raise Exception(
            f"Admin never saw pending request for community {community_id} "
            f"(join_id={join_id}) after 120s. "
            f"last_pending={last_pending}, last_latest={last_latest}"
        )

    # Phase 2: Accept the request
    response = await _async_retry_call(
        admin.wakuext_service.accept_request_to_join_community,
        accept_id,
        max_attempts=20,
        retry_interval=1.0,
    )

    accept_state = (response.get("requestsToJoinCommunity") or [{}])[0].get("state")
    if not _is_accepted(accept_state):
        raise Exception(f"Accept request failed. State: {accept_state}, Response: {response}")

    chats = response.get("communities", [{}])[0].get("chats", {})
    chat_id = _pick_chat_id(chats)
    if not chat_id:
        raise Exception(f"No valid chat found in community response: {response}")
    return community_id + chat_id
