import logging
import random
import string
import time
from uuid import uuid4
import pytest
from tenacity import retry, stop_after_delay, wait_fixed
from clients.signals import SignalType
from clients.status_backend import StatusBackend
from resources.constants import USE_IPV6
from resources.enums import MessageContentType
from steps.network_conditions import NetworkConditionsSteps


class MessengerSteps(NetworkConditionsSteps):

    await_signals = [
        SignalType.MESSAGES_NEW.value,
        SignalType.MESSAGE_DELIVERED.value,
        SignalType.NODE_LOGIN.value,
        SignalType.NODE_LOGOUT.value,
    ]

    @pytest.fixture(scope="function", autouse=False)
    def setup_two_privileged_nodes(self, request):
        request.cls.sender = self.sender = self.initialize_backend(self.await_signals, True)
        request.cls.receiver = self.receiver = self.initialize_backend(self.await_signals, True)

    @pytest.fixture(scope="function", autouse=False)
    def setup_two_unprivileged_nodes(self, request):
        request.cls.sender = self.sender = self.initialize_backend(self.await_signals, False)
        request.cls.receiver = self.receiver = self.initialize_backend(self.await_signals, False)

    def initialize_backend(self, await_signals, privileged=True, ipv6=USE_IPV6, **kwargs):
        backend = StatusBackend(await_signals, privileged=privileged, ipv6=ipv6)
        backend.init_status_backend()
        backend.create_account_and_login(**kwargs)
        backend.find_public_key()
        backend.wakuext_service.start_messenger()
        return backend

    def send_contact_request_and_wait_for_signal_to_be_received(self):
        response = self.sender.wakuext_service.send_contact_request(self.receiver.public_key, "contact_request")
        expected_message = self.get_message_by_content_type(response, content_type=MessageContentType.CONTACT_REQUEST.value)[0]
        message_id = expected_message.get("id")
        self.receiver.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=message_id)
        return message_id

    def accept_contact_request_and_wait_for_signal_to_be_received(self, message_id):
        self.receiver.wakuext_service.accept_contact_request(message_id)
        accepted_signal = f"@{self.receiver.public_key} accepted your contact request"
        self.sender.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=accepted_signal)

    def make_contacts(self):
        existing_contacts = self.receiver.wakuext_service.get_contacts()

        if self.sender.public_key in str(existing_contacts):
            return

        message_id = self.send_contact_request_and_wait_for_signal_to_be_received()
        self.accept_contact_request_and_wait_for_signal_to_be_received(message_id)
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
        messages = response.get("result", {}).get("messages", [])
        for message in messages:
            if message.get("contentType") != content_type:
                continue
            if not message_pattern or message_pattern in str(message):
                matched_messages.append(message)
        if matched_messages:
            return matched_messages
        else:
            raise ValueError(f"Failed to find a message with contentType '{content_type}' in response")

    def join_private_group(self):
        private_group_name = f"private_group_{uuid4()}"
        response = self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], private_group_name)
        expected_group_creation_msg = f"@{self.sender.public_key} created the group {private_group_name}"
        expected_message = self.get_message_by_content_type(
            response,
            content_type=MessageContentType.SYSTEM_MESSAGE_CONTENT_PRIVATE_GROUP.value,
            message_pattern=expected_group_creation_msg,
        )[0]
        self.receiver.find_signal_containing_pattern(
            SignalType.MESSAGES_NEW.value,
            event_pattern=expected_message.get("id"),
            timeout=60,
        )
        return response.get("result", {}).get("chats", [])[0].get("id")

    def create_community(self, node):
        name = f"vac_qa_community_{''.join(random.choices(string.ascii_letters, k=10))}"
        response = node.wakuext_service.create_community(name)
        self.community_id = response.get("result", {}).get("communities", [{}])[0].get("id")
        return self.community_id

    def fetch_community(self, node, community_id=None):
        if not community_id:
            community_id = self.community_id
        return node.wakuext_service.fetch_community(community_id)

    def join_community(self, node):
        self.fetch_community(node)
        response_to_join = node.wakuext_service.request_to_join_community(self.community_id)
        join_id = response_to_join.get("result", {}).get("requestsToJoinCommunity", [{}])[0].get("id")

        # I couldn't find any signal related to the requestToJoinCommunity request in the peer node.
        # That's why I need this retry logic for accepting the request to join the community.
        max_retries = 40
        retry_interval = 0.5
        for attempt in range(max_retries):
            try:
                response = self.sender.wakuext_service.accept_request_to_join_community(join_id)
                if response.get("result"):
                    break
            except Exception as e:
                logging.error(f"Attempt {attempt + 1}/{max_retries}: Unexpected error: {e}")
                time.sleep(retry_interval)
        else:
            raise Exception(f"Failed to accept request to join community in {max_retries * retry_interval} seconds.")

        chats = response.get("result", {}).get("communities", [{}])[0].get("chats", {})
        chat_id = list(chats.keys())[0] if chats else None
        return self.community_id + chat_id

    @retry(stop=stop_after_delay(20), wait=wait_fixed(0.5), reraise=True)
    def leave_the_community(self, node, community_id=None):
        if not community_id:
            community_id = self.community_id
        response = node.wakuext_service.leave_community(community_id)
        target_community = [
            existing_community for existing_community in response.get("result", {}).get("communities") if existing_community.get("id") == community_id
        ][0]
        assert target_community.get("joined") is False

    @retry(stop=stop_after_delay(20), wait=wait_fixed(2), reraise=True)
    def check_node_joined_community(self, node, joined, community_id=None):
        if not community_id:
            community_id = self.community_id
        response = self.fetch_community(node, community_id)
        assert response.get("result", {}).get("joined") is joined

    def community_messages(self, message_chat_id, message_count):
        sent_messages = []
        for i in range(message_count):
            message_text = f"test_message_{i+1}_{uuid4()}"
            response = self.sender.wakuext_service.send_community_chat_message(message_chat_id, message_text)
            expected_message = self.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
            sent_messages.append(expected_message)
            time.sleep(0.01)

        for i, expected_message in enumerate(sent_messages):
            messages_new_event = self.receiver.find_signal_containing_pattern(
                SignalType.MESSAGES_NEW.value,
                event_pattern=expected_message.get("id"),
                timeout=60,
            )
            self.validate_signal_event_against_response(
                signal_event=messages_new_event,
                fields_to_validate={"text": "text"},
                expected_message=expected_message,
            )

    def one_to_one_message(self, message_count):
        _, responses = self.send_multiple_one_to_one_messages(message_count)
        messages = list(map(lambda r: r.get("result", {}).get("messages", [])[0], responses))

        for expected_message in messages:
            messages_new_event = self.receiver.find_signal_containing_pattern(
                SignalType.MESSAGES_NEW.value,
                event_pattern=expected_message.get("id"),
                timeout=60,
            )
            self.validate_signal_event_against_response(
                signal_event=messages_new_event,
                fields_to_validate={"text": "text"},
                expected_message=expected_message,
            )

        return responses

    def send_multiple_one_to_one_messages(self, message_count=1) -> tuple[list[str], list[dict]]:
        sent_texts = []
        responses = []

        for i in range(message_count):
            message_text = f"test_message_{i}_{uuid4()}"
            sent_texts.append(message_text)
            response = self.sender.wakuext_service.send_one_to_one_message(self.receiver.public_key, message_text)
            responses.append(response)

        return sent_texts, responses

    def add_contact(self, execution_number, network_condition=None, privileged=True):
        message_text = f"test_contact_request_{execution_number}_{uuid4()}"
        sender = self.initialize_backend(await_signals=self.await_signals, privileged=privileged)
        receiver = self.initialize_backend(await_signals=self.await_signals, privileged=privileged)

        existing_contacts = receiver.wakuext_service.get_contacts()

        if sender.public_key in str(existing_contacts):
            pytest.skip("Contact request was already sent for this sender<->receiver. Skipping test!!")

        if network_condition:
            network_condition(receiver)

        response = sender.wakuext_service.send_contact_request(receiver.public_key, message_text)
        expected_message = self.get_message_by_content_type(response, content_type=MessageContentType.CONTACT_REQUEST.value)[0]

        messages_new_event = receiver.find_signal_containing_pattern(
            SignalType.MESSAGES_NEW.value,
            event_pattern=expected_message.get("id"),
            timeout=60,
        )

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

    def create_private_group(self, private_groups_count):
        private_groups = []
        for i in range(private_groups_count):
            private_group_name = f"private_group_{i+1}_{uuid4()}"
            response = self.sender.wakuext_service.create_group_chat_with_members([self.receiver.public_key], private_group_name)

            expected_group_creation_msg = f"@{self.sender.public_key} created the group {private_group_name}"
            expected_message = self.get_message_by_content_type(
                response,
                content_type=MessageContentType.SYSTEM_MESSAGE_CONTENT_PRIVATE_GROUP.value,
                message_pattern=expected_group_creation_msg,
            )[0]

            private_groups.append(expected_message)
            time.sleep(0.01)

        for i, expected_message in enumerate(private_groups):
            messages_new_event = self.receiver.find_signal_containing_pattern(
                SignalType.MESSAGES_NEW.value,
                event_pattern=expected_message.get("id"),
                timeout=60,
            )
            self.validate_signal_event_against_response(
                signal_event=messages_new_event,
                expected_message=expected_message,
                fields_to_validate={"text": "text"},
            )

    def private_group_message(self, message_count, private_group_id):
        sent_messages = []
        for i in range(message_count):
            message_text = f"test_message_{i+1}_{uuid4()}"
            response = self.sender.wakuext_service.send_group_chat_message(private_group_id, message_text)
            expected_message = self.get_message_by_content_type(response, content_type=MessageContentType.TEXT_PLAIN.value)[0]
            sent_messages.append(expected_message)
            time.sleep(0.01)

        for _, expected_message in enumerate(sent_messages):
            messages_new_event = self.receiver.find_signal_containing_pattern(
                SignalType.MESSAGES_NEW.value,
                event_pattern=expected_message.get("id"),
                timeout=60,
            )
            self.validate_signal_event_against_response(
                signal_event=messages_new_event,
                fields_to_validate={"text": "text"},
                expected_message=expected_message,
            )
