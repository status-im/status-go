from time import sleep
from uuid import uuid4
import pytest
from test_cases import MessengerTestCase
from constants import DEFAULT_DISPLAY_NAME
from clients.signals import SignalType

@pytest.mark.usefixtures("setup_two_nodes")
@pytest.mark.rpc
class TestCreatePrivateGroups(MessengerTestCase):

    @pytest.mark.dependency(name="test_create_private_group_baseline")
    def test_create_private_group_baseline(self, private_groups_count=1):
        pk_sender = self.sender.get_pubkey(DEFAULT_DISPLAY_NAME)
        pk_receiver = self.receiver.get_pubkey(DEFAULT_DISPLAY_NAME)

        response = self.sender.send_contact_request(pk_receiver, "contact_request")
        chat_id = self.get_message_id(response)
        self.receiver.accept_contact_request(chat_id)

        private_groups = []
        for i in range(private_groups_count):
            private_group_name = f"private_group_{i+1}_{uuid4()}"
            response = self.sender.create_group_chat_with_members([pk_receiver], private_group_name)

            messages = response.get("result", {}).get("messages", [])
            messages_texts = [message["text"] for message in messages]

            expected_group_creation_msg = f"@{pk_sender} created the group {private_group_name}"
            assert expected_group_creation_msg in messages_texts

            expected_message = next((message for message in messages if expected_group_creation_msg == message["text"]), None)
            private_groups.append(expected_message)
            sleep(0.01)


        for i, expected_message in enumerate(private_groups):
            messages_new_event = self.receiver.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=expected_message.get("id"), timeout=60)
            self.validate_signal_event_against_response(
                signal_event=messages_new_event,
                expected_message=expected_message,
                fields_to_validate={"text": "text"},
            )

    @pytest.mark.dependency(depends=["test_create_private_group_baseline"])
    def test_multiple_one_create_private_groups(self):
        self.test_create_private_group_baseline(message_count=50)

    @pytest.mark.dependency(depends=["test_create_private_group_baseline"])
    @pytest.mark.skip(reason="Skipping until add_latency is implemented")
    def test_create_private_groups_with_latency(self):
        with self.add_latency():
            self.test_create_private_group_baseline()

    @pytest.mark.dependency(depends=["test_create_private_group_baseline"])
    @pytest.mark.skip(reason="Skipping until add_packet_loss is implemented")
    def test_create_private_groups_with_packet_loss(self):
        with self.add_packet_loss():
            self.test_create_private_group_baseline()

    @pytest.mark.dependency(depends=["test_create_private_group_baseline"])
    @pytest.mark.skip(reason="Skipping until add_low_bandwith is implemented")
    def test_create_private_groups_with_low_bandwidth(self):
        with self.add_low_bandwith():
            self.test_create_private_group_baseline()

    @pytest.mark.dependency(depends=["test_create_private_group_baseline"])
    @pytest.mark.skip(reason="Skipping until node_pause is implemented")
    def test_create_private_groups_with_node_pause_30_seconds(self):
        pk_receiver = self.receiver.get_pubkey(DEFAULT_DISPLAY_NAME)

        response = self.sender.send_contact_request(pk_receiver, "contact_request")
        chat_id = self.get_message_id(response)
        self.receiver.accept_contact_request(chat_id)

        with self.node_pause(self.receiver):
            private_group_name = f"private_group_{uuid4()}"
            self.sender.create_group_chat_with_members([pk_receiver], private_group_name)
            sleep(30)
        self.receiver.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=private_group_name)

    
