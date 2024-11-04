from contextlib import contextmanager
import inspect
import subprocess
import pytest

from src.libs.common import delay
from src.libs.custom_logger import get_custom_logger
from src.node.status_node import StatusNode
from datetime import datetime
from constants import *

logger = get_custom_logger(__name__)


class StepsCommon:
    @pytest.fixture(scope="function", autouse=False)
    def start_2_nodes(self):
        logger.debug(f"Running fixture setup: {inspect.currentframe().f_code.co_name}")

        account_data_first = {
            **ACCOUNT_PAYLOAD_DEFAULTS,
            "rootDataDir": LOCAL_DATA_DIR1,
            "displayName": "first_node_user"
        }
        account_data_second = {
            **ACCOUNT_PAYLOAD_DEFAULTS,
            "rootDataDir": LOCAL_DATA_DIR2,
            "displayName": "second_node_user"
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

        self.first_node_pubkey = self.first_node.get_pubkey("first_node_user")
        self.second_node_pubkey = self.second_node.get_pubkey("second_node_user")

        logger.debug(f"First Node Public Key: {self.first_node_pubkey}")
        logger.debug(f"Second Node Public Key: {self.second_node_pubkey}")

    @contextmanager
    def add_latency(self):
        logger.debug("Adding network latency")
        subprocess.Popen(LATENCY_CMD, shell=True)
        try:
            yield
        finally:
            logger.debug("Removing network latency")
            subprocess.Popen(REMOVE_TC_CMD, shell=True)

    @contextmanager
    def add_packet_loss(self):
        logger.debug("Adding packet loss")
        subprocess.Popen(PACKET_LOSS_CMD, shell=True)
        try:
            yield
        finally:
            logger.debug("Removing packet loss")
            subprocess.Popen(REMOVE_TC_CMD, shell=True)

    @contextmanager
    def add_low_bandwidth(self):
        logger.debug("Adding low bandwidth")
        subprocess.Popen(LOW_BANDWIDTH_CMD, shell=True)
        try:
            yield
        finally:
            logger.debug("Removing low bandwidth")
            subprocess.Popen(REMOVE_TC_CMD, shell=True)

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
        response_messages = response.get("result", {}).get("messages", [])
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

    def create_group_chat_with_timestamp(self, sender_node, member_list, private_group_name):
        timestamp = datetime.now().strftime("%H:%M:%S")
        response = sender_node.create_group_chat_with_members(member_list, private_group_name)
        response_messages = response["result"]["messages"]
        message_id = None
        for m in response_messages:
            if private_group_name in m["text"]:
                message_id = m["id"]
                break
        return timestamp, message_id
