import os
import asyncio
import random
import shutil
import signal
import string
import subprocess
import re
import threading
import time
import requests
from tenacity import retry, stop_after_delay, wait_fixed, stop_after_attempt
from src.data_storage import DS
from src.libs.custom_logger import get_custom_logger
from src.node.rpc_client import StatusNodeRPC
from tests.clients.signals import SignalClient

logger = get_custom_logger(__name__)


class StatusNode:
    def __init__(self, name=None, port=None, pubkey=None):
        try:
            os.remove(f"{name}.log")
        except:
            pass
        self.name = self.random_node_name() if not name else name.lower()
        self.port = str(random.randint(1024, 65535)) if not port else port
        self.pubkey = pubkey
        self.process = None
        self.log_thread = None
        self.capture_logs = True
        self.logs = []
        self.pid = None
        self.signal_client = None
        self.api = StatusNodeRPC(self.port, self.name)

    def initialize_node(self, name, port, data_dir, account_data):
        """Centralized method to initialize a node."""
        self.name = name
        self.port = port
        self.start(data_dir)
        self.wait_fully_started()
        self.create_account_and_login(account_data)
        self.start_messenger()

    def start_node(self, command):
        """Start the node using a subprocess command."""
        logger.info(f"Starting node with command: {command}")
        self.process = subprocess.Popen(command, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
        self.pid = self.process.pid
        self.log_thread = self.capture_process_logs(self.process, self.logs)

    def start(self, data_dir, capture_logs=True):
        """Start the status-backend node and initialize it before subscribing to signals."""
        self.capture_logs = capture_logs
        command = ["./status-backend", f"--address=localhost:{self.port}"]
        self.start_node(command)
        self.wait_fully_started()
        self.api.initialize_application(data_dir)
        self.api = StatusNodeRPC(self.port, self.name)
        self.start_signal_client()

    def create_account_and_login(self, account_data):
        """Create an account and log in using the status-backend."""
        logger.info(f"Creating account and logging in for node {self.name}")
        self.api.create_account_and_login(account_data)

    def start_messenger(self):
        """Start the Waku messenger."""
        logger.info(f"Starting Waku messenger for node {self.name}")
        self.api.start_messenger()

    def start_signal_client(self):
        """Start a SignalClient for the given node to listen for WebSocket signals."""
        ws_url = f"ws://localhost:{self.port}"
        await_signals = ["community.chatMessage", "mediaserver.started"]
        self.signal_client = SignalClient(ws_url, await_signals)

        websocket_thread = threading.Thread(target=self.signal_client._connect)
        websocket_thread.daemon = True
        websocket_thread.start()
        logger.info("WebSocket client started and subscribed to signals.")

    def wait_fully_started(self):
        """Wait until the node logs indicate that the server has started."""
        logger.info(f"Waiting for {self.name} to fully start...")
        start_time = time.time()
        while time.time() - start_time < 20:
            if any("server started" in log for log in self.logs):
                logger.info(f"Node {self.name} has fully started.")
                return
            time.sleep(0.5)
        raise TimeoutError(f"Node {self.name} did not fully start in time.")

    def capture_process_logs(self, process, logs):
        """Capture logs from a subprocess."""

        def read_output():
            while True:
                line = process.stdout.readline()
                if not line:
                    break
                logs.append(line.strip())
                logger.debug(f"{self.name.upper()} - {line.strip()}")

        log_thread = threading.Thread(target=read_output)
        log_thread.daemon = True
        log_thread.start()
        return log_thread

    def random_node_name(self, length=10):
        """Generate a random node name."""
        allowed_chars = string.ascii_lowercase + string.digits + "_-"
        return ''.join(random.choice(allowed_chars) for _ in range(length))

    def get_pubkey(self):
        """Retrieve the public key of the node."""
        if self.pubkey:
            return self.pubkey
        else:
            raise Exception(f"Public key not set for node {self.name}")

    def wait_for_signal(self, signal_type, timeout=20):
        """Wait for a signal using the signal client."""
        return self.signal_client.wait_for_signal(signal_type, timeout)

    def stop(self, remove_local_data=True):
        """Stop the status-backend process."""
        if self.process:
            logger.info(f"Stopping node with name: {self.name}")
            self.process.kill()
            if self.capture_logs:
                self.log_thread.join()
            if remove_local_data:
                node_dir = f"test-{self.name}"
                if os.path.exists(node_dir):
                    try:
                        shutil.rmtree(node_dir)
                    except Exception as ex:
                        logger.warning(f"Couldn't delete node dir {node_dir} because of {str(ex)}")
            self.process = None

    @retry(stop=stop_after_delay(30), wait=wait_fixed(0.1), reraise=True)
    # wakuext_fetchCommunity times out sometimes so that's why we need this retry mechanism
    def fetch_community(self, community_key):
        params = [{"communityKey": community_key, "waitForResponse": True, "tryDatabase": True}]
        return self.api.send_rpc_request("wakuext_fetchCommunity", params, timeout=10)

    def request_to_join_community(self, community_id):
        print("request_to_join_community: ", community_id, self.name, self.api)
        params = [{"communityId": community_id, "addressesToReveal": ["fakeaddress"], "airdropAddress": "fakeaddress"}]
        return self.api.send_rpc_request("wakuext_requestToJoinCommunity", params)

    def accept_request_to_join_community(self, request_to_join_id):
        print("accept_request_to_join_community: ", request_to_join_id, self.name, self.api)
        self._ensure_api_initialized()
        params = [{"id": request_to_join_id}]
        return self.api.send_rpc_request("wakuext_acceptRequestToJoinCommunity", params)

    def _ensure_api_initialized(self):
        if not self.api:
            logger.warning(f"API client is not initialized for node {self.name}. Reinitializing...")
            self.api = StatusNodeRPC(self.port, self.name)
            if not self.api:
                raise Exception(f"Failed to initialize the RPC client for node {self.name}")

    def send_community_chat_message(self, chat_id, message):
        params = [{"chatId": chat_id, "text": message, "contentType": 1}]
        return self.api.send_rpc_request("wakuext_sendChatMessage", params)

    def leave_community(self, community_id):
        params = [community_id]
        return self.api.send_rpc_request("wakuext_leaveCommunity", params)

    def send_contact_request(self, pubkey, message):
        params = [{"id": pubkey, "message": message}]
        return self.api.send_rpc_request("wakuext_sendContactRequest", params)

    async def wait_for_logs_async(self, strings=None, timeout=10):
        if not isinstance(strings, list):
            raise ValueError("strings must be a list")
        start_time = time.time()
        while time.time() - start_time < timeout:
            all_found = True
            for string in strings:
                logs = self.search_logs(string=string)
                if not logs:
                    all_found = False
                    break
            if all_found:
                return True
            await asyncio.sleep(0.5)
        return False

    def pause_process(self):
        if self.pid:
            logger.info(f"Pausing node with pid: {self.pid}")
            os.kill(self.pid, signal.SIGTSTP)

    def resume_process(self):
        if self.pid:
            logger.info(f"Resuming node with pid: {self.pid}")
            os.kill(self.pid, signal.SIGCONT)
