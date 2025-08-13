import base64
import pytest

from clients.signals import SignalType
from resources.enums import MessageContentType
from steps.messenger import MessengerSteps


@pytest.mark.rpc
@pytest.mark.parametrize("waku_light_client", [False, True], indirect=True, ids=["wakuV2LightClient_False", "wakuV2LightClient_True"])
class TestTransactionsChatMessages(MessengerSteps):
    REQUEST_TRANSACTION_TEXT = "Request transaction"
    REQUEST_TRANSACTION_DECLINED_TEXT = "Transaction request declined"
    REQUEST_ADDRESS_FOR_TRANSACTION_TEXT = "Request address for transaction"
    REQUEST_ADDRESS_FOR_TRANSACTION_DECLINED_TEXT = "Request address for transaction declined"
    REQUEST_ADDRESS_FOR_TRANSACTION_ACCEPTED_TEXT = "Request address for transaction accepted"
    TRANSACTION_SENT_TEXT = "Transaction sent"

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile, waku_light_client):
        """Initialize two backends (sender and receiver) for each test function"""
        self.sender = backend_new_profile("sender", waku_light_client=waku_light_client)
        self.receiver = backend_new_profile("receiver", waku_light_client=waku_light_client)

    @pytest.fixture
    def transaction_data(self):
        return {
            "value": "10000000",
            "contract": "0xCONTRACT",
            "address": "0xADDRESS",
            "from": "0xFROM",
            "tx_hash": "0xTXHASH",
            "signature": "0xa123",
        }

    def assert_transaction_command_response(self, response, expected_text: str, parameters_to_assert: list[tuple[str, str]]):
        message_id = self.get_message_id(response)
        message = self.get_message_by_message_id(response, message_id)
        assert message.get("text", "") == expected_text
        assert message.get("contentType", -1) == MessageContentType.TRANSACTION_COMMAND.value
        command_parameters = message.get("commandParameters", {})

        for parameter, expected_value in parameters_to_assert:
            print(parameter, expected_value)
            assert command_parameters.get(parameter, "") == expected_value

    def test_request_transaction(self, transaction_data):
        self.make_contacts(sender=self.sender, receiver=self.receiver)
        response = self.sender.wakuext_service.request_transaction(
            self.receiver.public_key, transaction_data["value"], transaction_data["contract"], transaction_data["address"]
        )
        # TODO: Add more assertions on response

        self.assert_transaction_command_response(
            response,
            self.REQUEST_TRANSACTION_TEXT,
            [("value", transaction_data["value"]), ("contract", transaction_data["contract"]), ("address", transaction_data["address"])],
        )

    def test_decline_request_transaction(self, transaction_data):
        self.make_contacts(sender=self.sender, receiver=self.receiver)
        sender_chat_id = self.receiver.public_key
        response = self.sender.wakuext_service.request_transaction(
            sender_chat_id, transaction_data["value"], transaction_data["contract"], transaction_data["address"]
        )
        message_id = self.get_message_id(response)

        self.receiver.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=self.REQUEST_TRANSACTION_TEXT, timeout=5)

        response = self.receiver.wakuext_service.decline_request_transaction(message_id)
        # TODO: Add more assertions on response

        self.assert_transaction_command_response(
            response,
            self.REQUEST_TRANSACTION_DECLINED_TEXT,
            [("value", transaction_data["value"]), ("contract", transaction_data["contract"]), ("address", transaction_data["address"])],
        )

    def test_accept_request_transaction(self, transaction_data):
        self.make_contacts(sender=self.sender, receiver=self.receiver)
        sender_chat_id = self.receiver.public_key
        response = self.sender.wakuext_service.request_transaction(
            sender_chat_id, transaction_data["value"], transaction_data["contract"], transaction_data["address"]
        )
        message_id = self.get_message_id(response)

        self.receiver.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=self.REQUEST_TRANSACTION_TEXT, timeout=5)

        response = self.receiver.wakuext_service.accept_request_transaction(transaction_data["tx_hash"], message_id, transaction_data["signature"])
        # TODO: Add more assertions on response

        self.assert_transaction_command_response(
            response,
            self.TRANSACTION_SENT_TEXT,
            [
                ("value", transaction_data["value"]),
                ("contract", transaction_data["contract"]),
                ("address", transaction_data["address"]),
                ("transactionHash", transaction_data["tx_hash"]),
                ("signature", base64.b64encode(bytes.fromhex(transaction_data["signature"].replace("0x", ""))).decode()),
            ],
        )

    def test_request_address_for_transaction(self, transaction_data):
        self.make_contacts(sender=self.sender, receiver=self.receiver)
        response = self.sender.wakuext_service.request_address_for_transaction(
            self.receiver.public_key, transaction_data["from"], transaction_data["value"], transaction_data["contract"]
        )
        # TODO: Add more assertions on response

        self.assert_transaction_command_response(
            response,
            self.REQUEST_ADDRESS_FOR_TRANSACTION_TEXT,
            [("value", transaction_data["value"]), ("contract", transaction_data["contract"]), ("from", transaction_data["from"])],
        )

    def test_decline_request_address_for_transaction(self, transaction_data):
        self.make_contacts(sender=self.sender, receiver=self.receiver)
        sender_chat_id = self.receiver.public_key
        response = self.sender.wakuext_service.request_address_for_transaction(
            sender_chat_id, transaction_data["from"], transaction_data["value"], transaction_data["contract"]
        )
        message_id = self.get_message_id(response)

        self.receiver.find_signal_containing_pattern(
            SignalType.MESSAGES_NEW.value, event_pattern=self.REQUEST_ADDRESS_FOR_TRANSACTION_TEXT, timeout=5
        )

        response = self.receiver.wakuext_service.decline_request_address_for_transaction(message_id)
        # TODO: Add more assertions on response

        self.assert_transaction_command_response(
            response,
            self.REQUEST_ADDRESS_FOR_TRANSACTION_DECLINED_TEXT,
            [("value", transaction_data["value"]), ("contract", transaction_data["contract"])],
        )

    def test_accept_request_address_for_transaction(self, transaction_data):
        self.make_contacts(sender=self.sender, receiver=self.receiver)
        sender_chat_id = self.receiver.public_key
        response = self.sender.wakuext_service.request_address_for_transaction(
            sender_chat_id, transaction_data["from"], transaction_data["value"], transaction_data["contract"]
        )
        message_id = self.get_message_id(response)

        self.receiver.find_signal_containing_pattern(
            SignalType.MESSAGES_NEW.value, event_pattern=self.REQUEST_ADDRESS_FOR_TRANSACTION_TEXT, timeout=5
        )

        response = self.receiver.wakuext_service.accept_request_address_for_transaction(message_id, transaction_data["address"])
        # TODO: Add more assertions on response

        self.assert_transaction_command_response(
            response,
            self.REQUEST_ADDRESS_FOR_TRANSACTION_ACCEPTED_TEXT,
            [("value", transaction_data["value"]), ("contract", transaction_data["contract"]), ("address", transaction_data["address"])],
        )

    def test_send_transaction(self, transaction_data):
        self.make_contacts(sender=self.sender, receiver=self.receiver)
        sender_chat_id = self.receiver.public_key
        response = self.sender.wakuext_service.send_transaction(
            sender_chat_id, transaction_data["value"], transaction_data["contract"], transaction_data["tx_hash"], transaction_data["signature"]
        )
        # TODO: Add more assertions on response

        self.assert_transaction_command_response(
            response,
            self.TRANSACTION_SENT_TEXT,
            [
                ("value", transaction_data["value"]),
                ("contract", transaction_data["contract"]),
                ("transactionHash", transaction_data["tx_hash"]),
                ("signature", base64.b64encode(bytes.fromhex(transaction_data["signature"].replace("0x", ""))).decode()),
            ],
        )
