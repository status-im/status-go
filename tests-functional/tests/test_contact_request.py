from time import sleep
from uuid import uuid4
import pytest
from test_cases import OneToOneMessageTestCase
from constants import DEFAULT_DISPLAY_NAME
from clients.signals import SignalType

@pytest.mark.rpc
class TestContactRequests(OneToOneMessageTestCase):


    @pytest.mark.dependency(name="test_contact_request_baseline")
    def test_contact_request_baseline(self, contact_request_count=1):

        await_signals = [
            SignalType.MESSAGES_NEW.value,
            SignalType.MESSAGE_DELIVERED.value,
        ]

        for i in range(contact_request_count):

            message_text = f"test_contact_request_{i+1}_{uuid4()}"

            sender = self.initialize_backend(await_signals=await_signals)
            receiver  = self.initialize_backend(await_signals=await_signals)

            pk_sender = sender.get_pubkey(DEFAULT_DISPLAY_NAME)
            pk_receiver = receiver.get_pubkey(DEFAULT_DISPLAY_NAME)

            existing_contacts = receiver.get_accounts()

            response = sender.send_contact_request([{"id": pk_receiver, "message": message_text}])

            messages_new_event = receiver.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=message_text, timeout=60)

            signal_texts = []
            if "messages" in messages_new_event.get("event", {}):
                signal_texts.extend(
                    message["text"]
                    for message in messages_new_event["event"]["messages"]
                    if "text" in message
                )
            if "chats" in messages_new_event.get("event", {}) and messages_new_event["event"]["chats"]:
                last_message = messages_new_event["event"]["chats"][0].get("lastMessage", {})
                if "text" in last_message:
                    signal_texts.append(last_message["text"])

            if pk_sender not in str(existing_contacts): # we check that the contact request wasn"t already sent for this sender
                assert f"@{pk_sender} sent you a contact request" in signal_texts, "Couldn't find the signal corresponding to the contact request"

            self.validate_event_against_response(
                messages_new_event,
                fields_to_validate={
                    "text": "text",
                    "id": "id",
                },
                response=response,
            )

    @pytest.mark.skip(reason="Skipping because of error 'Not enough status-backend containers, please add more'. Unkipping when we merge https://github.com/status-im/status-go/pull/6159")
    @pytest.mark.dependency(depends=["test_contact_request_baseline"])
    def test_multiple_contact_requests(self):
        self.test_contact_request_baseline(contact_request_count=10)

    @pytest.mark.dependency(depends=["test_contact_request_baseline"])
    @pytest.mark.skip(reason="Skipping until add_latency is implemented")
    def test_contact_request_with_latency(self):
        with self.add_latency():
            self.test_contact_request_baseline()

    @pytest.mark.dependency(depends=["test_contact_request_baseline"])
    @pytest.mark.skip(reason="Skipping until add_packet_loss is implemented")
    def test_contact_request_with_packet_loss(self):
        with self.add_packet_loss():
            self.test_contact_request_baseline()

    @pytest.mark.dependency(depends=["test_contact_request_baseline"])
    @pytest.mark.skip(reason="Skipping until add_low_bandwith is implemented")
    def test_contact_request_with_low_bandwidth(self):
        with self.add_low_bandwith():
            self.test_contact_request_baseline()

    @pytest.mark.dependency(depends=["test_contact_request_baseline"])
    @pytest.mark.skip(reason="Skipping until node_pause is implemented")
    def test_contact_request_with_node_pause_30_seconds(self):
        await_signals = [
            SignalType.MESSAGES_NEW.value,
            SignalType.MESSAGE_DELIVERED.value,
        ]
        sender = self.initialize_backend(await_signals=await_signals)
        receiver  = self.initialize_backend(await_signals=await_signals)
        pk_receiver = receiver.get_pubkey(DEFAULT_DISPLAY_NAME)

        with self.node_pause(receiver):
            message_text = f"test_contact_request_{uuid4()}"
            sender.send_contact_request([{"id": pk_receiver, "message": message_text}])
            sleep(30)
        receiver.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=message_text)
        sender.wait_for_signal("messages.delivered")
