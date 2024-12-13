import json
import logging
import time
import random
import threading
import requests
import docker
import os

from tenacity import retry, stop_after_delay, wait_fixed
from clients.signals import SignalClient
from clients.rpc import RpcClient
from datetime import datetime
from conftest import option
from resources.constants import user_1, DEFAULT_DISPLAY_NAME, USER_DIR


class StatusBackend(RpcClient, SignalClient):

    def __init__(self, await_signals=[]):

        if option.status_backend_url:
            url = option.status_backend_url
        else:
            self.docker_client = docker.from_env()
            host_port = random.choice(option.status_backend_port_range)

            self.container = self._start_container(host_port)
            url = f"http://127.0.0.1:{host_port}"
            option.status_backend_port_range.remove(host_port)

        self.api_url = f"{url}/statusgo"
        self.ws_url = f"{url}".replace("http", "ws")
        self.rpc_url = f"{url}/statusgo/CallRPC"

        RpcClient.__init__(self, self.rpc_url)
        SignalClient.__init__(self, self.ws_url, await_signals)

        self._health_check()

        websocket_thread = threading.Thread(target=self._connect)
        websocket_thread.daemon = True
        websocket_thread.start()

    def _start_container(self, host_port):
        docker_project_name = option.docker_project_name

        timestamp = datetime.now().strftime("%Y%m%d%H%M%S")
        image_name = f"{docker_project_name}-status-backend:latest"
        container_name = f"{docker_project_name}-status-backend-{timestamp}"

        coverage_path = option.codecov_dir if option.codecov_dir else os.path.abspath("./coverage/binary")

        container_args = {
            "image": image_name,
            "detach": True,
            "name": container_name,
            "labels": {"com.docker.compose.project": docker_project_name},
            "entrypoint": [
                "status-backend",
                "--address",
                "0.0.0.0:3333",
            ],
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

        if "FUNCTIONAL_TESTS_DOCKER_UID" in os.environ:
            container_args["user"] = os.environ["FUNCTIONAL_TESTS_DOCKER_UID"]

        container = self.docker_client.containers.run(**container_args)

        network = self.docker_client.networks.get(f"{docker_project_name}_default")
        network.connect(container)

        option.status_backend_containers.append(container.id)
        return container

    def _health_check(self):
        start_time = time.time()
        while True:
            try:
                self.api_valid_request(method="Fleets", data=[])
                break
            except Exception as e:
                if time.time() - start_time > 20:
                    raise Exception(e)
                time.sleep(1)

    def api_request(self, method, data, url=None):
        url = url if url else self.api_url
        url = f"{url}/{method}"
        logging.info(f"Sending POST request to url {url} with data: {json.dumps(data, sort_keys=True, indent=4)}")
        response = requests.post(url, json=data)
        logging.info(f"Got response: {response.content}")
        return response

    def verify_is_valid_api_response(self, response):
        assert response.status_code == 200, f"Got response {response.content}, status code {response.status_code}"
        assert response.content
        logging.info(f"Got response: {response.content}")
        try:
            error = response.json()["error"]
            assert not error, f"Error: {error}"
        except json.JSONDecodeError:
            raise AssertionError(f"Invalid JSON in response: {response.content}")
        except KeyError:
            pass

    def api_valid_request(self, method, data):
        response = self.api_request(method, data)
        self.verify_is_valid_api_response(response)
        return response

    def init_status_backend(self, data_dir=USER_DIR):
        method = "InitializeApplication"
        data = {
            "dataDir": data_dir,
            "logEnabled": True,
            "logLevel": "DEBUG",
            "apiLogging": True,
        }
        return self.api_valid_request(method, data)

    def _set_proxy_credentials(self, data):
        if not "STATUS_BUILD_PROXY_USER" in os.environ:
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

    def create_account_and_login(
        self,
        data_dir=USER_DIR,
        display_name=DEFAULT_DISPLAY_NAME,
        password=user_1.password,
    ):
        method = "CreateAccountAndLogin"
        data = {
            "rootDataDir": data_dir,
            "kdfIterations": 256000,
            "displayName": display_name,
            "password": password,
            "customizationColor": "primary",
            "logEnabled": True,
            "logLevel": "DEBUG",
        }
        data = self._set_proxy_credentials(data)
        return self.api_valid_request(method, data)

    def restore_account_and_login(
        self,
        data_dir=USER_DIR,
        display_name=DEFAULT_DISPLAY_NAME,
        user=user_1,
        network_id=31337,
    ):
        method = "RestoreAccountAndLogin"
        data = {
            "rootDataDir": data_dir,
            "kdfIterations": 256000,
            "displayName": display_name,
            "password": user.password,
            "mnemonic": user.passphrase,
            "customizationColor": "blue",
            "logEnabled": True,
            "logLevel": "DEBUG",
            "testNetworksEnabled": False,
            "networkId": network_id,
            "networksOverride": [
                {
                    "ChainID": network_id,
                    "ChainName": "Anvil",
                    "DefaultRPCURL": "http://anvil:8545",
                    "RPCURL": "http://anvil:8545",
                    "ShortName": "eth",
                    "NativeCurrencyName": "Ether",
                    "NativeCurrencySymbol": "ETH",
                    "NativeCurrencyDecimals": 18,
                    "IsTest": False,
                    "Layer": 1,
                    "Enabled": True,
                }
            ],
        }
        data = self._set_proxy_credentials(data)
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

    def logout(self, user=user_1):
        method = "Logout"
        return self.api_valid_request(method, {})

    def restore_account_and_wait_for_rpc_client_to_start(self, timeout=60):
        self.restore_account_and_login()
        start_time = time.time()
        # ToDo: change this part for waiting for `node.login` signal when websockets are migrated to StatusBackend
        while time.time() - start_time <= timeout:
            try:
                self.rpc_valid_request(method="accounts_getKeypairs")
                return
            except AssertionError:
                time.sleep(3)
        raise TimeoutError(f"RPC client was not started after {timeout} seconds")

    @retry(stop=stop_after_delay(10), wait=wait_fixed(0.5), reraise=True)
    def start_messenger(self, params=[]):
        method = "wakuext_startMessenger"
        response = self.rpc_request(method, params)
        json_response = response.json()

        if "error" in json_response:
            assert json_response["error"]["code"] == -32000
            assert json_response["error"]["message"] == "messenger already started"
            return

        self.verify_is_valid_json_rpc_response(response)

    def start_wallet(self, params=[]):
        method = "wallet_startWallet"
        response = self.rpc_request(method, params)
        self.verify_is_valid_json_rpc_response(response)

    def get_settings(self, params=[]):
        method = "settings_getSettings"
        response = self.rpc_request(method, params)
        self.verify_is_valid_json_rpc_response(response)

    def get_accounts(self, params=[]):
        method = "accounts_getAccounts"
        response = self.rpc_request(method, params)
        self.verify_is_valid_json_rpc_response(response)
        return response.json()

    def get_pubkey(self, display_name):
        response = self.get_accounts()
        accounts = response.get("result", [])
        for account in accounts:
            if account.get("name") == display_name:
                return account.get("public-key")
        raise ValueError(f"Public key not found for display name: {display_name}")

    def send_contact_request(self, contact_id: str, message: str):
        method = "wakuext_sendContactRequest"
        params = [{"id": contact_id, "message": message}]
        response = self.rpc_request(method, params)
        self.verify_is_valid_json_rpc_response(response)
        return response.json()

    def accept_contact_request(self, chat_id: str):
        method = "wakuext_acceptContactRequest"
        params = [{"id": chat_id}]
        response = self.rpc_request(method, params)
        self.verify_is_valid_json_rpc_response(response)
        return response.json()

    def get_contacts(self):
        method = "wakuext_contacts"
        response = self.rpc_request(method)
        self.verify_is_valid_json_rpc_response(response)
        return response.json()

    def send_message(self, contact_id: str, message: str):
        method = "wakuext_sendOneToOneMessage"
        params = [{"id": contact_id, "message": message}]
        response = self.rpc_request(method, params)
        self.verify_is_valid_json_rpc_response(response)
        return response.json()
