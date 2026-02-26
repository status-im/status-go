# pyright: reportOptionalMemberAccess=false
# pyright: reportAttributeAccessIssue=false
import logging
import time
from uuid import uuid4

import pytest
from tenacity import retry, stop_after_delay, wait_fixed

from clients.api import ApiResponseError
from clients.signals import SignalType
from resources.enums import MessageContentType
from steps.network_conditions import NetworkConditionsSteps
from utils import fake

logger = logging.getLogger(__name__)


class MessengerSteps(NetworkConditionsSteps):

    def send_contact_request_and_wait_for_signal_to_be_received(self, sender, receiver) -> str:
        """
        Send a contact request from sender to receiver and wait for confirmation.

        This function sends a contact request through the WakuExt service and waits
        for the receiver to receive the MESSAGES_NEW signal containing the message ID.
        This ensures the request has been delivered before proceeding with other operations.

        Args:
            sender: StatusBackend instance of the sender
            receiver: StatusBackend instance of the receiver

        Returns:
            str: The message ID of the sent contact request

        Raises:
            AssertionError: If the message is not found or signal is not received
        """
        response = sender.wakuext_service.send_contact_request(receiver.public_key, "contact_request")
        expected_message = self.get_message_by_content_type(response, content_type=MessageContentType.CONTACT_REQUEST.value)[0]
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

    def accept_contact_request_and_wait_for_signal_to_be_received(self, message_id, sender, receiver):
        accepted_signal = f"@{receiver.public_key} accepted your contact request"
        with sender.expect_signal(SignalType.MESSAGES_NEW, pattern=accepted_signal, timeout=60):
            receiver.wakuext_service.accept_contact_request(message_id, sender.public_key)

    def make_contacts(self, sender, receiver) -> str:
        """
        Create a contact between sender and receiver.

        This function sends a contact request from sender to receiver and waits for confirmation.
        It also checks if the contact request has been accepted by the receiver.

        Args:
            sender: StatusBackend instance of the sender
            receiver: StatusBackend instance of the receiver

        Returns:
            str: The message ID of the sent contact request
        """
        existing_contacts = receiver.wakuext_service.get_contacts()

        if sender.public_key in str(existing_contacts):
            return  # type: ignore

        message_id = self.send_contact_request_and_wait_for_signal_to_be_received(sender, receiver)
        self.accept_contact_request_and_wait_for_signal_to_be_received(message_id, sender, receiver)
        return message_id

    def validate_signal_event_against_response(self, signal_event, fields_to_validate, expected_message):
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
            "Some Sender RPC responses are not matching the signals received by the receiver.\n"
            "Details of mismatches:\n" + "\n".join(message_mismatch)
        )

    def get_message_by_content_type(self, response, content_type, message_pattern=""):
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

    def get_message_id(self, response, index=0):
        return response.get("messages", [])[index].get("id", "")

    def get_message_by_message_id(self, response, message_id: str):
        messages = response.get("messages", [])
        matched_message = None
        for message in messages:
            if message.get("id", "") == message_id:
                matched_message = message
                break
        if matched_message is None:
            raise ValueError(f"Failed to find a message with message id '{message_id}' in response")
        return matched_message

    def join_private_group(self, admin=None, member=None) -> str:

        private_group_name = f"private_group_{uuid4()}"
        response = admin.wakuext_service.create_group_chat_with_members([member.public_key], private_group_name)
        expected_group_creation_msg = f"@{admin.public_key} created the group {private_group_name}"
        expected_message = self.get_message_by_content_type(
            response,
            content_type=MessageContentType.SYSTEM_MESSAGE_CONTENT_PRIVATE_GROUP.value,
            message_pattern=expected_group_creation_msg,
        )[0]
        with member.expect_signal(SignalType.MESSAGES_NEW, pattern=expected_message.get("id"), timeout=60):
            pass
        return response.get("chats", [])[0].get("id")

    def create_community(self, node):
        response = node.wakuext_service.create_community(fake.community_name(), fake.community_description())
        self.community_id = response.get("communities", [{}])[0].get("id")
        return self.community_id

    def fetch_community(self, node, community_id=None, wait_for_response=True, try_database=True, **kwargs):
        if not community_id:
            community_id = self.community_id
        return node.wakuext_service.fetch_community(
            community_id,
            wait_for_response=wait_for_response,
            try_database=try_database,
            **kwargs,
        )

    def join_community(self, member=None, admin=None):
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

        def _is_accepted(state) -> bool:
            return state == 3 or str(state) == "3"

        def _is_community_not_found(error: ApiResponseError) -> bool:
            return "community not found" in str(error).lower()

        def _is_already_joined(error: ApiResponseError) -> bool:
            msg = str(error).lower()
            return "already joined" in msg or "already a member" in msg

        # Keep a single budget for all phases.
        deadline = time.monotonic() + 120
        logger.info("join_community: starting for community_id=%s", self.community_id)
        member_messages_start = len(member.received_signals[SignalType.MESSAGES_NEW])

        member_warmup_attempts = 0
        member_fast_fetch_attempts = 0
        member_blocking_fetch_attempts = 0
        member_last_fetch = None
        member_last_fetch_error = None
        member_last_spectate_error = None

        request_attempts = 0
        request_resend_attempts = 0
        last_request_error = None

        join_id = None
        join_state = None
        response_to_join = None

        last_pending = None
        last_latest_admin = None
        last_member_latest = None
        last_member_comm = None
        last_accept_error = None
        last_chat_messages_error = None
        last_poll_error = None
        last_accept_attempt_at = 0.0
        last_resend_at = 0.0
        accepted_seen = False
        accepted_via = None

        logger.info("join_community: Phase A — member visibility warm-up")
        phase_a_deadline = min(deadline, time.monotonic() + 50.0)
        warmup_timed_out = False
        while time.monotonic() < phase_a_deadline:
            member_warmup_attempts += 1
            member_fast_fetch_attempts += 1
            try:
                member_last_fetch = self.fetch_community(member, wait_for_response=False, try_database=False)
            except ApiResponseError as e:
                if not _is_community_not_found(e):
                    raise
                member_last_fetch_error = str(e)
                member_last_fetch = None

            if member_last_fetch:
                logger.info("join_community: Phase A — community visible after %d fast-fetch attempts", member_warmup_attempts)
                break

            remaining_phase_a = phase_a_deadline - time.monotonic()
            if member_warmup_attempts % 10 == 0 and remaining_phase_a > 20:
                member_blocking_fetch_attempts += 1
                try:
                    member_last_fetch = self.fetch_community(
                        member,
                        wait_for_response=True,
                        try_database=False,
                        timeout=10,
                    )
                except ApiResponseError as e:
                    if not _is_community_not_found(e):
                        raise
                    member_last_fetch_error = str(e)
                    member_last_fetch = None
                except Exception as e:
                    member_last_fetch_error = str(e)
                    member_last_fetch = None

                if member_last_fetch:
                    logger.info("join_community: Phase A — community visible after blocking fetch (attempt %d)", member_blocking_fetch_attempts)
                    break

            try:
                member.wakuext_service.spectate_community(self.community_id)
            except ApiResponseError as e:
                if not _is_community_not_found(e):
                    raise
                member_last_spectate_error = str(e)

            time.sleep(1)

        if not member_last_fetch:
            warmup_timed_out = True
            logger.warning(
                "join_community: Phase A — warm-up timed out after %d attempts (last_error=%s)", member_warmup_attempts, member_last_fetch_error
            )

        # Ensure admin is aware of the community before handling membership updates.
        self.fetch_community(admin)

        logger.info("join_community: Phase B — requesting to join")
        request_retry_interval = 25.0
        next_request_at = time.monotonic()
        while time.monotonic() < deadline and not join_id:
            now = time.monotonic()
            if now >= next_request_at:
                request_attempts += 1
                try:
                    response_to_join = member.wakuext_service.request_to_join_community(self.community_id)
                    join_req = (response_to_join.get("requestsToJoinCommunity") or [{}])[0]
                    join_id = join_req.get("id")
                    join_state = join_req.get("state")
                    if join_id:
                        logger.info("join_community: Phase B — got join_id=%s, state=%s (attempt %d)", join_id, join_state, request_attempts)
                        if _is_accepted(join_state):
                            accepted_seen = True
                            accepted_via = "rpc_response"
                        break
                    last_request_error = f"Missing join id in response: {response_to_join}"
                    logger.debug("join_community: Phase B — no join_id in response (attempt %d)", request_attempts)
                except ApiResponseError as e:
                    if _is_already_joined(e):
                        accepted_seen = True
                        accepted_via = "request_already_joined"
                        logger.info("join_community: Phase B — already joined")
                        break
                    elif _is_community_not_found(e):
                        last_request_error = str(e)
                        logger.debug("join_community: Phase B — community not found (attempt %d): %s", request_attempts, e)
                    else:
                        raise
                finally:
                    next_request_at = now + request_retry_interval

            # Keep nudging discovery while waiting for next request attempt.
            try:
                member.wakuext_service.spectate_community(self.community_id)
            except ApiResponseError as spectate_error:
                if not _is_community_not_found(spectate_error):
                    raise
                member_last_spectate_error = str(spectate_error)

            member_fast_fetch_attempts += 1
            try:
                member_last_fetch = self.fetch_community(member, wait_for_response=False, try_database=False)
                if member_last_fetch:
                    warmup_timed_out = False
            except ApiResponseError as e:
                if not _is_community_not_found(e):
                    raise
                member_last_fetch_error = str(e)
                member_last_fetch = None
            time.sleep(1)

        if not join_id and not accepted_seen:
            raise Exception(
                "Failed to request join for community within timeout. "
                f"community_id={self.community_id}, request_attempts={request_attempts}, "
                f"last_request_error={last_request_error}, response_to_join={response_to_join}, "
                f"warmup_timed_out={warmup_timed_out}, warmup_attempts={member_warmup_attempts}, "
                f"fast_fetch_attempts={member_fast_fetch_attempts}, blocking_fetch_attempts={member_blocking_fetch_attempts}, "
                f"member_last_fetch_error={member_last_fetch_error}, member_last_spectate_error={member_last_spectate_error}"
            )

        last_resend_at = time.monotonic()

        logger.info("join_community: Phase C — waiting for admin to accept join request (join_id=%s)", join_id)
        while time.monotonic() < deadline and not accepted_seen:
            try:
                last_member_latest = member.wakuext_service.latest_request_to_join_for_community(self.community_id)
                if last_member_latest and last_member_latest.get("id") == join_id and _is_accepted(last_member_latest.get("state")):
                    accepted_seen = True
                    accepted_via = "member_latest"
                    join_state = last_member_latest.get("state")
                    logger.info("join_community: Phase C — member sees accepted state via latest_request")
                    break
            except Exception as e:
                last_poll_error = f"member_latest: {e}"

            admin_observed_ids: set[str] = set()
            try:
                last_pending = admin.wakuext_service.pending_requests_to_join_for_community(self.community_id) or []
                admin_observed_ids.update([r.get("id") for r in last_pending if r.get("id")])
            except Exception as e:
                last_poll_error = f"pending_requests: {e}"

            try:
                last_latest_admin = admin.wakuext_service.latest_request_to_join_for_community(self.community_id)
                if last_latest_admin and last_latest_admin.get("id"):
                    admin_observed_ids.add(last_latest_admin.get("id"))
            except Exception as e:
                last_poll_error = f"latest_request: {e}"

            try:
                all_non_approved = admin.wakuext_service.all_non_approved_communities_requests_to_join() or []
                for request in all_non_approved:
                    if request.get("communityId") == self.community_id and request.get("id"):
                        admin_observed_ids.add(request.get("id"))
            except Exception as e:
                last_poll_error = f"all_non_approved: {e}"

            accept_id: str | None = None
            if join_id and join_id in admin_observed_ids:
                accept_id = join_id
            elif len(admin_observed_ids) == 1:
                accept_id = next(iter(admin_observed_ids))

            logger.debug("join_community: Phase C — admin observed request ids: %s, will accept: %s", admin_observed_ids, accept_id)

            if accept_id and (time.monotonic() - last_accept_attempt_at) >= 2.0:
                try:
                    accept_resp = admin.wakuext_service.accept_request_to_join_community(accept_id)
                    accept_state = (accept_resp.get("requestsToJoinCommunity") or [{}])[0].get("state")
                    if _is_accepted(accept_state):
                        accepted_seen = True
                        accepted_via = "admin_accept"
                        join_state = accept_state
                        logger.info("join_community: Phase C — admin accepted request %s", accept_id)
                        if accept_id != join_id:
                            join_id = accept_id
                except Exception as e:
                    last_accept_error = str(e)
                    logger.debug("join_community: Phase C — accept failed for %s: %s", accept_id, e)
                finally:
                    last_accept_attempt_at = time.monotonic()

            # Re-send join request every ~25 seconds while waiting for acceptance.
            if not accepted_seen and (time.monotonic() - last_resend_at) >= 25.0:
                logger.debug("join_community: Phase C — re-sending join request (resend attempt %d)", request_resend_attempts + 1)
                request_resend_attempts += 1
                try:
                    retry_resp = member.wakuext_service.request_to_join_community(self.community_id)
                    retry_req = (retry_resp.get("requestsToJoinCommunity") or [{}])[0]
                    if retry_req.get("id"):
                        join_id = retry_req.get("id")
                        join_state = retry_req.get("state")
                        if _is_accepted(join_state):
                            accepted_seen = True
                            accepted_via = "resend_rpc_response"
                except ApiResponseError as e:
                    if _is_already_joined(e):
                        accepted_seen = True
                        accepted_via = "resend_already_joined"
                    elif _is_community_not_found(e):
                        last_request_error = str(e)
                    else:
                        raise
                finally:
                    last_resend_at = time.monotonic()

            if not accepted_seen:
                time.sleep(1)

        # Best-effort wait for acceptance signal propagation.
        logger.info("join_community: accepted_seen=%s, accepted_via=%s", accepted_seen, accepted_via)
        if accepted_seen:
            try:
                with member.expect_signal(
                    SignalType.MESSAGES_NEW,
                    accept_fn=lambda signal: any(
                        request.get("id") == join_id and request.get("state") == 3
                        for request in (signal.get("event", {}).get("requestsToJoinCommunity") or [])
                    ),
                    start=member_messages_start,
                    timeout=5,
                ):
                    pass
                accepted_via = "member_signal"
            except Exception:
                pass

        # Phase D: completion condition.
        logger.info("join_community: Phase D — waiting for member to see community as joined")
        completion_attempt = 0
        while time.monotonic() < deadline:
            completion_attempt += 1
            try:
                last_member_comm = self.fetch_community(member, wait_for_response=False)
            except ApiResponseError as e:
                if not _is_community_not_found(e):
                    raise
                member_last_fetch_error = str(e)
                last_member_comm = None

            if not last_member_comm and completion_attempt % 10 == 0:
                try:
                    last_member_comm = self.fetch_community(member)
                except ApiResponseError as e:
                    if not _is_community_not_found(e):
                        raise
                    member_last_fetch_error = str(e)
                    last_member_comm = None

            is_joined = last_member_comm and (last_member_comm.get("joined") is True or last_member_comm.get("isMember") is True)
            chats = (last_member_comm.get("chats") or {}) if last_member_comm else {}
            chat_id = _pick_chat_id(chats)

            if completion_attempt % 5 == 0:
                logger.debug(
                    "join_community: Phase D — attempt %d, community_found=%s, is_joined=%s, chats=%d, chat_id=%s",
                    completion_attempt,
                    last_member_comm is not None,
                    is_joined,
                    len(chats),
                    chat_id,
                )

            if chat_id:
                full_chat_id = self.community_id + chat_id
                if is_joined:
                    logger.info("join_community: Phase D — member joined, chat_id=%s (attempt %d)", full_chat_id, completion_attempt)
                    return full_chat_id
                if accepted_seen:
                    try:
                        response = member.wakuext_service.chat_messages(full_chat_id, limit=1)
                        if response is not None and "messages" in response:
                            logger.info(
                                "join_community: Phase D — member can access chat messages, chat_id=%s (attempt %d)", full_chat_id, completion_attempt
                            )
                            return full_chat_id
                    except Exception as e:
                        last_chat_messages_error = str(e)

            time.sleep(1)

        raise Exception(
            "Failed to complete community join within timeout. "
            f"community_id={self.community_id}, join_id={join_id}, join_state={join_state}, "
            f"accepted_seen={accepted_seen}, accepted_via={accepted_via}, "
            f"request_attempts={request_attempts}, resend_attempts={request_resend_attempts}, "
            f"member_warmup_attempts={member_warmup_attempts}, "
            f"member_fast_fetch_attempts={member_fast_fetch_attempts}, "
            f"member_blocking_fetch_attempts={member_blocking_fetch_attempts}, "
            f"member_last_fetch_error={member_last_fetch_error}, member_last_spectate_error={member_last_spectate_error}, "
            f"last_pending={last_pending}, last_latest_admin={last_latest_admin}, last_member_latest={last_member_latest}, "
            f"last_member_joined={getattr(last_member_comm, 'get', lambda *_: None)('joined') if last_member_comm else None}, "
            f"last_member_is_member={getattr(last_member_comm, 'get', lambda *_: None)('isMember') if last_member_comm else None}, "
            f"last_member_chats_keys={list((last_member_comm or {}).get('chats', {}).keys()) if last_member_comm else None}, "
            f"last_request_error={last_request_error}, last_accept_error={last_accept_error}, "
            f"last_chat_messages_error={last_chat_messages_error}, last_poll_error={last_poll_error}"
        )

    def wait_for_community_chat_visible(self, member, community_id, chat_id, timeout=60):
        """Force member to sync community from store node until the new chat appears.

        After join_community, Waku relay subscriptions may be in a transitional
        state and the member can miss community description updates published via
        relay.  This method bypasses relay by polling the store node directly
        (tryDatabase=False forces a network fetch through the store protocol).
        """
        deadline = time.time() + timeout
        last_error = None
        last_chats_keys = None
        last_joined = None
        last_is_member = None
        while time.time() < deadline:
            try:
                comm = self.fetch_community(
                    member,
                    community_id,
                    wait_for_response=True,
                    try_database=False,
                    timeout=10,
                )
            except ApiResponseError as e:
                last_error = e
                time.sleep(2)
                continue
            if comm:
                chats = comm.get("chats") or {}
                last_chats_keys = list(chats.keys())
                last_joined = comm.get("joined")
                last_is_member = comm.get("isMember")
                if chat_id in chats or any(v.get("id") == chat_id for v in chats.values() if isinstance(v, dict)):
                    return
            time.sleep(2)
        raise TimeoutError(
            f"Member did not see chat {chat_id} in community {community_id} "
            f"within {timeout}s. last_error={last_error}, "
            f"last_chats_keys={last_chats_keys}, "
            f"last_joined={last_joined}, last_is_member={last_is_member}"
        )

    @retry(stop=stop_after_delay(20), wait=wait_fixed(0.5), reraise=True)
    def leave_the_community(self, node, community_id=None):
        if not community_id:
            community_id = self.community_id
        response = node.wakuext_service.leave_community(community_id)
        target_community = [existing_community for existing_community in response.get("communities") if existing_community.get("id") == community_id][
            0
        ]
        assert target_community.get("joined") is False

    @retry(stop=stop_after_delay(20), wait=wait_fixed(2), reraise=True)
    def check_node_joined_community(self, node, joined, community_id=None):
        if not community_id:
            community_id = self.community_id
        response = self.fetch_community(node, community_id)
        assert response.get("joined") is joined

    def community_messages(self, message_chat_id, message_count, sender=None, receiver=None):
        start_index = len(receiver.received_signals[SignalType.MESSAGES_NEW])
        sent_messages = []
        for i in range(message_count):
            message_text = f"test_message_{i+1}_{uuid4()}"
            response = sender.wakuext_service.send_chat_message(message_chat_id, message_text)
            expected_message = self.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
            sent_messages.append(expected_message)
            time.sleep(0.01)

        for i, expected_message in enumerate(sent_messages):
            with receiver.expect_signal(SignalType.MESSAGES_NEW, pattern=expected_message.get("id"), timeout=60, start=start_index) as exp:
                pass
            messages_new_event = exp.result
            self.validate_signal_event_against_response(
                signal_event=messages_new_event,
                fields_to_validate={"text": "text"},
                expected_message=expected_message,
            )

    def one_to_one_message(self, message_count, sender=None, receiver=None):
        start_index = len(receiver.received_signals[SignalType.MESSAGES_NEW])
        _, responses = self.send_multiple_one_to_one_messages(message_count, sender=sender, receiver=receiver)
        messages = list(map(lambda r: r.get("messages", [])[0], responses))

        for expected_message in messages:
            with receiver.expect_signal(SignalType.MESSAGES_NEW, pattern=expected_message.get("id"), timeout=60, start=start_index) as exp:
                pass
            messages_new_event = exp.result
            self.validate_signal_event_against_response(
                signal_event=messages_new_event,
                fields_to_validate={"text": "text"},
                expected_message=expected_message,
            )

        return responses

    def send_multiple_one_to_one_messages(self, message_count=1, sender=None, receiver=None) -> tuple[list[str], list[dict]]:
        sent_texts = []
        responses = []

        for i in range(message_count):
            message_text = f"test_message_{i}_{uuid4()}"
            sent_texts.append(message_text)
            response = sender.wakuext_service.send_one_to_one_message(receiver.public_key, message_text)
            responses.append(response)

        return sent_texts, responses

    def add_contact(self, sender=None, receiver=None, execution_number=None, network_condition=None):
        message_text = f"test_contact_request_{execution_number}_{uuid4()}"
        existing_contacts = receiver.wakuext_service.get_contacts()

        if sender.public_key in str(existing_contacts):
            pytest.skip("Contact request was already sent for this sender<->receiver. Skipping test!!")

        if network_condition:
            network_condition(receiver)

        response = sender.wakuext_service.send_contact_request(receiver.public_key, message_text)
        expected_message = self.get_message_by_content_type(response, content_type=MessageContentType.CONTACT_REQUEST.value)[0]

        with receiver.expect_signal(SignalType.MESSAGES_NEW, pattern=expected_message.get("id"), timeout=60) as exp:
            pass
        messages_new_event = exp.result

        signal_messages_texts = []
        if "messages" in messages_new_event.get("event", {}):
            signal_messages_texts.extend(message["text"] for message in messages_new_event["event"]["messages"] if "text" in message)

        assert (
            f"@{sender.public_key} sent you a contact request" in signal_messages_texts
        ), "Couldn't find the signal corresponding to the contact request"

        self.validate_signal_event_against_response(
            signal_event=messages_new_event,
            fields_to_validate={"text": "text"},
            expected_message=expected_message,
        )

    def create_private_group(self, private_groups_count, admin, member):
        """Create one or more private groups between admin and member and validate signals.

        Args:
            private_groups_count: number of private groups to create
            admin: node creating the group (required)
            member: node invited to the group (required)
        """
        start_index = len(member.received_signals[SignalType.MESSAGES_NEW])
        private_groups = []
        for i in range(private_groups_count):
            private_group_name = f"private_group_{i+1}_{uuid4()}"
            response = admin.wakuext_service.create_group_chat_with_members([member.public_key], private_group_name)

            expected_group_creation_msg = f"@{admin.public_key} created the group {private_group_name}"
            expected_message = self.get_message_by_content_type(
                response,
                content_type=MessageContentType.SYSTEM_MESSAGE_CONTENT_PRIVATE_GROUP.value,
                message_pattern=expected_group_creation_msg,
            )[0]

            private_groups.append(expected_message)
            time.sleep(0.01)

        for i, expected_message in enumerate(private_groups):
            with member.expect_signal(SignalType.MESSAGES_NEW, pattern=expected_message.get("id"), timeout=60, start=start_index) as exp:
                pass
            messages_new_event = exp.result
            self.validate_signal_event_against_response(
                signal_event=messages_new_event,
                expected_message=expected_message,
                fields_to_validate={"text": "text"},
            )

    def private_group_message(self, message_count, private_group_id, sender=None, receiver=None):
        start_index = len(receiver.received_signals[SignalType.MESSAGES_NEW])
        sent_messages = []
        for i in range(message_count):
            message_text = f"test_message_{i+1}_{uuid4()}"
            response = sender.wakuext_service.send_group_chat_message(private_group_id, message_text)
            expected_message = self.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
            sent_messages.append(expected_message)
            time.sleep(0.01)

        for _, expected_message in enumerate(sent_messages):
            with receiver.expect_signal(SignalType.MESSAGES_NEW, pattern=expected_message.get("id"), timeout=60, start=start_index) as exp:
                pass
            messages_new_event = exp.result
            self.validate_signal_event_against_response(
                signal_event=messages_new_event,
                fields_to_validate={"text": "text"},
                expected_message=expected_message,
            )
