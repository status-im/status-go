import json
import logging
import os
import tempfile
import time
import uuid

import qrcode
import requests
from tenacity import retry, stop_after_delay, wait_fixed, wait_exponential, retry_if_exception_type

import resources.constants as constants
from clients.api import ApiClient
from clients.expvar import ExpvarClient
from clients.metrics import Events, StatusGoMetrics
from clients.rpc import RpcClient
from clients.services.accounts import AccountService
from clients.services.appgeneral import AppgeneralService
from clients.services.connector import ConnectorService
from clients.services.eth import EthService
from clients.services.multiaccounts import MultiAccountsService
from clients.services.newsfeed import NewsFeedService
from clients.services.settings import SettingsService
from clients.services.sharedurls import SharedURLsService
from clients.services.wakuext import (
    WakuextService,
    PushNotificationRegistrationTokenType,
)
from clients.services.wallet import WalletService
from clients.signals import SignalClient, SignalType
from clients.statusgo_container import StatusBackendContainer
from resources.constants import USE_IPV6, user_1, ANVIL_NETWORK_ID
from utils import fake
from utils import keys
from utils.config import Config

NANOSECONDS_PER_SECOND = 1_000_000_000


class StatusBackend(RpcClient, SignalClient, ApiClient):
    container = None

    def __init__(self, privileged=False, ipv6=USE_IPV6, **kwargs):
        self.temp_dir = None
        self.ipv6 = True if ipv6 == "Yes" else False
        logging.debug(f"Flag USE_IPV6 is: {self.ipv6}")

        url = None
        if kwargs.__contains__("url"):
            url = kwargs.get("url", "")
        elif Config.status_backend_urls:
            try:
                url = next(Config.status_backend_urls)
            except StopIteration:
                raise Exception("--status-backend-url is found, but not enough backends provided")

        data_dir = kwargs.get("data_dir", None)  # TODO: Should be argument of `init_status_backend` or fetched from the app
        self.logLevel = kwargs.get("logLevel", "DEBUG")

        if url:
            assert url != "", "not enough status-backend urls provided"
            if data_dir is None:
                self.temp_dir = tempfile.TemporaryDirectory()
                self.data_dir = self.temp_dir.name
            else:
                self.data_dir = data_dir
            if kwargs.get("connector_enabled", False):
                self.connector_ws_url = f"ws://localhost:{constants.STATUS_CONNECTOR_WS_PORT}"
        else:
            self.container = StatusBackendContainer(privileged, self.ipv6, **kwargs)
            self.temp_dir = None
            self.data_dir = self.container.data_dir()
            url = self.container.url
            if kwargs.get("connector_enabled", False):
                self.connector_ws_url = self.container.connector_ws_url

        assert self.data_dir != ""
        self.base_url = url
        self.api_url = f"{url}/statusgo"
        self.ws_url = f"{url}".replace("http", "ws")
        self.public_key = ""
        self.mnemonic = ""
        self.key_uid = ""
        self.password = ""
        self.display_name = ""
        self.device_id = str(uuid.uuid4())  # In reality this is taken from the device, don't confuse with Status installation_id
        self.device_platform = PushNotificationRegistrationTokenType.UNKNOWN
        self.node_login_event = {}
        self.events = Events()
        self.version = "unknown"
        self.network_id = 1

        RpcClient.__init__(self)
        ApiClient.__init__(self, self.api_url)
        SignalClient.__init__(self, self.ws_url)

        self.wait_for_healthy()

        SignalClient.connect(self)

        self.wallet_service = WalletService(self)
        self.wakuext_service = WakuextService(self)
        self.accounts_service = AccountService(self)
        self.newsfeed_service = NewsFeedService(self)
        self.multiaccounts_service = MultiAccountsService(self)
        self.settings_service = SettingsService(self)
        self.sharedurls_service = SharedURLsService(self)
        self.connector_service = ConnectorService(self)
        self.appgeneral_service = AppgeneralService(self)
        self.eth_service = EthService(self)
        self.expvar_client = ExpvarClient(self.base_url)

    def __del__(self):
        self.shutdown()

    def shutdown(self, log_sufix=""):
        SignalClient.disconnect(self)

        if self.container:
            self.container.shutdown(log_sufix)

        if self.temp_dir is not None:
            self.temp_dir.cleanup()

    @retry(
        stop=stop_after_delay(10),
        wait=wait_exponential(multiplier=1, min=0.1, max=5),
        retry=retry_if_exception_type((ConnectionError, requests.RequestException)),
        reraise=True,
    )
    def wait_for_healthy(self):
        response = self.health()
        response = json.loads(response.content)
        self.version = response.get("version", "unknown")
        logging.debug("StatusBackend is healthy")

    def health(self):
        return self.api_request("health", data=[], url=self.base_url, quiet=True)

    def initialize(self):
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
            "logLevel": self.logLevel,
            "apiLoggingEnabled": True,
            "wakuFleetsConfigFilePath": Config.waku_fleets_config,
            "pushFleetsConfigFilePath": Config.push_fleets_config,
            "mediaServerAddress": f"""{"0.0.0.0" if self.container else "127.0.0.1"}:{constants.STATUS_MEDIA_SERVER_PORT if self.container else 0}""",
            "mediaServerAdvertizeHost": "localhost" if self.container else "",
            "mediaServerAdvertizePort": self.container.media_server_port if self.container else 0,
        }

        return self.api_request_json(method, data)

    def _set_networks(self, data, **kwargs):
        self.network_id = kwargs.get("network_id", ANVIL_NETWORK_ID)
        anvil_network = {
            "chainID": self.network_id,
            "chainName": "Anvil",
            "rpcProviders": [
                {
                    "chainId": self.network_id,
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
        data["networkId"] = self.network_id
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

    def _set_multicall_overrides(self, data, kwargs):
        multicall_contract_address = kwargs.get("multicall_contract_address", None)
        if not multicall_contract_address:
            return data

        data["multicallOverrides"] = {self.network_id: multicall_contract_address}
        return data

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
        self.display_name = kwargs.get("display_name", fake.profile_name())

    def _create_account_request(self, password: str, **kwargs):
        self.password = password
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
            "logLevel": self.logLevel,
            # Waku config
            "wakuV2LightClient": kwargs.get("waku_light_client", False),
            "wakuV2Fleet": Config.waku_fleet,
            # Connector config
            "apiConfig": {
                "apiModules": "connector",
                "connectorEnabled": kwargs.get("connector_enabled", False),
                "httpEnabled": False,
                "httpHost": "0.0.0.0",
                "httpPort": 0,
                "wsEnabled": True,
                "wsHost": "0.0.0.0",
                "wsPort": constants.STATUS_CONNECTOR_WS_PORT,
            },
            "thirdpartyServicesEnabled": True,
        }
        if not Config.disable_override_networks:
            self._set_networks(data, **kwargs)

        data = self._set_proxy_credentials(data)
        data = self._set_wallet_secrets(data)
        data = self._set_multicall_overrides(data, kwargs)
        return data

    def create_account_and_login(self, password: str, **kwargs):
        self._set_display_name(**kwargs)
        method = "CreateAccountAndLogin"
        data = self._create_account_request(password=password, **kwargs)
        return self.api_request_json(method, data)

    def restore_account_and_login(self, user=user_1, **kwargs):
        self._set_display_name(**kwargs)
        method = "RestoreAccountAndLogin"
        data = self._create_account_request(password=user.password, **kwargs)
        data["mnemonic"] = user.passphrase
        return self.api_request_json(method, data)

    def login(self, key_uid, password: str, kdf_iterations=256000):
        self.password = password
        method = "LoginAccount"
        data = {
            "password": self.password,
            "keyUid": key_uid,
            "kdfIterations": kdf_iterations,
        }
        data = self._set_proxy_credentials(data)
        data = self._set_wallet_secrets(data)
        return self.api_request_json(method, data)

    def logout(self, **kwargs):
        method = "Logout"
        return self.api_request_json(method, {}, **kwargs)

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

    def wait_for_messages(self, timeout: int | None = 20):
        return self.wait_for_signal(SignalType.MESSAGES_NEW, timeout)

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
            response = self.wakuext_service.peers()
            if len(response.keys()) == 0:
                time.sleep(0.5)
                continue
            logging.info(f"StatusBackend is online after {time.time() - start_time} seconds")
            return
        raise TimeoutError(f"StatusBackend was not online after {timeout} seconds")

    def get_connection_string_for_bootstrapping_another_device(self, message_sync_enabled=False):
        method = "GetConnectionStringForBootstrappingAnotherDevice"
        data = {
            "senderConfig": {
                "keystorePath": os.path.join(self.data_dir, "keystore", self.key_uid),
                "deviceType": "macos",
                "keyUID": self.key_uid,
                "password": self.password,
                "chatKey": "",
                "messageSyncingEnabled": message_sync_enabled,
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
        data = {
            "connectionString": connection_string,
            "receiverClientConfig": {
                "receiverConfig": {"createAccount": self._create_account_request(password="")},
                "clientConfig": {},
            },
        }
        return self.api_request_json(method, data)

    def get_connection_string_for_being_bootstrapped(self):
        method = "GetConnectionStringForBeingBootstrapped"
        data = {
            "receiverConfig": {
                "createAccount": self._create_account_request(password=""),
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
        return self.api_request_json(method, data)

    def gather_metrics(self):
        if not self.container:
            raise RuntimeError("Gathering metrics is only supported when running status-backend in a Docker container")

        # Stop both monitoring threads and get independent arrays
        container_stats = self.container.stop_performance_monitoring()
        go_metrics = self.expvar_client.stop_monitoring()

        # Create PerformanceMetrics with independent arrays
        return StatusGoMetrics(
            container_stats=container_stats,
            go_metrics=go_metrics,
            events=self.events,
            version=self.version,
        )

    def start_performance_monitoring(self):
        """Start performance monitoring with independent threads"""
        if not self.container:
            raise RuntimeError("Performance monitoring is only supported when running status-backend in a Docker container")

        self.container.start_performance_monitoring()
        self.expvar_client.start_monitoring()

    def free_os_memory(self):
        url = f"{self.base_url}/statusgo/debug/FreeOSMemory"
        requests.post(url)

    def change_database_password(self, old_password, new_password):
        method = "ChangeDatabasePasswordV2"
        data = {
            "keyUid": self.key_uid,
            "oldPassword": old_password,
            "newPassword": new_password,
        }
        return self.api_request_json(method, data)

    def image_server_tls_cert(self):
        method = "ImageServerTLSCert"
        response = self.api_request(method, {})
        return response.content.decode("utf-8")

    def serialize_legacy_key(self, key):
        method = "SerializeLegacyKey"
        # Use client.post directly, because this method is old and has json-incompatible arguments
        response = self.client.post(self.method_url(method), data=key)
        return response.content.decode()

    def start_with_account(self, display_name: str, password: str, identity_image_path: str = "", **kwargs):
        response = self.initialize()
        if response is None:
            response = {}
        account_created = False
        for account in response.get("accounts", []) or []:
            if account["name"] == display_name:
                self.login(account["key-uid"], password=password)
                break
        else:
            print(f"Account '{display_name}' not found, creating...")
            self.create_account_and_login(password=password, display_name=display_name)
            account_created = True

        try:
            self.wait_for_login()
            self.wakuext_service.start_messenger()
            self.wallet_service.start_wallet()
        except Exception as e:
            if "node is already running" not in str(e):
                raise e

        if account_created:
            self.multiaccounts_service.store_identity_image(self.key_uid, identity_image_path, 0, 0, 1024, 1024)

    def generate_profile_qr_code(self):
        bot_url = self.wakuext_service.share_user_url_with_data(self.public_key)
        print(f"--- URL: {bot_url}")
        print(f"--- Public Key: {self.public_key}")

        img = qrcode.make(bot_url)
        img.save("qr_code.png")  # type: ignore
