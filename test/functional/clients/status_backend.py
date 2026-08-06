import copy
import itertools
import json
import logging
import os
import shutil
import tempfile
import time
import uuid

import requests
from tenacity import retry, stop_after_delay, wait_fixed, wait_exponential, retry_if_exception_type

import resources.constants as constants
from clients.api import ApiClient, ApiResponseError
from clients.expvar import ExpvarClient
from clients.metrics import Events, StatusGoMetrics
from clients.rpc import RpcClient
from clients.services.accounts import AccountService
from clients.services.appgeneral import AppgeneralService
from clients.services.connector import ConnectorService
from clients.services.ens import EnsService
from clients.services.eth import EthService
from clients.services.linkpreview import LinkPreviewService
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
    name: str = ""
    container: StatusBackendContainer | None = None
    _media_server_port_gen = itertools.count(constants.STATUS_MEDIA_SERVER_PORT, 1)
    _connector_ws_port_gen = itertools.count(constants.STATUS_CONNECTOR_WS_PORT, 1)

    def __init__(self, privileged=False, ipv6=USE_IPV6, **kwargs):
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
            if kwargs.get("connector_enabled", False):
                self.connector_ws_url = f"ws://localhost:{next(StatusBackend._connector_ws_port_gen)}"
            self.media_server_port = next(StatusBackend._media_server_port_gen)
        else:
            self.container = StatusBackendContainer(privileged, self.ipv6, **kwargs)
            self.temp_dir = None
            self.data_dir = self.container.data_dir()
            url = self.container.url
            if kwargs.get("connector_enabled", False):
                self.connector_ws_url = self.container.connector_ws_url
            self.media_server_port = self.container.ports.media_server.host_port

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
        self._boot_api_config = None
        try:
            RpcClient.__init__(self)
            ApiClient.__init__(self, self.api_url)
            SignalClient.__init__(self, self.ws_url)

            self.wait_for_healthy()

            # Skip sync signal client if we'll use async wrapper
            if not kwargs.get("skip_signal_client", False):
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
            self.ens_service = EnsService(self)
            self.eth_service = EthService(self)
            self.linkpreview_service = LinkPreviewService(self)
            self.expvar_client = ExpvarClient(self.base_url)
        except Exception:
            if self.container:
                self.container.shutdown()
            raise

    def __del__(self):
        self.shutdown()

    def shutdown(self, log_sufix=""):
        SignalClient.disconnect(self)

        if self.container:
            self.container.shutdown(log_sufix)

        if Config.logs_dir:
            try:
                # Check if base_url was initialized (may be missing if __init__ failed early)
                if hasattr(self, "base_url") and hasattr(self, "data_dir"):
                    self._export_logs(Config.logs_dir, log_sufix)
            except Exception as e:
                logging.warning(f"Failed to export logs: {e}")

        if self.temp_dir is not None:
            self.temp_dir.cleanup()

    def _export_logs(self, logs_dir: str, log_sufix: str):
        if self.container:
            # Container logs are exported by StatusGoContainer
            return

        timestamp = time.time()
        log_identifier = self.base_url.replace("http://", "").replace(":", "-")
        log_name = f"status-backend_{timestamp}_{log_sufix}_{log_identifier}"

        # Create logs subdirectory
        log_dir_path = os.path.join(logs_dir, log_name)
        os.makedirs(log_dir_path, exist_ok=True)

        # Copy all .log files from data_dir to log_dir_path
        for filename in os.listdir(self.data_dir):
            if not filename.endswith(".log"):
                continue
            src_path = os.path.join(self.data_dir, filename)
            dst_path = os.path.join(log_dir_path, filename)
            shutil.copy2(src_path, dst_path)

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
            "mediaServerAddress": (
                f"{'[::]' if self.ipv6 else '0.0.0.0'}:{self.container.ports.media_server.container_port}"
                if self.container
                else f"127.0.0.1:{self.media_server_port}"
            ),
            "mediaServerAdvertizeHost": "localhost",
            "mediaServerAdvertizePort": self.media_server_port,
        }

        return self.api_request_json(method, data)

    def _build_anvil_network(self, provider_type="embedded-direct", **kwargs):
        network_id = kwargs.get("network_id", ANVIL_NETWORK_ID)
        anvil_network = {
            "chainID": network_id,
            "chainName": "Anvil",
            "rpcProviders": [
                {
                    "chainId": network_id,
                    "name": "Anvil Direct",
                    "url": Config.anvil_url,
                    "enableRpsLimiter": False,
                    "type": provider_type,
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
        return self._set_token_overrides(anvil_network, kwargs.get("token_overrides", []))

    def _set_networks(self, data, **kwargs):
        self.network_id = kwargs.get("network_id", ANVIL_NETWORK_ID)

        # Allow callers (fixtures/tests) to add additional networks on top of the default Anvil network.
        # - networks_override: full replacement for networksOverride (list[dict])
        # - extra_networks_override: appended to the default Anvil network (list[dict])
        networks_override = kwargs.get("networks_override", None)
        extra_networks_override = kwargs.get("extra_networks_override", []) or []

        anvil_network = self._build_anvil_network(**kwargs)

        data["testNetworksEnabled"] = False
        data["networkId"] = self.network_id
        if networks_override is not None:
            data["networksOverride"] = networks_override
        else:
            data["networksOverride"] = [anvil_network, *extra_networks_override]

    def add_anvil_network(self, **kwargs):
        # LoginAccount rebuilds the network list from defaults (status-im/status-go#6010, #5597), so a
        # paired device that signs in via login() never gets the Anvil chain and drops token-gated
        # community messages. wallet_addEthereumChain (Upsert) keeps only USER providers, so add the
        # chain with a user provider — an embedded one is stripped, leaving the chain with no usable
        # provider, which fails as "could not find any enabled RPC providers for chain: 31337".
        network = self._build_anvil_network(provider_type="user", **kwargs)
        return self.wallet_service.add_ethereum_chain(network)

    def _set_proxy_credentials(self, data):
        if "STATUS_BUILD_PROXY_USER" not in os.environ:
            return data

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

    def _set_community_token_deployer_overrides(self, data, kwargs):
        deployer_address = kwargs.get("community_token_deployer_contract_address", None)
        if not deployer_address:
            return data

        data["communityTokenDeployerOverrides"] = {self.network_id: deployer_address}
        return data

    def _set_custom_tokens(self, data, kwargs):
        token_overrides = kwargs.get("token_overrides", [])
        if not token_overrides:
            return data

        tokens = []
        for token in token_overrides:
            tokens.append(
                {
                    "chainId": self.network_id,
                    "address": token["address"],
                    "symbol": token.get("symbol"),
                    "name": token.get("name", token.get("symbol")),
                    "decimals": token.get("decimals", 18),
                }
            )

        data["customTokens"] = tokens
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
        self.waku_light_client = kwargs.get("waku_light_client", False)
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
            "logosStorageConfigEnabled": kwargs.get("logos_storage_config_enabled", False),
            "logosStorageConfigBootstrapNode": kwargs.get("logos_storage_config_bootstrap_node", None),
            "importInitialDelay": kwargs.get("import_initial_delay", None),
            "messageArchiveInterval": kwargs.get("message_archive_interval", None),
            "torrentConfigEnabled": False,
            "torrentConfigPort": 9025,
            "thirdpartyServicesEnabled": True,
        }

        verify_ens = kwargs.get("verify_ens_contract_address")
        if verify_ens is not None:
            data["verifyENSContractAddress"] = verify_ens

        if not Config.disable_override_networks:
            self._set_networks(data, **kwargs)

        data = self._set_proxy_credentials(data)
        data = self._set_wallet_secrets(data)
        data = self._set_multicall_overrides(data, kwargs)
        data = self._set_community_token_deployer_overrides(data, kwargs)
        data = self._set_custom_tokens(data, kwargs)
        return data

    def create_account_and_login(self, password: str, **kwargs):
        self._set_display_name(**kwargs)
        method = "CreateAccountAndLogin"
        data = self._create_account_request(password=password, **kwargs)
        self._boot_api_config = copy.deepcopy(data.get("apiConfig", {}))
        return self.api_request_json(method, data)

    def restore_account_and_login(self, user=user_1, **kwargs):
        self._set_display_name(**kwargs)
        method = "RestoreAccountAndLogin"
        data = self._create_account_request(password=user.password, **kwargs)
        data["mnemonic"] = user.passphrase
        self._boot_api_config = copy.deepcopy(data.get("apiConfig", {}))
        return self.api_request_json(method, data)

    def login(self, key_uid, password: str, kdf_iterations=256000):
        self.password = password
        # Reconnect to signals before login to avoid missing node.login after logout.
        SignalClient.disconnect(self)
        SignalClient.connect(self)
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
        try:
            return self.api_request_json(method, {}, **kwargs)
        except ApiResponseError as e:
            # Logout is idempotent: a node that is already logged out (e.g. a
            # test that logs out explicitly, or a failed login) reports "there
            # is no running node". Treat that as a successful no-op so teardown
            # doesn't turn a clean run into an error.
            if "there is no running node" in str(e):
                logging.debug("Logout called on an already-logged-out node; treating as no-op")
                return None
            raise

    def wait_for_login(self):
        """Wait until the backend has completed login.

        Historically we relied on the `node.login` signal.
        In some environments (notably `wakuV2LightClient=true`) this signal can be delayed or
        occasionally missed by the websocket client. To keep tests stable we:
        - try to wait for `node.login` first
        - if it doesn't arrive, fall back to polling RPC state and extracting the same fields
        """

        def _apply_login_signal(signal: dict):
            if "error" in signal.get("event", {}):
                error_details = signal["event"]["error"]
                assert not error_details, f"Unexpected error during login: {error_details}"
            self.node_login_event = signal
            logging.debug(f"Node login event: {self.node_login_event}")
            self.public_key = self.node_login_event.get("event", {}).get("settings", {}).get("public-key", "")
            self.mnemonic = self.node_login_event.get("event", {}).get("settings", {}).get("mnemonic", "")
            self.key_uid = self.node_login_event.get("event", {}).get("account", {}).get("key-uid", "")

        # 1) Preferred path: wait for the `node.login` signal (race-safe with backlog).
        try:
            with self.expect_signal(SignalType.NODE_LOGIN, timeout=60, start="beginning") as exp:
                pass
            signal = exp.result
            assert isinstance(signal, dict), f"Unexpected NODE_LOGIN signal payload type: {type(signal)}"
            _apply_login_signal(signal)
            return signal
        except TimeoutError:
            logging.warning("NODE_LOGIN signal was not received in time; falling back to RPC polling to confirm login")

        # 2) Fallback path: poll RPC state until it reflects a logged-in account.
        # We keep this bounded to avoid hiding real hangs.
        deadline = time.monotonic() + 60
        last_settings = None
        last_keypairs = None
        last_error = None

        while time.monotonic() < deadline:
            try:
                # If the signal arrived late while we were falling back, prefer it.
                buffered = self.received_signals.get(SignalType.NODE_LOGIN, [])
                if buffered:
                    signal = buffered[-1]
                    assert isinstance(signal, dict), f"Unexpected buffered NODE_LOGIN payload type: {type(signal)}"
                    _apply_login_signal(signal)
                    return signal

                last_settings = self.settings_service.get_settings()
                public_key = (last_settings or {}).get("public-key", "")
                mnemonic = (last_settings or {}).get("mnemonic", "")

                last_keypairs = self.accounts_service.get_account_keypairs() or []
                key_uid = ""
                if isinstance(last_keypairs, list) and last_keypairs:
                    # The profile keypair is expected to be present as the first entry in most setups.
                    key_uid = (last_keypairs[0] or {}).get("key-uid", "")

                if public_key and key_uid:
                    signal = {
                        "type": SignalType.NODE_LOGIN.value,
                        "event": {
                            "settings": {
                                "public-key": public_key,
                                "mnemonic": mnemonic,
                            },
                            "account": {
                                "key-uid": key_uid,
                            },
                        },
                    }
                    _apply_login_signal(signal)
                    return signal
            except Exception as e:
                last_error = str(e)

            time.sleep(0.5)

        raise TimeoutError(
            "Login did not complete within timeout: NODE_LOGIN not received and RPC state did not converge. "
            f"last_error={last_error}, last_settings_keys={list((last_settings or {}).keys()) if isinstance(last_settings, dict) else None}, "
            f"last_keypairs_len={len(last_keypairs) if isinstance(last_keypairs, list) else None}"
        )

    def wait_for_wakuext_ready(self, timeout: float = 30, poll_interval: float = 0.5, *, start_messenger: bool = True):
        """Wait until `wakuext_*` RPC namespace is available after login.

        In some environments `wait_for_login()` can complete before the `wakuext` RPC namespace is fully
        registered/ready, so the first `wakuext_*` call may fail with `-32601`.

        This helper keeps the waiting logic out of tests.

        Args:
            timeout: Total time in seconds.
            poll_interval: Sleep interval between attempts.
            start_messenger: If True, try to call `wakuext_startMessenger` while waiting.

        Raises:
            TimeoutError: if `wakuext` does not become ready within the given timeout.
        """

        deadline = time.monotonic() + float(timeout)
        last_error = None
        messenger_started = False

        while time.monotonic() < deadline:
            try:
                if start_messenger and not messenger_started:
                    # Best-effort: this itself can fail with -32601 if the namespace isn't ready yet.
                    self.wakuext_service.start_messenger()
                    messenger_started = True

                # Probe a lightweight wakuext call that doesn't produce log noise.
                _ = self.wakuext_service.chats()
                return
            except Exception as e:
                last_error = str(e)
                time.sleep(poll_interval)

        raise TimeoutError(f"wakuext RPC namespace did not become ready in {timeout} seconds: {last_error}")

    def wait_for_messages(self, timeout: int | None = 20):
        with self.expect_signal(SignalType.MESSAGES_NEW, timeout=timeout or 20) as exp:
            pass
        return exp.result

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
            try:
                response = self.wakuext_service.peers()
            except Exception as ex:
                logging.debug(f"StatusBackend peers() check failed: {ex}")
                time.sleep(0.5)
                continue
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

    def change_database_password(self, old_password, new_password, rekey=False):
        method = "ChangeDatabasePasswordV2"
        data = {
            "keyUid": self.key_uid,
            "oldPassword": old_password,
            "newPassword": new_password,
            "rekey": rekey,
        }
        return self.api_request_json(method, data)

    def get_profile_encryption_info(self):
        method = "GetProfileEncryptionInfo"
        data = {
            "keyUid": self.key_uid,
        }
        return self.api_request_json(method, data)

    def image_server_tls_cert(self):
        method = "ImageServerTLSCert"
        response = self.api_request(method, {})
        return response.content.decode("utf-8")

    def get_boot_api_config(self):
        return self._boot_api_config

    def serialize_legacy_key(self, key):
        method = "SerializeLegacyKey"
        # Use client.post directly, because this method is old and has json-incompatible arguments
        response = self.client.post(self.method_url(method), data=key)
        return response.content.decode()
