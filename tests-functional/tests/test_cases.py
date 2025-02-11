from contextlib import contextmanager
import json
import logging
import random
import string
import threading
import time
from collections import namedtuple
from uuid import uuid4

import pytest

from clients.services.wallet import WalletService
from clients.signals import SignalClient, SignalType
from clients.status_backend import RpcClient, StatusBackend
from conftest import option
from resources.constants import user_1, user_2
from resources.enums import MessageContentType


class StatusDTestCase:
    network_id = 31337

    def setup_method(self):
        self.rpc_client = RpcClient(option.rpc_url_statusd)


class StatusBackendTestCase:

    reuse_container = True  # Skip close_status_backend_containers cleanup
    await_signals = [SignalType.NODE_LOGIN.value]

    network_id = 31337

    def setup_class(self):
        self.rpc_client = StatusBackend(await_signals=self.await_signals)
        self.wallet_service = WalletService(self.rpc_client)

        self.rpc_client.init_status_backend()
        self.rpc_client.restore_account_and_login()
        self.rpc_client.wait_for_login()

    def teardown_class(self):
        for container in option.status_backend_containers:
            container.kill()


class WalletTestCase(StatusBackendTestCase):

    def wallet_create_multi_transaction(self, **kwargs):
        method = "wallet_createMultiTransaction"
        transfer_tx_data = {
            "data": "",
            "from": user_1.address,
            "gas": "0x5BBF",
            "input": "",
            "maxFeePerGas": "0xbcc0f04fd",
            "maxPriorityFeePerGas": "0xbcc0f04fd",
            "to": user_2.address,
            "type": "0x02",
            "value": "0x5af3107a4000",
        }
        for key, new_value in kwargs.items():
            if key in transfer_tx_data:
                transfer_tx_data[key] = new_value
            else:
                logging.info(f"Warning: The key '{key}' does not exist in the transferTx parameters and will be ignored.")
        params = [
            {
                "fromAddress": user_1.address,
                "fromAmount": "0x5af3107a4000",
                "fromAsset": "ETH",
                "type": 0,  # MultiTransactionSend
                "toAddress": user_2.address,
                "toAsset": "ETH",
            },
            [
                {
                    "bridgeName": "Transfer",
                    "chainID": 31337,
                    "transferTx": transfer_tx_data,
                }
            ],
            f"{option.password}",
        ]
        return self.rpc_client.rpc_request(method, params)

    def send_valid_multi_transaction(self, **kwargs):
        response = self.wallet_create_multi_transaction(**kwargs)

        tx_hash = None
        self.rpc_client.verify_is_valid_json_rpc_response(response)
        try:
            tx_hash = response.json()["result"]["hashes"][str(self.network_id)][0]
        except (KeyError, json.JSONDecodeError):
            raise Exception(response.content)
        return tx_hash


class TransactionTestCase(WalletTestCase):

    def setup_method(self):
        self.tx_hash = self.send_valid_multi_transaction()


class EthRpcTestCase(WalletTestCase):

    @pytest.fixture(autouse=True, scope="class")
    def tx_data(self):
        tx_hash = self.send_valid_multi_transaction()
        self.wait_until_tx_not_pending(tx_hash)

        receipt = self.get_transaction_receipt(tx_hash)
        try:
            block_number = receipt.json()["result"]["blockNumber"]
            block_hash = receipt.json()["result"]["blockHash"]
        except (KeyError, json.JSONDecodeError):
            raise Exception(receipt.content)

        tx_data = namedtuple("TxData", ["tx_hash", "block_number", "block_hash"])
        return tx_data(tx_hash, block_number, block_hash)

    def get_block_header(self, block_number):
        method = "ethclient_headerByNumber"
        params = [self.network_id, block_number]
        return self.rpc_client.rpc_valid_request(method, params)

    def get_transaction_receipt(self, tx_hash):
        method = "ethclient_transactionReceipt"
        params = [self.network_id, tx_hash]
        return self.rpc_client.rpc_valid_request(method, params)

    def wait_until_tx_not_pending(self, tx_hash, timeout=10):
        method = "ethclient_transactionByHash"
        params = [self.network_id, tx_hash]
        response = self.rpc_client.rpc_valid_request(method, params)

        start_time = time.time()
        while response.json()["result"]["isPending"] is True:
            time_passed = time.time() - start_time
            if time_passed >= timeout:
                raise TimeoutError(f"Tx {tx_hash} is still pending after {timeout} seconds")
            time.sleep(0.5)
            response = self.rpc_client.rpc_valid_request(method, params)
        return response.json()["result"]["tx"]


class SignalTestCase(StatusDTestCase):
    await_signals = []

    def setup_method(self):
        super().setup_method()
        self.signal_client = SignalClient(option.ws_url_statusd, self.await_signals)

        websocket_thread = threading.Thread(target=self.signal_client._connect)
        websocket_thread.daemon = True
        websocket_thread.start()


class NetworkConditionTestCase:

    @contextmanager
    def add_latency(self, node, latency=300, jitter=50):
        logging.info("Entering context manager: add_latency")
        node.container_exec(f"apk add iproute2 && tc qdisc add dev eth0 root netem delay {latency}ms {jitter}ms distribution normal")
        try:
            yield
        finally:
            logging.info("Exiting context manager: add_latency")
            node.container_exec("tc qdisc del dev eth0 root")

    @contextmanager
    def add_packet_loss(self, node, packet_loss=2):
        logging.info("Entering context manager: add_packet_loss")
        node.container_exec(f"apk add iproute2 && tc qdisc add dev eth0 root netem loss {packet_loss}%")
        try:
            yield
        finally:
            logging.info("Exiting context manager: add_packet_loss")
            node.container_exec("tc qdisc del dev eth0 root netem")

    @contextmanager
    def add_low_bandwith(self, node, rate="1mbit", burst="32kbit", limit="12500"):
        logging.info("Entering context manager: add_low_bandwith")
        node.container_exec(f"apk add iproute2 && tc qdisc add dev eth0 root tbf rate {rate} burst {burst} limit {limit}")
        try:
            yield
        finally:
            logging.info("Exiting context manager: add_low_bandwith")
            node.container_exec("tc qdisc del dev eth0 root")

    @contextmanager
    def node_pause(self, node):
        logging.info("Entering context manager: node_pause")
        node.container_pause()
        try:
            yield
        finally:
            logging.info("Exiting context manager: node_pause")
            node.container_unpause()


class MessengerTestCase(NetworkConditionTestCase):

    await_signals = [
        SignalType.MESSAGES_NEW.value,
        SignalType.MESSAGE_DELIVERED.value,
        SignalType.NODE_LOGIN.value,
        SignalType.NODE_LOGOUT.value,
    ]

    @pytest.fixture(scope="function", autouse=False)
    def setup_two_nodes(self, request):
        request.cls.sender = self.sender = self.initialize_backend(self.await_signals, True)
        request.cls.receiver = self.receiver = self.initialize_backend(self.await_signals, True)

    def initialize_backend(self, await_signals, privileged=True):
        backend = StatusBackend(await_signals, privileged)
        backend.init_status_backend()
        backend.create_account_and_login()
        backend.find_public_key()
        backend.wakuext_service.start_messenger()
        return backend

    def make_contacts(self):
        existing_contacts = self.receiver.wakuext_service.get_contacts()

        if self.sender.public_key in str(existing_contacts):
            return

        response = self.sender.wakuext_service.send_contact_request(self.receiver.public_key, "contact_request")
        expected_message = self.get_message_by_content_type(response, content_type=MessageContentType.CONTACT_REQUEST.value)[0]
        self.receiver.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=expected_message.get("id"))
        self.receiver.wakuext_service.accept_contact_request(expected_message.get("id"))
        accepted_signal = f"@{self.receiver.public_key} accepted your contact request"
        self.sender.find_signal_containing_pattern(SignalType.MESSAGES_NEW.value, event_pattern=accepted_signal)

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

    def create_and_join_community(self):
        name = f"vac_qa_community_{''.join(random.choices(string.ascii_letters, k=10))}"
        response = self.sender.wakuext_service.create_community(name)

        community_id = response.get("result", {}).get("communities", [{}])[0].get("id")
        self.receiver.wakuext_service.fetch_community(community_id)

        response_to_join = self.receiver.wakuext_service.request_to_join_community(community_id)
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
        return community_id + chat_id

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
        sent_messages = []
        for i in range(message_count):
            message_text = f"test_message_{i+1}_{uuid4()}"
            response = self.sender.wakuext_service.send_message(self.receiver.public_key, message_text)
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
