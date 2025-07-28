import json
import logging
import string
import tempfile
import time
import random
import threading
import uuid
import requests
import os

from tenacity import retry, stop_after_delay, wait_fixed
from clients.services.wallet import WalletService
from clients.services.wakuext import WakuextService, PushNotificationRegistrationTokenType
from clients.services.accounts import AccountService
from clients.services.settings import SettingsService
from clients.signals import SignalClient, SignalType
from clients.rpc import RpcClient
from clients.statusgo_container import StatusBackendContainer
from utils.config import Config
from resources.constants import USE_IPV6, user_1, ANVIL_NETWORK_ID, Account
from utils import keys

NANOSECONDS_PER_SECOND = 1_000_000_000


class StatusBackend(RpcClient, SignalClient):
    container = None

    def __init__(self, await_signals=[], privileged=False, ipv6=USE_IPV6):
        self.temp_dir = None
        self.ipv6 = True if ipv6 == "Yes" else False
        logging.debug(f"Flag USE_IPV6 is: {self.ipv6}")

        if Config.status_backend_urls:
            try:
                url = next(Config.status_backend_urls)
            except StopIteration:
                raise Exception("--status-backend-url is found, but not enough backends provided")

            assert url != "", "not enough status-backend urls provided"
            self.temp_dir = tempfile.TemporaryDirectory()
            self.data_dir = self.temp_dir.name
        else:
            host_port = random.choice(Config.status_backend_port_range)
            Config.status_backend_port_range.remove(host_port)

            self.container = StatusBackendContainer(host_port, privileged, self.ipv6)
            self.temp_dir = None
            self.data_dir = self.container.data_dir()
            url = self.container.url

        assert self.data_dir != ""
        self.base_url = url
        self.api_url = f"{url}/statusgo"
        self.ws_url = f"{url}".replace("http", "ws")
        self.rpc_url = f"{url}/statusgo/CallRPC"
        self.public_key = ""
        self.mnemonic = ""
        self.key_uid = ""
        self.password = ""
        self.display_name = ""
        self.device_id = str(uuid.uuid4())  # In reality this is taken from the device, don't confuse with Status installation_id
        self.device_platform = PushNotificationRegistrationTokenType.UNKNOWN
        self.node_login_event = {}

        RpcClient.__init__(self, self.rpc_url)
        SignalClient.__init__(self, self.ws_url, await_signals)

        self.wait_for_healthy()

        websocket_thread = threading.Thread(target=self._connect)
        websocket_thread.daemon = True
        websocket_thread.start()

        self.wallet_service = WalletService(self)
        self.wakuext_service = WakuextService(self)
        self.accounts_service = AccountService(self)
        self.settings_service = SettingsService(self)

    def __del__(self):
        if self.temp_dir is not None:
            self.temp_dir.cleanup()

    def wait_for_healthy(self, timeout=10):
        start_time = time.time()
        while time.time() - start_time <= timeout:
            try:
                self.health(enable_logging=True)
                logging.debug(f"StatusBackend is healthy after {time.time() - start_time} seconds")
                return
            except Exception as ex:
                logging.debug(f"StatusBackend error: {ex}")
                time.sleep(0.1)
        raise TimeoutError(f"StatusBackend was not healthy after {timeout} seconds")

    def health(self, enable_logging=True):
        return self.api_request("health", data=[], url=self.base_url, enable_logging=enable_logging)

    def api_request(self, method, data, url=None, enable_logging=True):
        url = url if url else self.api_url
        url = f"{url}/{method}"
        if enable_logging:
            logging.debug(f"Sending POST request to url {url} with data: {json.dumps(data, sort_keys=True)}")
        response = requests.post(url, json=data)
        if enable_logging:
            logging.debug(f"Got response: {response.content}")
        return response

    def verify_is_valid_api_response(self, response):
        assert response.status_code == 200, f"Got response {response.content}, status code {response.status_code}"
        assert response.content
        logging.debug(f"Got response: {response.content}")
        try:
            error = response.json()["error"]
            assert not error, f"Error: {error}"
        except json.JSONDecodeError:
            raise AssertionError(f"Invalid JSON in response: {response.content}")
        except KeyError:
            pass

    def api_valid_request(self, method, data, url=None):
        response = self.api_request(method, data, url)
        self.verify_is_valid_api_response(response)
        return response

    def init_status_backend(self):
        if Config.logout:
            logging.warning("automatically logging out before InitializeApplication")
            try:
                self.logout()
                logging.debug("successfully logged out")
            except Exception:
                logging.debug("failed to log out")
                pass

        method = "InitializeApplication"
        data = {
            "dataDir": self.data_dir,
            "logEnabled": True,
            "logLevel": "DEBUG",
            "apiLoggingEnabled": True,
            "wakuFleetsConfigFilePath": Config.waku_fleets_config,
            "pushFleetsConfigFilePath": Config.push_fleets_config,
        }

        return self.api_valid_request(method, data)

    def _set_networks(self, data, **kwargs):
        network_id = kwargs.get("network_id", ANVIL_NETWORK_ID)
        anvil_network = {
            "chainID": network_id,
            "chainName": "Anvil",
            "rpcProviders": [
                {
                    "chainId": network_id,
                    "name": "Anvil Direct",
                    "url": "http://anvil:8545",
                    "enableRpsLimiter": False,
                    "type": "embedded-direct",
                    "enabled": True,
                    "authType": "no-auth",
                }
            ],
            "shortName": "eth",
            "nativeCurrencyName": "Ether",
            "nativeCurrencySymbol": "ETH",
            "nativeCurrencyDecimals": 18,
            "isTest": False,
            "layer": 1,
            "enabled": True,
            "isActive": True,
            "isDeactivatable": False,
        }
        anvil_network = self._set_token_overrides(anvil_network, kwargs.get("token_overrides", []))

        data["testNetworksEnabled"] = False
        data["networkId"] = network_id
        data["networksOverride"] = [anvil_network]

    def _set_proxy_credentials(self, data):
        if "STATUS_BUILD_PROXY_USER" not in os.environ:
            return data

        user = os.environ["STATUS_BUILD_PROXY_USER"]
        password = os.environ["STATUS_BUILD_PROXY_PASSWORD"]

        data["StatusProxyMarketUser"] = user
        data["StatusProxyMarketPassword"] = password
        data["StatusProxyBlockchainUser"] = user
        data["StatusProxyBlockchainPassword"] = password

        data["StatusProxyEnabled"] = True
        data["StatusProxyStageName"] = "test"
        return data

    def _set_wallet_secrets(self, data):
        if "STATUS_BUILD_INFURA_TOKEN" in os.environ:
            data["infuraToken"] = os.environ["STATUS_BUILD_INFURA_TOKEN"]
        if "STATUS_BUILD_INFURA_SECRET" in os.environ:
            data["infuraSecret"] = os.environ["STATUS_BUILD_INFURA_SECRET"]
        if "STATUS_BUILD_POKT_TOKEN" in os.environ:
            data["poktToken"] = os.environ["STATUS_BUILD_POKT_TOKEN"]
        return data

    def _set_token_overrides(self, network, token_overrides):
        if not token_overrides:
            return network

        network["TokenOverrides"] = token_overrides
        return network

    def extract_data(self, path: str):
        if self.container:
            return self.container.extract_data(path)

        if not os.path.exists(path):
            return None

        return path

    def import_data(self, src_path: str, dest_path: str):
        """
        Import a file from the host (src_path) into the container at dest_path.
        If not running in a container, just copy the file locally.
        """
        if self.container:
            self.container.import_data(src_path, dest_path)
            return

        # Not running in a container, just copy the file locally
        if not os.path.exists(src_path):
            raise FileNotFoundError(f"Source path '{src_path}' does not exist.")

        os.makedirs(os.path.dirname(dest_path), exist_ok=True)
        with open(src_path, "rb") as src, open(dest_path, "wb") as dst:
            dst.write(src.read())

    def _set_display_name(self, **kwargs):
        self.display_name = kwargs.get(
            "display_name",
            f"DISP_NAME_{''.join(random.choices(string.ascii_letters + string.digits + '_-', k=10))}",
        )

    def _create_account_request(self, user, **kwargs):
        self.password = kwargs.get("password", user.password)
        data = {
            "rootDataDir": self.data_dir,
            "kdfIterations": 256000,
            # Profile config
            "displayName": self.display_name,
            "password": self.password,
            "customizationColor": kwargs.get("customizationColor", "primary"),
            # Logs config
            "logEnabled": True,
            "logToStderr": True,
            "logLevel": "DEBUG",
            # Waku config
            "wakuV2LightClient": kwargs.get("wakuV2LightClient", False),
            "wakuV2Fleet": Config.waku_fleet,
        }
        if not Config.disable_override_networks:
            self._set_networks(data, **kwargs)

        data = self._set_proxy_credentials(data)
        data = self._set_wallet_secrets(data)
        return data

    def create_account_and_login(self, user=user_1, **kwargs):
        self._set_display_name(**kwargs)
        method = "CreateAccountAndLogin"
        data = self._create_account_request(user, **kwargs)
        return self.api_valid_request(method, data)

    def restore_account_and_login(self, user=user_1, **kwargs):
        self._set_display_name(**kwargs)
        method = "RestoreAccountAndLogin"
        data = self._create_account_request(user, **kwargs)
        data["mnemonic"] = user.passphrase
        return self.api_valid_request(method, data)

    def login(self, keyUid, user=user_1):
        self.password = user.password
        method = "LoginAccount"
        data = {
            "password": user.password,
            "keyUid": keyUid,
            "kdfIterations": 256000,
        }
        data = self._set_proxy_credentials(data)
        data = self._set_wallet_secrets(data)
        return self.api_valid_request(method, data)

    def logout(self):
        method = "Logout"
        return self.api_valid_request(method, {})

    def wait_for_login(self):
        signal = self.wait_for_signal(SignalType.NODE_LOGIN.value)
        if "error" in signal["event"]:
            error_details = signal["event"]["error"]
            assert not error_details, f"Unexpected error during login: {error_details}"
        self.node_login_event = signal
        logging.debug(f"Node login event: {self.node_login_event}")
        self.public_key = self.node_login_event.get("event", {}).get("settings", {}).get("public-key")
        self.mnemonic = self.node_login_event.get("event", {}).get("settings", {}).get("mnemonic")
        self.key_uid = self.node_login_event.get("event", {}).get("account", {}).get("key-uid")
        return signal

    def container_pause(self):
        if not self.container:
            raise RuntimeError("Container is not initialized.")
        self.container.pause()

    def container_unpause(self):
        if not self.container:
            raise RuntimeError("Container is not initialized.")
        self.container.unpause()

    def container_exec(self, command):
        if not self.container:
            raise RuntimeError("Container is not initialized.")
        return self.container.exec(command)

    def compressed_public_key(self):
        if not self.public_key:
            return ""
        return keys.compress_public_key(self.public_key)

    @retry(stop=stop_after_delay(10), wait=wait_fixed(0.1), reraise=True)
    def change_container_ip(self, new_ipv4=None, new_ipv6=None):
        if not self.container:
            raise RuntimeError("Container is not initialized.")
        self.container.change_ip(new_ipv4, new_ipv6)

    def wait_for_online(self, timeout=10):
        start_time = time.time()
        while time.time() - start_time <= timeout:
            response = self.wakuext_service.peers(enable_logging=False)
            if len(response["result"].keys()) == 0:
                time.sleep(0.5)
                continue
            logging.info(f"StatusBackend is online after {time.time() - start_time} seconds")
            return
        raise TimeoutError(f"StatusBackend was not online after {timeout} seconds")

    def get_connection_string_for_bootstrapping_another_device(self):
        method = "GetConnectionStringForBootstrappingAnotherDevice"
        data = {
            "senderConfig": {
                "keystorePath": os.path.join(self.data_dir, "keystore", self.key_uid),
                "deviceType": "macos",
                "keyUID": self.key_uid,
                "password": self.password,
                "chatKey": "",
            },
            "serverConfig": {
                "timeout": 5 * 60 * 1000,
            },
        }
        response = self.api_request(method, data)
        return response.content.decode()

    def input_connection_string_for_bootstrapping(self, connection_string):
        method = "InputConnectionStringForBootstrappingV2"
        # Empty user
        user = Account(
            address="",
            private_key="",
            password="",
            passphrase="",
        )
        data = {
            "connectionString": connection_string,
            "receiverClientConfig": {
                "receiverConfig": {"createAccount": self._create_account_request(user)},
                "clientConfig": {},
            },
        }
        response = self.api_valid_request(method, data)
        return json.loads(response.content)

    def get_connection_string_for_being_bootstrapped(self):
        method = "GetConnectionStringForBeingBootstrapped"
        user = Account(
            address="",
            private_key="",
            password="",
            passphrase="",
        )
        data = {
            "receiverConfig": {
                "createAccount": self._create_account_request(user),
                "deviceType": "macos",
            },
            "serverConfig": {
                "timeout": 5 * 60 * 1000,
            },
        }
        response = self.api_request(method, data)
        return response.content.decode()

    def input_connection_string_for_bootstrapping_another_device(self, connection_string):
        method = "InputConnectionStringForBootstrappingAnotherDeviceV2"
        data = {
            "connectionString": connection_string,
            "senderClientConfig": {
                "senderConfig": {
                    "keystorePath": os.path.join(self.data_dir, "keystore", self.key_uid),
                    "deviceType": "macos",
                    "keyUID": self.key_uid,
                    "password": self.password,
                    "chatKey": "",
                },
                "clientConfig": {},
            },
        }
        response = self.api_request(method, data)
        return json.loads(response.content)
