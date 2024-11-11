from contextlib import contextmanager
import inspect
import subprocess
import pytest
import json
import threading
import time
from collections import namedtuple
from src.libs.common import delay
from src.libs.custom_logger import get_custom_logger
from src.node.status_node import StatusNode
from datetime import datetime
from src.constants import *
from src.node.clients.signals import SignalClient
from src.node.clients.status_backend import RpcClient, StatusBackend
from conftest import option

logger = get_custom_logger(__name__)


class StatusDTestCase:
    network_id = 31337

    def setup_method(self):
        self.rpc_client = RpcClient(option.rpc_url_statusd)


class StatusBackendTestCase:
    def setup_class(self):
        self.rpc_client = StatusBackend()
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
                logger.info(f"Warning: The key '{key}' does not exist in the transferTx parameters and will be ignored.")
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
        while response.json()["result"]["isPending"] == True:
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


class StepsCommon:
    @pytest.fixture(scope="function", autouse=False)
    def start_2_nodes(self):
        logger.debug(f"Running fixture setup: {inspect.currentframe().f_code.co_name}")
        self.first_node_display_name = "first_node_user"
        self.second_node_display_name = "second_node_user"

        account_data_first = {
            **ACCOUNT_PAYLOAD_DEFAULTS,
            "rootDataDir": LOCAL_DATA_DIR1,
            "displayName": self.first_node_display_name,
        }
        account_data_second = {
            **ACCOUNT_PAYLOAD_DEFAULTS,
            "rootDataDir": LOCAL_DATA_DIR2,
            "displayName": self.second_node_display_name,
        }

        self.first_node = StatusNode(name="first_node")
        self.first_node.start(data_dir=LOCAL_DATA_DIR1)
        self.first_node.wait_fully_started()

        self.second_node = StatusNode(name="second_node")
        self.second_node.start(data_dir=LOCAL_DATA_DIR2)
        self.second_node.wait_fully_started()

        self.first_node.create_account_and_login(account_data_first)
        self.second_node.create_account_and_login(account_data_second)

        delay(4)
        self.first_node.start_messenger()
        delay(1)
        self.second_node.start_messenger()

        self.first_node_pubkey = self.first_node.get_pubkey(self.first_node_display_name)
        self.second_node_pubkey = self.second_node.get_pubkey(self.second_node_display_name)

        logger.debug(f"First Node Public Key: {self.first_node_pubkey}")
        logger.debug(f"Second Node Public Key: {self.second_node_pubkey}")

    @contextmanager
    def add_latency(self):
        logger.debug("Entering context manager: add_latency")
        subprocess.Popen(
            "sudo tc qdisc add dev eth0 root netem delay 1s 100ms distribution normal",
            shell=True,
        )
        try:
            yield
        finally:
            logger.debug(f"Exiting context manager: add_latency")
            subprocess.Popen("sudo tc qdisc del dev eth0 root", shell=True)

    @contextmanager
    def add_packet_loss(self):
        logger.debug("Entering context manager: add_packet_loss")
        subprocess.Popen("sudo tc qdisc add dev eth0 root netem loss 50%", shell=True)
        try:
            yield
        finally:
            logger.debug(f"Exiting context manager: add_packet_loss")
            subprocess.Popen("sudo tc qdisc del dev eth0 root netem", shell=True)

    @contextmanager
    def add_low_bandwith(self):
        logger.debug("Entering context manager: add_low_bandwith")
        subprocess.Popen("sudo tc qdisc add dev eth0 root tbf rate 1kbit burst 1kbit", shell=True)
        try:
            yield
        finally:
            logger.debug(f"Exiting context manager: add_low_bandwith")
            subprocess.Popen("sudo tc qdisc del dev eth0 root", shell=True)

    @contextmanager
    def node_pause(self, node):
        logger.debug("Entering context manager: node_pause")
        node.pause_process()
        try:
            yield
        finally:
            logger.debug(f"Exiting context manager: node_pause")
            node.resume_process()

    def send_with_timestamp(self, send_method, id, message):
        timestamp = datetime.now().strftime("%H:%M:%S")
        response = send_method(id, message)
        response_messages = response.json().get("result", {}).get("messages", [])
        message_id = None

        for m in response_messages:
            if m["text"] == message:
                message_id = m["id"]
                break

        return timestamp, message_id, response

    def accept_contact_request(self, sending_node=None, receiving_node_pk=None):
        if not sending_node:
            sending_node = self.second_node
        if not receiving_node_pk:
            receiving_node_pk = self.first_node_pubkey
        sending_node.send_contact_request(receiving_node_pk, "hi")
