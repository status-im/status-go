import io
import json
import logging
import string
import tarfile
import tempfile
import time
import random
import threading
import requests
import docker
import docker.errors
import os

from tenacity import retry, stop_after_delay, wait_fixed
from clients.services.wallet import WalletService
from clients.services.wakuext import WakuextService
from clients.services.accounts import AccountService
from clients.services.settings import SettingsService
from clients.signals import SignalClient
from clients.rpc import RpcClient
from conftest import option
from resources.constants import USE_IPV6, user_1, ANVIL_NETWORK_ID
from docker.errors import APIError

NANOSECONDS_PER_SECOND = 1_000_000_000


class StatusBackend(RpcClient, SignalClient):
    container = None

    def __init__(self, await_signals=[], privileged=False, ipv6=USE_IPV6):
        self.ipv6 = True if ipv6 == "Yes" else False
        logging.debug(f"Flag USE_IPV6 is: {self.ipv6}")
        self.docker_project_name = option.docker_project_name
        self.network_name = f"{self.docker_project_name}_default"

        if option.status_backend_url:
            url = next(option.status_backend_urls)
            assert url != "", "not enough status-backend urls provided"
            self.temp_dir = tempfile.TemporaryDirectory()
            self.data_dir = self.temp_dir.name
        else:
            self.data_dir = "/usr/status-user"
            self.docker_client = docker.from_env()
            retries = 5
            ports_tried = []
            for _ in range(retries):
                try:
                    host_port = random.choice(option.status_backend_port_range)
                    ports_tried.append(host_port)
                    self.container = self._start_container(host_port, privileged)
                    url = f"http://{'[::1]' if self.ipv6 else '127.0.0.1'}:{host_port}"
                    option.status_backend_port_range.remove(host_port)
                    break
                except Exception as ex:
                    logging.error(f"Error in starting the container: {str(ex)}")
            else:
                raise RuntimeError(f"Failed to start container on ports: {ports_tried}")

        assert self.data_dir != ""
        self.base_url = url
        self.api_url = f"{url}/statusgo"
        self.ws_url = f"{url}".replace("http", "ws")
        self.rpc_url = f"{url}/statusgo/CallRPC"
        self.public_key = ""

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

    def _start_container(self, host_port, privileged):
        identifier = os.environ.get("BUILD_ID") if os.environ.get("CI") else os.popen("git rev-parse --short HEAD").read().strip()
        image_name = f"{self.docker_project_name}-status-backend:latest"
        container_name = f"{self.docker_project_name}-{identifier}-status-backend-{host_port}"

        coverage_path = option.codecov_dir if option.codecov_dir else os.path.abspath("./coverage/binary")

        container_args = {
            "image": image_name,
            "detach": True,
            "privileged": privileged,
            "name": container_name,
            "labels": {"com.docker.compose.project": self.docker_project_name},
            "entrypoint": ["status-backend", "--address", "0.0.0.0:3333"],
            "ports": {"3333/tcp": host_port},
            "environment": {
                "GOCOVERDIR": "/coverage/binary",
            },
            "volumes": {
                coverage_path: {
                    "bind": "/coverage/binary",
                    "mode": "rw",
                }
            },
        }

        if self.ipv6:
            container_args.update(
                {
                    "entrypoint": ["status-backend", "--address", f"[::]:{host_port}"],
                    "ports": {
                        f"{host_port}/tcp": [
                            {"HostIp": "::", "HostPort": str(host_port)},
                        ]
                    },
                }
            )

        if "FUNCTIONAL_TESTS_DOCKER_UID" in os.environ:
            container_args["user"] = os.environ["FUNCTIONAL_TESTS_DOCKER_UID"]

        container = self.docker_client.containers.run(**container_args)

        network = self.docker_client.networks.get(self.network_name)
        network.connect(container)

        option.status_backend_containers.append(self)
        return container

    def wait_for_healthy(self, timeout=10):
        start_time = time.time()
        while time.time() - start_time <= timeout:
            try:
                self.health(enable_logging=True)
                logging.info(f"StatusBackend is healthy after {time.time() - start_time} seconds")
                return
            except Exception as ex:
                logging.error(ex)
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
        if option.logout:
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
            "apiLogging": True,
            "wakuFleetsConfigFilePath": option.waku_fleets_config,
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

    def _set_token_overrides(self, network, token_overrides):
        if not token_overrides:
            return network

        network["TokenOverrides"] = token_overrides
        return network

    def extract_data(self, path: str):
        if not self.container:
            return path

        try:
            stream, _ = self.container.get_archive(path)
        except docker.errors.NotFound:
            return None

        temp_dir = tempfile.mkdtemp()
        tar_bytes = io.BytesIO(b"".join(stream))

        with tarfile.open(fileobj=tar_bytes) as tar:
            tar.extractall(path=temp_dir)
            # If the tar contains a single file, return the path to that file
            # Otherwise it's a directory, just return temp_dir.
            if len(tar.getmembers()) == 1:
                return os.path.join(temp_dir, tar.getmembers()[0].name)

        return temp_dir

    def _set_display_name(self, **kwargs):
        self.display_name = kwargs.get(
            "display_name",
            f"DISP_NAME_{''.join(random.choices(string.ascii_letters + string.digits + '_-', k=10))}",
        )

    def _create_account_request(self, user, **kwargs):
        data = {
            "rootDataDir": self.data_dir,
            "kdfIterations": 256000,
            # Profile config
            "displayName": self.display_name,
            "password": kwargs.get("password", user.password),
            "customizationColor": kwargs.get("customizationColor", "primary"),
            # Logs config
            "logEnabled": True,
            "logLevel": "DEBUG",
            # Waku config
            "wakuV2LightClient": kwargs.get("wakuV2LightClient", False),
            "wakuV2Fleet": option.waku_fleet,
        }
        self._set_networks(data, **kwargs)
        data = self._set_proxy_credentials(data)
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
        method = "LoginAccount"
        data = {
            "password": user.password,
            "keyUid": keyUid,
            "kdfIterations": 256000,
        }
        data = self._set_proxy_credentials(data)
        return self.api_valid_request(method, data)

    def logout(self):
        method = "Logout"
        return self.api_valid_request(method, {})

    def container_pause(self):
        if not self.container:
            raise RuntimeError("Container is not initialized.")
        self.container.pause()
        logging.info(f"Container {self.container.name} paused.")

    def container_unpause(self):
        if not self.container:
            raise RuntimeError("Container is not initialized.")
        self.container.unpause()
        logging.info(f"Container {self.container.name} unpaused.")

    def container_exec(self, command):
        if not self.container:
            raise RuntimeError("Container is not initialized.")
        try:
            exec_result = self.container.exec_run(cmd=["sh", "-c", command], stdout=True, stderr=True, tty=False)
            if exec_result.exit_code != 0:
                raise RuntimeError(f"Failed to execute command in container {self.container.id}:\n" f"OUTPUT: {exec_result.output.decode().strip()}")
            return exec_result.output.decode().strip()
        except APIError as e:
            raise RuntimeError(f"API error during container execution: {str(e)}") from e

    def find_public_key(self):
        self.public_key = self.node_login_event.get("event", {}).get("settings", {}).get("public-key")

    def find_key_uid(self):
        return self.node_login_event.get("event", {}).get("account", {}).get("key-uid")

    @retry(stop=stop_after_delay(10), wait=wait_fixed(0.1), reraise=True)
    def change_container_ip(self, new_ipv4=None, new_ipv6=None):
        if not self.container:
            raise RuntimeError("Container is not initialized.")

        logging.info(f"Trying to change container {self.container.name} IPs (IPv6 Mode: {self.ipv6})")

        try:
            # Get the network details
            network = self.docker_client.networks.get(self.network_name)

            # Ensure network has explicitly configured subnets
            ipam_config = network.attrs.get("IPAM", {}).get("Config", [])
            if not ipam_config:
                raise RuntimeError("Network does not have a user-defined subnet, cannot assign a custom IP.")

            self.container.reload()
            container_info = self.container.attrs["NetworkSettings"]["Networks"].get(self.network_name, {})
            current_ipv4 = container_info.get("IPAddress", "Unknown")
            current_ipv6 = container_info.get("GlobalIPv6Address", "Unknown")

            logging.info(f"Current IPs for {self.container.name} - IPv4: {current_ipv4}, IPv6: {current_ipv6}")

            # Generate new IPs based on mode
            for config in ipam_config:
                subnet = config.get("Subnet")

                if self.ipv6 and ":" in subnet and not new_ipv6:  # IPv6 Subnet
                    base_ipv6 = subnet.rstrip("::/64")
                    new_ipv6 = f"{base_ipv6}::{random.randint(1, 9999):x}:{random.randint(1, 9999):x}"
                    logging.info(f"Generated new IPv6: {new_ipv6}")

                elif not self.ipv6 and "." in subnet and not new_ipv4:  # IPv4 Subnet
                    new_ipv4 = subnet.rsplit(".", 1)[0] + f".{random.randint(2, 254)}"
                    logging.info(f"Generated new IPv4: {new_ipv4}")

            # Disconnect and reconnect with only the needed IP type
            network.disconnect(self.container)
            if self.ipv6:
                network.connect(self.container, ipv6_address=new_ipv6)
            else:
                network.connect(self.container, ipv4_address=new_ipv4)

            self.container.reload()
            updated_info = self.container.attrs["NetworkSettings"]["Networks"].get(self.network_name, {})
            updated_ipv4 = updated_info.get("IPAddress", "Unknown")
            updated_ipv6 = updated_info.get("GlobalIPv6Address", "Unknown")

            if self.ipv6 and current_ipv6 == updated_ipv6:
                raise RuntimeError("IPV6 is the same after network reconnect")
            if not self.ipv6 and current_ipv4 == updated_ipv4:
                raise RuntimeError("IPV4 is the same after network reconnect")

            logging.info(f"Changed container {self.container.name} IPs - New IPv4: {updated_ipv4}, New IPv6: {updated_ipv6}")

        except Exception as e:
            raise RuntimeError(f"Failed to change container IP: {e}")

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
