import pytest

from clients.signals import SignalType
from resources.enums import MessageContentType
from steps.messenger import MessengerSteps


@pytest.mark.usefixtures("setup_two_unprivileged_nodes")
@pytest.mark.rpc
class TestTransactionsChatMessages(MessengerSteps):
    REQUEST_TRANSACTION_TEXT = "Request transaction"
    REQUEST_TRANSACTION_DECLINED_TEXT = "Transaction request declined"

    def test_request_transaction(self):
        self.make_contacts()
        value = "1"
        contract = "TEST_CONTRACT"
        address = "TEST_ADDRESS"
        response = self.sender.wakuext_service.request_transaction(self.receiver.public_key, value, contract, address)
        self.receiver.verify_json_schema(response, method="wakuext_requestTransaction")

        self.receiver.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=self.REQUEST_TRANSACTION_TEXT, timeout=5)

        response = self.receiver.wakuext_service.chat_messages(self.sender.public_key)
        message = self.get_message_by_content_type(response, content_type=MessageContentType.TRANSACTION_COMMAND.value)[0]
        assert message.get("text", "") == self.REQUEST_TRANSACTION_TEXT
        command_parameters = message.get("commandParameters", {})
        assert command_parameters.get("value", "") == value
        assert command_parameters.get("contract", "") == contract
        assert command_parameters.get("address", "") == address

    def test_decline_request_transaction(self):
        self.make_contacts()
        sender_chat_id = self.receiver.public_key
        value = "1"
        contract = "TEST_CONTRACT"
        address = "TEST_ADDRESS"
        response = self.sender.wakuext_service.request_transaction(sender_chat_id, value, contract, address)
        message_id = response.get("result", {}).get("messages", [])[0].get("id", "")

        self.receiver.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=self.REQUEST_TRANSACTION_TEXT, timeout=5)

        response = self.receiver.wakuext_service.decline_request_transaction(message_id)
        self.receiver.verify_json_schema(response, method="wakuext_requestTransaction")  # same schema

        self.sender.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=self.REQUEST_TRANSACTION_DECLINED_TEXT, timeout=5)

        response = self.sender.wakuext_service.chat_messages(sender_chat_id)
        message = self.get_message_by_content_type(response, content_type=MessageContentType.TRANSACTION_COMMAND.value)[0]
        assert message.get("text", "") == self.REQUEST_TRANSACTION_DECLINED_TEXT
        command_parameters = message.get("commandParameters", {})
        assert command_parameters.get("value", "") == value
        assert command_parameters.get("contract", "") == contract
        assert command_parameters.get("address", "") == address
