import os
import random
import shutil
import signal
import string
import subprocess
import threading
import time

from src.libs.custom_logger import get_custom_logger
from src.node.rpc_client import StatusNodeRPC
from clients.signals import SignalClient
from src.libs.common import build_and_copy_binary
from pathlib import Path

logger = get_custom_logger(__name__)

PROJECT_ROOT = Path(__file__).resolve().parents[2]


class StatusNode:
    binary_built = False

    def __init__(self, name=None, port=None, pubkey=None):
        self.data_dir = None
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
        self.last_response = None
        self.api = StatusNodeRPC(self.port, self.name)

    def initialize_node(self, name, port, data_dir, account_data):
        self.name = name
        self.port = port
        self.start(data_dir)
        self.wait_fully_started()
        self.create_account_and_login(account_data)
        self.start_messenger()
        self.pubkey = self.get_pubkey(account_data["displayName"])

    def start_node(self, command):
        logger.info(f"Starting node with command: {command}")
        self.process = subprocess.Popen(command, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
        self.pid = self.process.pid
        self.log_thread = self.capture_process_logs(self.process, self.logs)

    def start(self, data_dir, capture_logs=True):
        dest_binary_path = Path(PROJECT_ROOT) / "status-backend"

        if not StatusNode.binary_built and not dest_binary_path.exists():
            if not build_and_copy_binary():
                raise RuntimeError("Failed to build or copy the status-backend binary.")
            StatusNode.binary_built = True

        self.capture_logs = capture_logs
        self.data_dir = data_dir
        command = ["./status-backend", f"--address=localhost:{self.port}"]
        self.start_node(command)
        self.wait_fully_started()
        self.last_response = self.api.initialize_application(data_dir)
        self.api = StatusNodeRPC(self.port, self.name)
        self.start_signal_client()

    def create_account_and_login(self, account_data):
        logger.info(f"Creating account and logging in for node {self.name}")
        self.api.create_account_and_login(account_data)

    def start_messenger(self):
        logger.info(f"Starting Waku messenger for node {self.name}")
        self.api.start_messenger()

    def start_signal_client(self):
        ws_url = f"ws://localhost:{self.port}"
        await_signals = ["history.request.started", "messages.new", "message.delivered", "history.request.completed"]
        self.signal_client = SignalClient(ws_url, await_signals)

        websocket_thread = threading.Thread(target=self.signal_client._connect)
        websocket_thread.daemon = True
        websocket_thread.start()
        logger.info("WebSocket client started and subscribed to signals.")

    def wait_fully_started(self):
        logger.info(f"Waiting for {self.name} to fully start...")
        start_time = time.time()
        while time.time() - start_time < 30:
            if any("server started" in log for log in self.logs):
                logger.info(f"Node {self.name} has fully started.")
                return
            time.sleep(0.5)
        raise TimeoutError(f"Node {self.name} did not fully start in time.")

    def capture_process_logs(self, process, logs):
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
        allowed_chars = string.ascii_lowercase + string.digits + "_-"
        return ''.join(random.choice(allowed_chars) for _ in range(length))

    def get_pubkey(self, display_name):
        response = self.api.send_rpc_request("accounts_getAccounts")
        accounts = response.get("result", [])
        for account in accounts:
            if account.get("name") == display_name:
                return account.get("public-key")
        raise ValueError(f"Public key not found for display name: {display_name}")

    def wait_for_signal(self, signal_type, timeout=20):
        return self.signal_client.wait_for_signal(signal_type, timeout)

    def wait_for_complete_signal(self, signal_type, timeout=20):
        return self.signal_client.wait_for_complete_signal(signal_type, timeout)

    def stop(self, remove_local_data=True):
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

    def send_contact_request(self, pubkey, message):
        params = [{"id": pubkey, "message": message}]
        return self.api.send_rpc_request("wakuext_sendContactRequest", params)

    def send_message(self, pubkey, message):
        params = [{"id": pubkey, "message": message}]
        return self.api.send_rpc_request("wakuext_sendOneToOneMessage", params)

    def pause_process(self):
        if self.pid:
            logger.info(f"Pausing node with pid: {self.pid}")
            os.kill(self.pid, signal.SIGTSTP)

    def resume_process(self):
        if self.pid:
            logger.info(f"Resuming node with pid: {self.pid}")
            os.kill(self.pid, signal.SIGCONT)
