import pytest

from clients.signals import SignalType
from resources.enums import MessageContentType
from steps.messenger import MessengerSteps


@pytest.mark.usefixtures("setup_two_unprivileged_nodes")
@pytest.mark.rpc
class TestTransactionsChatMessages(MessengerSteps):
    def test_request_transaction(self):
        self.make_contacts()
        sender_chat_id = self.receiver.public_key
        value = "1"
        contract = "TEST_CONTRACT"
        address = "TEST_ADDRESS"
        response = self.sender.wakuext_service.request_transaction(sender_chat_id, value, contract, address)
        self.receiver.verify_json_schema(response, method="wakuext_requestTransaction")

        self.receiver.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern="Request transaction", timeout=5)
        receiver_chat_id = self.sender.public_key
        response = self.receiver.wakuext_service.chat_messages(receiver_chat_id)

        message = self.get_message_by_content_type(response, content_type=MessageContentType.TRANSACTION_COMMAND.value)[0]
        command_parameters = message.get("commandParameters", {})
        assert command_parameters.get("value", "") == value
        assert command_parameters.get("contract", "") == contract
        assert command_parameters.get("address", "") == address
