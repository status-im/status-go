import asyncio
import logging
from uuid import uuid4

from clients.signals import SignalType
from resources.enums import MessageContentType
from utils import fake


class AsyncMessengerSteps:
    """Base class with helper methods for async messenger tests.

    RPC calls are sync (fast, no need for async).
    Only signal waiting is async.
    """

    def get_message_by_content_type(self, response: dict, content_type: int, message_pattern: str = "") -> list[dict]:
        """Filter messages from response by content type."""
        matched_messages = []
        messages = response.get("messages", [])
        for message in messages:
            if message.get("contentType") != content_type:
                continue
            if not message_pattern or message_pattern in str(message):
                matched_messages.append(message)
        if matched_messages:
            return matched_messages
        raise ValueError(f"Failed to find a message with contentType '{content_type}' and message_pattern: `{message_pattern}` in response")

    def get_message_id(self, response: dict, index: int = 0) -> str:
        return response.get("messages", [])[index].get("id", "")

    def get_message_by_message_id(self, response: dict, message_id: str) -> dict:
        messages = response.get("messages", [])
        for message in messages:
            if message.get("id", "") == message_id:
                return message
        raise ValueError(f"Failed to find a message with message id '{message_id}' in response")

    async def send_contact_request_and_wait(self, sender, receiver, message_text: str = "contact_request") -> str:
        """Send contact request and wait for receiver to get the signal.

        Returns:
            str: The message ID of the contact request
        """
        # Sync RPC call
        response = sender.wakuext_service.send_contact_request(receiver.public_key, message_text)
        expected_message = self.get_message_by_content_type(response, MessageContentType.CONTACT_REQUEST.value)[0]
        message_id = expected_message.get("id")
        assert message_id, "Message ID should not be empty"

        # Async signal waiting
        await receiver.wait_for_signal(SignalType.MESSAGES_NEW, pattern=message_id, timeout=60, check_buffer=True)
        return message_id

    async def accept_contact_request_and_wait(self, message_id: str, sender, receiver) -> None:
        """Accept contact request and wait for sender to get the acceptance signal."""
        accepted_signal = f"@{receiver.public_key} accepted your contact request"
        # Sync RPC call
        receiver.wakuext_service.accept_contact_request(message_id, sender.public_key)
        # Async signal waiting
        await sender.wait_for_signal(SignalType.MESSAGES_NEW, pattern=accepted_signal, timeout=60, check_buffer=True)

    async def make_contacts(self, sender, receiver) -> str:
        """Create a mutual contact between sender and receiver.

        Returns:
            str: The message ID of the contact request
        """
        # Sync RPC call
        existing_contacts = receiver.wakuext_service.get_contacts()
        if sender.public_key in str(existing_contacts):
            return ""

        message_id = await self.send_contact_request_and_wait(sender, receiver)
        await self.accept_contact_request_and_wait(message_id, sender, receiver)
        return message_id

    def validate_signal_event_against_response(self, signal_event: dict, fields_to_validate: dict, expected_message: dict) -> None:
        """Validate that signal event contains expected message with matching fields."""
        expected_message_id = expected_message.get("id")
        signal_event_messages = signal_event.get("event", {}).get("messages")
        assert signal_event_messages and len(signal_event_messages) > 0, "No messages found in the signal event"

        message = next(
            (msg for msg in signal_event_messages if msg.get("id") == expected_message_id),
            None,
        )
        assert message, f"Message with ID {expected_message_id} not found in the signal event"

        message_mismatch = []
        for response_field, event_field in fields_to_validate.items():
            response_value = expected_message[response_field]
            event_value = message[event_field]
            if response_value != event_value:
                message_mismatch.append(f"Field '{response_field}': Expected '{response_value}', Found '{event_value}'")

        if message_mismatch:
            raise AssertionError(
                "Some Sender RPC responses are not matching the signals received by the receiver.\n"
                "Details of mismatches:\n" + "\n".join(message_mismatch)
            )

    def create_community(self, node) -> str:
        """Create a community using the given node.

        This is a sync RPC call, no async needed.
        """
        # Handle both sync and async backends
        wakuext = getattr(node, "wakuext_service", None)
        if wakuext is None and hasattr(node, "backend"):
            wakuext = node.backend.wakuext_service
        assert wakuext is not None, "Node must have wakuext_service attribute"
        response = wakuext.create_community(fake.community_name(), fake.community_description())
        self.community_id = response.get("communities", [{}])[0].get("id")
        return self.community_id

    def fetch_community(self, node, community_id: str = "") -> dict:
        """Fetch community info.

        This is a sync RPC call, no async needed.
        """
        if not community_id:
            community_id = self.community_id
        wakuext = getattr(node, "wakuext_service", None)
        if wakuext is None and hasattr(node, "backend"):
            wakuext = node.backend.wakuext_service
        if wakuext is None:
            raise ValueError("Node must have wakuext_service attribute")
        return wakuext.fetch_community(community_id)

    async def join_private_group(self, admin, member) -> str:
        """Create a private group and wait for member to receive the signal.

        Returns:
            str: The private group chat ID
        """
        private_group_name = f"private_group_{uuid4()}"
        response = admin.wakuext_service.create_group_chat_with_members([member.public_key], private_group_name)
        expected_group_creation_msg = f"@{admin.public_key} created the group {private_group_name}"
        expected_message = self.get_message_by_content_type(
            response,
            content_type=MessageContentType.SYSTEM_MESSAGE_CONTENT_PRIVATE_GROUP.value,
            message_pattern=expected_group_creation_msg,
        )[0]
        await member.wait_for_signal(SignalType.MESSAGES_NEW, pattern=expected_message.get("id"), timeout=60, check_buffer=True)
        return response.get("chats", [])[0].get("id")

    async def _async_retry_call(self, func, *args, max_attempts: int = 40, retry_interval: float = 0.5):
        """Async version of retry_call - simple RPC-level retry.

        This mirrors the sync retry_call from develop branch which works reliably
        for both full node and light client modes.

        Args:
            func: RPC function to call
            *args: Arguments to pass to the function
            max_attempts: Maximum number of retry attempts (default: 40)
            retry_interval: Seconds between retries (default: 0.5)

        Returns:
            Response from the function

        Raises:
            Exception: If all retries fail
        """
        for attempt in range(max_attempts):
            try:
                response = func(*args)
                if response is not None:  # Check for None, not truthiness (empty dict is valid)
                    return response
            except Exception as e:
                logging.debug(f"Attempt {attempt + 1}/{max_attempts} for {func.__name__} failed: {e}")
            await asyncio.sleep(retry_interval)
        raise Exception(f"Failed to execute {func.__name__} within {max_attempts * retry_interval}s")

    def _pick_chat_id(self, chats: dict) -> str | None:
        """Pick the first valid chat ID from community chats, sorted by position.

        Communities have multiple chats, we want the "general" chat which typically
        has position=0. This matches the logic from the original implementation.
        """
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

    async def join_community(self, member, admin) -> str:
        """Join member to community created by admin.

        Uses a two-phase approach:
        1. Wait for the admin to see the pending join request (propagation via Waku)
        2. Accept the request once visible

        This is critical for light client mode where Waku message propagation
        can take significantly longer than in relay mode.

        Returns:
            str: The community chat ID (community_id + chat_id)
        """
        # Fetch community with retry for light client (may need time to sync)
        community = None
        for attempt in range(12):
            community = self.fetch_community(member, self.community_id)
            if community and community.get("chats"):
                break
            logging.debug(f"Community {self.community_id} not ready on member (attempt {attempt + 1}/12)")
            await asyncio.sleep(5)

        if not community:
            raise Exception(f"Community {self.community_id} not visible to member after retries")

        # Ensure admin has community data (triggers filter subscriptions for community topics)
        self.fetch_community(admin, self.community_id)
        # In light client mode, filter subscriptions are async and take several seconds.
        # If the member sends the join request before the admin subscribes,
        # the message is lost (filter protocol doesn't deliver historical messages).
        if getattr(admin, "waku_light_client", False):
            await asyncio.sleep(10)

        # Request to join
        response_to_join = member.wakuext_service.request_to_join_community(self.community_id)
        join_req = (response_to_join.get("requestsToJoinCommunity") or [{}])[0]
        join_id = join_req.get("id")
        join_state = join_req.get("state")
        assert join_id, f"Failed to request to join community: {response_to_join}"

        def _is_accepted(state) -> bool:
            return state == 3 or str(state) == "3"

        # Auto-accept: if state is already accepted, skip admin accept phase
        if _is_accepted(join_state):
            community = self.fetch_community(member, self.community_id)
            chats = community.get("chats", {}) if community else {}
            chat_id = self._pick_chat_id(chats)
            if chat_id:
                return self.community_id + chat_id

        # Phase 1: Wait for admin to see the pending request.
        # In light client mode, the join request propagates via Waku filter protocol
        # and can take much longer than in relay mode.
        # Check multiple endpoints (like sync version) since the request may appear
        # in one before the others due to eventual consistency.
        accept_id = None
        last_pending = None
        last_latest = None
        for attempt in range(120):
            admin_observed_ids: set[str] = set()
            try:
                last_pending = admin.wakuext_service.pending_requests_to_join_for_community(self.community_id) or []
                for req in last_pending:
                    if req.get("id"):
                        admin_observed_ids.add(req["id"])
            except Exception as e:
                logging.debug(f"Attempt {attempt + 1}/120 pending_requests check failed: {e}")

            try:
                last_latest = admin.wakuext_service.latest_request_to_join_for_community(self.community_id)
                if last_latest and last_latest.get("id"):
                    admin_observed_ids.add(last_latest["id"])
            except Exception as e:
                logging.debug(f"Attempt {attempt + 1}/120 latest_request check failed: {e}")

            try:
                all_non_approved = admin.wakuext_service.all_non_approved_communities_requests_to_join() or []
                for r in all_non_approved:
                    if r.get("communityId") == self.community_id and r.get("id"):
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
            # (e.g. published before admin's filter subscription was active)
            if attempt > 0 and attempt % 30 == 0:
                logging.debug(f"Re-sending join request (attempt {attempt}/120)")
                try:
                    retry_resp = member.wakuext_service.request_to_join_community(self.community_id)
                    retry_req = (retry_resp.get("requestsToJoinCommunity") or [{}])[0]
                    if retry_req.get("id"):
                        join_id = retry_req["id"]
                except Exception as e:
                    logging.debug(f"Re-send join request failed: {e}")

            await asyncio.sleep(1)

        if not accept_id:
            raise Exception(
                f"Admin never saw pending request for community {self.community_id} "
                f"(join_id={join_id}) after 120s. "
                f"last_pending={last_pending}, last_latest={last_latest}"
            )

        # Phase 2: Accept the request (should succeed quickly now that it's visible)
        response = await self._async_retry_call(
            admin.wakuext_service.accept_request_to_join_community,
            accept_id,
            max_attempts=20,
            retry_interval=1.0,
        )

        # Validate accept was successful (state=3 means accepted)
        accept_state = (response.get("requestsToJoinCommunity") or [{}])[0].get("state")
        if not _is_accepted(accept_state):
            raise Exception(f"Accept request failed. State: {accept_state}, Response: {response}")

        # Extract chat_id from response (sorted by position, pick first valid)
        chats = response.get("communities", [{}])[0].get("chats", {})
        chat_id = self._pick_chat_id(chats)
        if not chat_id:
            raise Exception(f"No valid chat found in community response: {response}")
        return self.community_id + chat_id

    def send_multiple_one_to_one_messages(self, message_count: int, sender, receiver) -> tuple[list[str], list[dict]]:
        """Send multiple one-to-one messages.

        Returns:
            tuple: (list of sent texts, list of responses)
        """
        sent_texts = []
        responses = []
        for i in range(message_count):
            message_text = f"test_message_{i}_{uuid4()}"
            sent_texts.append(message_text)
            response = sender.wakuext_service.send_one_to_one_message(receiver.public_key, message_text)
            responses.append(response)
        return sent_texts, responses
