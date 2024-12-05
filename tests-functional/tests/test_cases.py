from contextlib import contextmanager
import json
import logging
import threading
import time
from collections import namedtuple

import pytest

from clients.signals import SignalClient, SignalType
from clients.status_backend import RpcClient, StatusBackend
from conftest import option
from constants import user_1, user_2, DEFAULT_DISPLAY_NAME


class StatusDTestCase:
    network_id = 31337

    def setup_method(self):
        self.rpc_client = RpcClient(
            option.rpc_url_statusd
        )


class StatusBackendTestCase:

    await_signals = [
        SignalType.NODE_LOGIN.value
    ]

    def setup_class(self):
        self.rpc_client = StatusBackend(await_signals=self.await_signals)

        self.rpc_client.init_status_backend()
        self.rpc_client.restore_account_and_login()
        self.rpc_client.wait_for_login()

        self.network_id = 31337


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
                logging.info(
                    f"Warning: The key '{key}' does not exist in the transferTx parameters and will be ignored.")
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
                    "transferTx": transfer_tx_data
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
            tx_hash = response.json(
            )["result"]["hashes"][str(self.network_id)][0]
        except (KeyError, json.JSONDecodeError):
            raise Exception(response.content)
        return tx_hash


class TransactionTestCase(WalletTestCase):

    def setup_method(self):
        self.tx_hash = self.send_valid_multi_transaction()


class EthRpcTestCase(WalletTestCase):

    @pytest.fixture(autouse=True, scope='class')
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
        while response.json()["result"]["isPending"] == True:
            time_passed = time.time() - start_time
            if time_passed >= timeout:
                raise TimeoutError(
                    f"Tx {tx_hash} is still pending after {timeout} seconds")
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
    def add_latency(self):
        pass
        #TODO: To be implemented when we have docker exec capability

    @contextmanager
    def add_packet_loss(self):
        pass
        #TODO: To be implemented when we have docker exec capability

    @contextmanager
    def add_low_bandwith(self):
        pass
        #TODO: To be implemented when we have docker exec capability

    @contextmanager
    def node_pause(self, node):
        pass
        #TODO: To be implemented when we have docker exec capability

class OneToOneMessageTestCase(NetworkConditionTestCase):

    def initialize_backend(self, await_signals, display_name=DEFAULT_DISPLAY_NAME, url=None):
        backend = StatusBackend(await_signals=await_signals, url=url)
        backend.init_status_backend()
        backend.create_account_and_login(display_name=display_name)
        backend.start_messenger()
        return backend


    def validate_event_against_response(self, event, fields_to_validate, response):
        messages_in_event = event["event"]["messages"]
        assert len(messages_in_event) > 0, "No messages found in the event"
        response_chat = response["result"]["chats"][0]

        message_id = response_chat["lastMessage"]["id"]
        message = next((message for message in messages_in_event if message["id"] == message_id), None)
        assert message, f"Message with ID {message_id} not found in the event"

        message_mismatch = []
        for response_field, event_field in fields_to_validate.items():
            response_value = response_chat["lastMessage"][response_field]
            event_value = message[event_field]
            if response_value != event_value:
                message_mismatch.append(
                    f"Field '{response_field}': Expected '{response_value}', Found '{event_value}'"
                )

        if not message_mismatch:
            return

        raise AssertionError(
            "Some Sender RPC responses are not matching the signals received by the receiver.\n"
            "Details of mismatches:\n" +
            "\n".join(message_mismatch)
        )
