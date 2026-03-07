# pyright: reportOptionalMemberAccess=false
# pyright: reportAttributeAccessIssue=false
import logging
import time
from contextlib import contextmanager
from uuid import uuid4

import pytest
from tenacity import retry, stop_after_delay, wait_fixed

from clients.signals import SignalType
from resources.enums import MessageContentType
from utils import fake


# --- Message parsing utilities ---


def get_message_by_content_type(response, content_type, message_pattern=""):
    matched_messages = []
    messages = response.get("messages", [])
    for message in messages:
        if message.get("contentType") != content_type:
            continue
        if not message_pattern or message_pattern in str(message):
            matched_messages.append(message)
    if matched_messages:
        return matched_messages
    else:
        raise ValueError(f"Failed to find a message with contentType '{content_type}' and message_pattern: `{message_pattern}` in response")


def get_message_id(response, index=0):
    return response.get("messages", [])[index].get("id", "")


def get_message_by_message_id(response, message_id: str):
    messages = response.get("messages", [])
    matched_message = None
    for message in messages:
        if message.get("id", "") == message_id:
            matched_message = message
            break
    if matched_message is None:
        raise ValueError(f"Failed to find a message with message id '{message_id}' in response")
    return matched_message


def validate_signal_event_against_response(signal_event, fields_to_validate, expected_message):
    expected_message_id = expected_message.get("id")
    signal_event_messages = signal_event.get("event", {}).get("messages")
    assert len(signal_event_messages) > 0, "No messages found in the signal event"

    message = next(
        (message for message in signal_event_messages if message.get("id") == expected_message_id),
        None,
    )
    assert message, f"Message with ID {expected_message_id} not found in the signal event"

    message_mismatch = []
    for response_field, event_field in fields_to_validate.items():
        response_value = expected_message[response_field]
        event_value = message[event_field]
        if response_value != event_value:
            message_mismatch.append(f"Field '{response_field}': Expected '{response_value}', Found '{event_value}'")

    if not message_mismatch:
        return

    raise AssertionError(
        "Some Sender RPC responses are not matching the signals received by the receiver.\n" "Details of mismatches:\n" + "\n".join(message_mismatch)
    )


# --- Contact operations ---


def send_contact_request_and_wait_for_signal_to_be_received(sender, receiver) -> str:
    response = sender.wakuext_service.send_contact_request(receiver.public_key, "contact_request")
    expected_message = get_message_by_content_type(response, content_type=MessageContentType.CONTACT_REQUEST.value)[0]
    message_id = expected_message.get("id")

    # `message_id` becomes known only after the RPC response, so this is a post-hoc waiter.
    # Use `start="beginning"` to avoid missing the signal if it arrived during the RPC.
    with receiver.expect_signal(
        SignalType.MESSAGES_NEW,
        pattern=message_id,
        timeout=60,
        start="beginning",
    ) as exp:
        pass

    signal = exp.result
    assert message_id in str(signal), f"Message ID {message_id} not found in signal"
    return message_id


def accept_contact_request_and_wait_for_signal_to_be_received(message_id, sender, receiver):
    accepted_signal = f"@{receiver.public_key} accepted your contact request"
    with sender.expect_signal(SignalType.MESSAGES_NEW, pattern=accepted_signal, timeout=60):
        receiver.wakuext_service.accept_contact_request(message_id, sender.public_key)


def make_contacts(sender, receiver) -> str:
    existing_contacts = receiver.wakuext_service.get_contacts()

    if sender.public_key in str(existing_contacts):
        return  # type: ignore

    message_id = send_contact_request_and_wait_for_signal_to_be_received(sender, receiver)
    accept_contact_request_and_wait_for_signal_to_be_received(message_id, sender, receiver)
    return message_id


def add_contact(sender=None, receiver=None, execution_number=None, network_condition=None):
    message_text = f"test_contact_request_{execution_number}_{uuid4()}"
    existing_contacts = receiver.wakuext_service.get_contacts()

    if sender.public_key in str(existing_contacts):
        pytest.skip("Contact request was already sent for this sender<->receiver. Skipping test!!")

    if network_condition:
        network_condition(receiver)

    response = sender.wakuext_service.send_contact_request(receiver.public_key, message_text)
    expected_message = get_message_by_content_type(response, content_type=MessageContentType.CONTACT_REQUEST.value)[0]

    with receiver.expect_signal(SignalType.MESSAGES_NEW, pattern=expected_message.get("id"), timeout=60) as exp:
        pass
    messages_new_event = exp.result

    signal_messages_texts = []
    if "messages" in messages_new_event.get("event", {}):
        signal_messages_texts.extend(message["text"] for message in messages_new_event["event"]["messages"] if "text" in message)

    assert (
        f"@{sender.public_key} sent you a contact request" in signal_messages_texts
    ), "Couldn't find the signal corresponding to the contact request"

    validate_signal_event_against_response(
        signal_event=messages_new_event,
        fields_to_validate={"text": "text"},
        expected_message=expected_message,
    )


# --- Private group operations ---


def join_private_group(admin=None, member=None) -> str:

    private_group_name = f"private_group_{uuid4()}"
    response = admin.wakuext_service.create_group_chat_with_members([member.public_key], private_group_name)
    expected_group_creation_msg = f"@{admin.public_key} created the group {private_group_name}"
    expected_message = get_message_by_content_type(
        response,
        content_type=MessageContentType.SYSTEM_MESSAGE_CONTENT_PRIVATE_GROUP.value,
        message_pattern=expected_group_creation_msg,
    )[0]
    with member.expect_signal(SignalType.MESSAGES_NEW, pattern=expected_message.get("id"), timeout=60):
        pass
    return response.get("chats", [])[0].get("id")


def create_private_group(private_groups_count, admin, member):
    private_groups = []
    for i in range(private_groups_count):
        private_group_name = f"private_group_{i+1}_{uuid4()}"
        response = admin.wakuext_service.create_group_chat_with_members([member.public_key], private_group_name)

        expected_group_creation_msg = f"@{admin.public_key} created the group {private_group_name}"
        expected_message = get_message_by_content_type(
            response,
            content_type=MessageContentType.SYSTEM_MESSAGE_CONTENT_PRIVATE_GROUP.value,
            message_pattern=expected_group_creation_msg,
        )[0]

        private_groups.append(expected_message)
        time.sleep(0.01)

    for i, expected_message in enumerate(private_groups):
        with member.expect_signal(SignalType.MESSAGES_NEW, pattern=expected_message.get("id"), timeout=60) as exp:
            pass
        messages_new_event = exp.result
        validate_signal_event_against_response(
            signal_event=messages_new_event,
            expected_message=expected_message,
            fields_to_validate={"text": "text"},
        )


def private_group_message(message_count, private_group_id, sender=None, receiver=None):
    sent_messages = []
    for i in range(message_count):
        message_text = f"test_message_{i+1}_{uuid4()}"
        response = sender.wakuext_service.send_group_chat_message(private_group_id, message_text)
        expected_message = get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
        sent_messages.append(expected_message)
        time.sleep(0.01)

    for _, expected_message in enumerate(sent_messages):
        with receiver.expect_signal(SignalType.MESSAGES_NEW, pattern=expected_message.get("id"), timeout=60) as exp:
            pass
        messages_new_event = exp.result
        validate_signal_event_against_response(
            signal_event=messages_new_event,
            fields_to_validate={"text": "text"},
            expected_message=expected_message,
        )


# --- Community operations ---


def create_community(node) -> str:
    response = node.wakuext_service.create_community(fake.community_name(), fake.community_description())
    community_id = response.get("communities", [{}])[0].get("id")
    return community_id


def fetch_community(node, community_id):
    return node.wakuext_service.fetch_community(community_id)


def join_community(member, admin, community_id):
    # Ensure both nodes are aware of the community before we proceed.
    # (Some RPCs depend on local DB state and can lag if the community wasn't fetched yet.)
    fetch_community(member, community_id)
    fetch_community(admin, community_id)

    # Capture start index to make any post-hoc signal waits race-safe.
    member_messages_start = len(member.received_signals[SignalType.MESSAGES_NEW])

    response_to_join = member.wakuext_service.request_to_join_community(community_id)
    join_req = (response_to_join.get("requestsToJoinCommunity") or [{}])[0]
    join_id = join_req.get("id")
    join_state = join_req.get("state")
    assert join_id, f"Failed to request to join community: {response_to_join}"

    def _pick_chat_id(chats: dict) -> str | None:
        if not chats:
            return None
        try:
            # Prefer the default/general chat (lowest position).
            # Different RPCs return `chats` in slightly different shapes:
            # - {chatId: {...}} (where value may or may not include "id")
            # - {chatId: {"id": chatId, ...}}
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
            # Fallback: take any key.
            return next(iter(chats.keys()), None)

    # Communities can be created with different membership rules.
    # We decide the flow by the request state returned from `request_to_join_community`:
    # - Accepted => auto-accept flow (no admin action required)
    # - Pending  => manual-accept flow (admin must accept)
    deadline = time.monotonic() + 240

    last_pending = None
    last_latest_admin = None
    last_member_comm = None
    last_member_latest = None
    last_accept_error = None
    last_chat_messages_error = None
    last_accept_attempt_at = 0.0
    accepted_seen = False
    accepted_via = None

    def _is_accepted(state) -> bool:
        return state == 3 or str(state) == "3"

    while time.monotonic() < deadline:
        # 1) Observe request state on the member side (it can transition from Pending -> Accepted).
        # This is the most reliable driver for the join flow.
        if not accepted_seen:
            if join_state is not None:
                # `join_state` comes from the immediate RPC response.
                # We consider it authoritative, but still allow later confirmation.
                if _is_accepted(join_state):
                    accepted_seen = True
                    accepted_via = "rpc_response"
            try:
                last_member_latest = member.wakuext_service.latest_request_to_join_for_community(community_id)
                if last_member_latest and last_member_latest.get("id") == join_id:
                    if _is_accepted(last_member_latest.get("state")):
                        accepted_seen = True
                        accepted_via = "member_latest"
            except Exception:
                pass

        if not accepted_seen and not _is_accepted(join_state):
            try:
                last_pending = admin.wakuext_service.pending_requests_to_join_for_community(community_id) or []
            except Exception:
                pass

            try:
                last_latest_admin = admin.wakuext_service.latest_request_to_join_for_community(community_id)
            except Exception:
                pass

            admin_observed_ids: set[str] = set()
            if last_pending:
                admin_observed_ids.update([r.get("id") for r in last_pending if r.get("id")])
            if last_latest_admin and last_latest_admin.get("id"):
                admin_observed_ids.add(last_latest_admin.get("id"))

            try:
                all_non_approved = admin.wakuext_service.all_non_approved_communities_requests_to_join() or []
                for r in all_non_approved:
                    if r.get("communityId") == community_id and r.get("id"):
                        admin_observed_ids.add(r.get("id"))
            except Exception:
                pass

            # If admin sees our join_id, we can accept it.
            # If join_id is not visible but admin sees exactly one request for this community,
            # accept that one (best-effort for eventual-consistency glitches).
            accept_id: str | None = None
            if join_id and join_id in admin_observed_ids:
                accept_id = join_id
            elif len(admin_observed_ids) == 1:
                accept_id = next(iter(admin_observed_ids))

            if accept_id and (time.monotonic() - last_accept_attempt_at) >= 2.0:
                try:
                    accept_resp = admin.wakuext_service.accept_request_to_join_community(accept_id)
                    if accept_resp and _is_accepted((accept_resp.get("requestsToJoinCommunity") or [{}])[0].get("state")):
                        accepted_seen = True
                        accepted_via = "admin_accept"
                        join_state = 3
                        if accept_id != join_id:
                            join_id = accept_id
                except Exception as e:
                    last_accept_error = str(e)
                finally:
                    last_accept_attempt_at = time.monotonic()

        # 3) Once accepted, wait for member to observe acceptance via signal (helps with propagation).
        if accepted_seen and accepted_via != "member_signal":
            try:
                with member.expect_signal(
                    SignalType.MESSAGES_NEW,
                    accept_fn=lambda s: any(
                        r.get("id") == join_id and r.get("state") == 3 for r in (s.get("event", {}).get("requestsToJoinCommunity") or [])
                    ),
                    start=member_messages_start,
                    timeout=5,
                ):
                    pass
                accepted_via = "member_signal"
            except Exception:
                # Not all setups emit the signal immediately; don't fail here.
                pass

        # 4) Completion: member reports joined/isMember OR chat becomes accessible.
        try:
            last_member_comm = member.wakuext_service.fetch_community(community_id)
            is_joined = last_member_comm and (last_member_comm.get("joined") is True or last_member_comm.get("isMember") is True)

            chats = (last_member_comm.get("chats") or {}) if last_member_comm else {}
            chat_id = _pick_chat_id(chats)
            if chat_id:
                if is_joined:
                    return community_id + chat_id

                # Only use chat access as a proof AFTER we've seen the request accepted.
                if accepted_seen:
                    try:
                        resp = member.wakuext_service.chat_messages(community_id + chat_id, limit=1)
                        if resp is not None and "messages" in resp:
                            return community_id + chat_id
                    except Exception as e:
                        last_chat_messages_error = str(e)
        except Exception:
            pass

        time.sleep(0.5)

    raise Exception(
        "Failed to join community within timeout. "
        f"community_id={community_id}, join_id={join_id}, join_state={join_state}, "
        f"accepted_seen={accepted_seen}, accepted_via={accepted_via}, "
        f"last_pending={last_pending}, "
        f"last_latest_admin={last_latest_admin}, "
        f"last_member_joined={getattr(last_member_comm, 'get', lambda *_: None)('joined') if last_member_comm else None}, "
        f"last_member_is_member={getattr(last_member_comm, 'get', lambda *_: None)('isMember') if last_member_comm else None}, "
        f"last_member_chats_keys={list((last_member_comm or {}).get('chats', {}).keys()) if last_member_comm else None}, "
        f"last_member_latest={last_member_latest}, "
        f"last_accept_error={last_accept_error}, "
        f"last_chat_messages_error={last_chat_messages_error}"
    )


@retry(stop=stop_after_delay(20), wait=wait_fixed(0.5), reraise=True)
def leave_the_community(node, community_id):
    response = node.wakuext_service.leave_community(community_id)
    target_community = [existing_community for existing_community in response.get("communities") if existing_community.get("id") == community_id][0]
    assert target_community.get("joined") is False


@retry(stop=stop_after_delay(20), wait=wait_fixed(2), reraise=True)
def check_node_joined_community(node, joined, community_id):
    response = fetch_community(node, community_id)
    assert response.get("joined") is joined


# --- One-to-one message operations ---


def send_multiple_one_to_one_messages(message_count=1, sender=None, receiver=None) -> tuple[list[str], list[dict]]:
    sent_texts = []
    responses = []

    for i in range(message_count):
        message_text = f"test_message_{i}_{uuid4()}"
        sent_texts.append(message_text)
        response = sender.wakuext_service.send_one_to_one_message(receiver.public_key, message_text)
        responses.append(response)

    return sent_texts, responses


def one_to_one_message(message_count, sender=None, receiver=None):
    _, responses = send_multiple_one_to_one_messages(message_count, sender=sender, receiver=receiver)
    messages = list(map(lambda r: r.get("messages", [])[0], responses))

    for expected_message in messages:
        with receiver.expect_signal(SignalType.MESSAGES_NEW, pattern=expected_message.get("id"), timeout=60) as exp:
            pass
        messages_new_event = exp.result
        validate_signal_event_against_response(
            signal_event=messages_new_event,
            fields_to_validate={"text": "text"},
            expected_message=expected_message,
        )

    return responses


def community_messages(message_chat_id, message_count, sender=None, receiver=None):
    sent_messages = []
    for i in range(message_count):
        message_text = f"test_message_{i+1}_{uuid4()}"
        response = sender.wakuext_service.send_chat_message(message_chat_id, message_text)
        expected_message = get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
        sent_messages.append(expected_message)
        time.sleep(0.01)

    for i, expected_message in enumerate(sent_messages):
        with receiver.expect_signal(SignalType.MESSAGES_NEW, pattern=expected_message.get("id"), timeout=60) as exp:
            pass
        messages_new_event = exp.result
        validate_signal_event_against_response(
            signal_event=messages_new_event,
            fields_to_validate={"text": "text"},
            expected_message=expected_message,
        )


# --- Network conditions ---


@contextmanager
def add_latency(node, latency=300, jitter=50):
    logging.info("Entering context manager: add_latency")
    node.container_exec(
        f"apt-get update && apt-get install -y iproute2 && tc qdisc add dev eth0 root netem delay {latency}ms {jitter}ms distribution normal"
    )
    try:
        yield
    finally:
        logging.info("Exiting context manager: add_latency")
        node.container_exec("tc qdisc del dev eth0 root")


@contextmanager
def add_packet_loss(node, packet_loss=2):
    logging.info("Entering context manager: add_packet_loss")
    node.container_exec(f"apt-get update && apt-get install -y iproute2 && tc qdisc add dev eth0 root netem loss {packet_loss}%")
    try:
        yield
    finally:
        logging.info("Exiting context manager: add_packet_loss")
        node.container_exec("tc qdisc del dev eth0 root netem")


@contextmanager
def add_low_bandwith(node, rate="1mbit", burst="32kbit", limit="12500"):
    logging.info("Entering context manager: add_low_bandwith")
    node.container_exec(f"apt-get update && apt-get install -y iproute2 && tc qdisc add dev eth0 root tbf rate {rate} burst {burst} limit {limit}")
    try:
        yield
    finally:
        logging.info("Exiting context manager: add_low_bandwith")
        node.container_exec("tc qdisc del dev eth0 root")


@contextmanager
def node_pause(node):
    logging.info("Entering context manager: node_pause")
    node.container_pause()
    try:
        yield
    finally:
        logging.info("Exiting context manager: node_pause")
        node.container_unpause()
