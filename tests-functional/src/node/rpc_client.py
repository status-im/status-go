from src.libs.base_api_client import BaseAPIClient
from src.config import Config
from src.libs.custom_logger import get_custom_logger
from tenacity import retry, stop_after_attempt, wait_fixed

logger = get_custom_logger(__name__)


class StatusNodeRPC(BaseAPIClient):
    def __init__(self, port, node_name):
        super().__init__(f"http://127.0.0.1:{port}/statusgo")
        self.node_name = node_name

    @retry(
        stop=stop_after_attempt(3),
        wait=wait_fixed(1),
        reraise=True
    )
    def send_rpc_request(self, method, params=None, timeout=Config.API_REQUEST_TIMEOUT):
        """Send JSON-RPC requests, used for standard JSON-RPC API calls."""
        payload = {"jsonrpc": "2.0", "method": method, "params": params or [], "id": 1}
        logger.info(f"Sending JSON-RPC request to {self.base_url} with payload: {payload}")

        response = self.send_post_request("CallRPC", payload, timeout=timeout)

        logger.info(f"Response received: {response}")

        if response.get("error"):
            logger.error(f"RPC request failed with error: {response['error']}")
            raise RuntimeError(f"RPC request failed with error: {response['error']}")

        return response

    @retry(
        stop=stop_after_attempt(3),
        wait=wait_fixed(1),
        reraise=True
    )
    def initialize_application(self, data_dir, timeout=Config.API_REQUEST_TIMEOUT):
        """Send a direct POST request to the InitializeApplication endpoint."""
        payload = {"dataDir": data_dir}
        logger.info(f"Sending direct POST request to InitializeApplication with payload: {payload}")

        response = self.send_post_request("InitializeApplication", payload, timeout=timeout)

        logger.info(f"Response from InitializeApplication: {response}")

        if response.get("error"):
            logger.error(f"InitializeApplication request failed with error: {response['error']}")
            raise RuntimeError(f"Failed to initialize application: {response['error']}")

        return response

    @retry(
        stop=stop_after_attempt(3),
        wait=wait_fixed(1),
        reraise=True
    )
    def create_account_and_login(self, account_data, timeout=Config.API_REQUEST_TIMEOUT):
        """Send a direct POST request to CreateAccountAndLogin endpoint."""
        payload = {
            "rootDataDir": account_data.get("rootDataDir"),
            "displayName": account_data.get("displayName", "test1"),
            "password": account_data.get("password", "test1"),
            "customizationColor": account_data.get("customizationColor", "primary")
        }
        logger.info(f"Sending direct POST request to CreateAccountAndLogin with payload: {payload}")

        response = self.send_post_request("CreateAccountAndLogin", payload, timeout=timeout)

        logger.info(f"Response from CreateAccountAndLogin: {response}")

        if response.get("error"):
            logger.error(f"CreateAccountAndLogin request failed with error: {response['error']}")
            raise RuntimeError(f"Failed to create account and login: {response['error']}")

        return response

    @retry(
        stop=stop_after_attempt(3),
        wait=wait_fixed(1),
        reraise=True
    )
    def start_messenger(self, timeout=Config.API_REQUEST_TIMEOUT):
        """Send JSON-RPC request to start Waku messenger."""
        payload = {
            "jsonrpc": "2.0",
            "method": "wakuext_startMessenger",
            "params": [],
            "id": 1
        }
        logger.info(f"Sending JSON-RPC request to start Waku messenger: {payload}")

        response = self.send_post_request("CallRPC", payload, timeout=timeout)

        logger.info(f"Response from Waku messenger start: {response}")

        if response.get("error"):
            logger.error(f"Starting Waku messenger failed with error: {response['error']}")
            raise RuntimeError(f"Failed to start Waku messenger: {response['error']}")

        return response
