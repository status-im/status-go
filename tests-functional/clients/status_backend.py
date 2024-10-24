import json
import logging
import time
from datetime import datetime
from json import JSONDecodeError

import jsonschema
import requests

from clients.signals import SignalClient
from conftest import option
from constants import user_1


class RpcClient:

    def __init__(self, rpc_url, client=requests.Session()):
        self.client = client
        self.rpc_url = rpc_url

    def _check_decode_and_key_errors_in_response(self, response, key):
        try:
            return response.json()[key]
        except json.JSONDecodeError:
            raise AssertionError(
                f"Invalid JSON in response: {response.content}")
        except KeyError:
            raise AssertionError(
                f"Key '{key}' not found in the JSON response: {response.content}")

    def verify_is_valid_json_rpc_response(self, response, _id=None):
        assert response.status_code == 200, f"Got response {response.content}, status code {response.status_code}"
        assert response.content
        self._check_decode_and_key_errors_in_response(response, "result")

        if _id:
            try:
                if _id != response.json()["id"]:
                    raise AssertionError(
                        f"got id: {response.json()['id']} instead of expected id: {_id}"
                    )
            except KeyError:
                raise AssertionError(f"no id in response {response.json()}")
        return response

    def verify_is_json_rpc_error(self, response):
        assert response.status_code == 200
        assert response.content
        self._check_decode_and_key_errors_in_response(response, "error")

    def rpc_request(self, method, params=[], request_id=13, url=None):
        url = url if url else self.rpc_url
        data = {"jsonrpc": "2.0", "method": method, "id": request_id}
        if params:
            data["params"] = params
        logging.info(f"Sending POST request to url {url} with data: {json.dumps(data, sort_keys=True, indent=4)}")
        response = self.client.post(url, json=data)
        try:
            logging.info(f"Got response: {json.dumps(response.json(), sort_keys=True, indent=4)}")
        except JSONDecodeError:
            logging.info(f"Got response: {response.content}")
        return response

    def rpc_valid_request(self, method, params=[], _id=None, url=None):
        response = self.rpc_request(method, params, _id, url)
        self.verify_is_valid_json_rpc_response(response, _id)
        return response
            
    def verify_json_schema(self, response, method):
        with open(f"{option.base_dir}/schemas/{method}", "r") as schema:
            jsonschema.validate(instance=response,
                                schema=json.load(schema))


class StatusBackend(RpcClient, SignalClient):

    def __init__(self, await_signals=list()):

        self.api_url = f"{option.rpc_url_status_backend}/statusgo"
        self.ws_url = f"{option.ws_url_status_backend}"
        self.rpc_url = f"{option.rpc_url_status_backend}/statusgo/CallRPC"

        RpcClient.__init__(self, self.rpc_url)
        SignalClient.__init__(self, self.ws_url, await_signals)

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
            assert not response.json()["error"]
        except json.JSONDecodeError:
            raise AssertionError(
                f"Invalid JSON in response: {response.content}")
        except KeyError:
            pass

    def api_valid_request(self, method, data):
        response = self.api_request(method, data)
        self.verify_is_valid_api_response(response)
        return response

    def init_status_backend(self, data_dir="/"):
        method = "InitializeApplication"
        data = {
            "dataDir": data_dir
        }
        return self.api_valid_request(method, data)

    def create_account_and_login(self, display_name="Mr_Meeseeks", password=user_1.password):
        data_dir = f"dataDir_{datetime.now().strftime('%Y%m%d_%H%M%S')}"
        method = "CreateAccountAndLogin"
        data = {
            "rootDataDir": data_dir,
            "kdfIterations": 256000,
            "displayName": display_name,
            "password": password,
            "customizationColor": "primary"
        }
        return self.api_valid_request(method, data)

    def restore_account_and_login(self, display_name="Mr_Meeseeks", user=user_1):
        method = "RestoreAccountAndLogin"
        data = {
            "rootDataDir": "/",
            "displayName": display_name,
            "password": user.password,
            "mnemonic": user.passphrase,
            "customizationColor": "blue",
            "testNetworksEnabled": True,
            "networkId": 31337,
            "networksOverride": [
                {
                    "ChainID": 31337,
                    "ChainName": "Anvil",
                    "DefaultRPCURL": "http://anvil:8545",
                    "RPCURL": "http://anvil:8545",
                    "ShortName": "eth",
                    "NativeCurrencyName": "Ether",
                    "NativeCurrencySymbol": "ETH",
                    "NativeCurrencyDecimals": 18,
                    "IsTest": False,
                    "Layer": 1,
                    "Enabled": True
                }
            ]
        }
        return self.api_valid_request(method, data)

    def restore_account_and_wait_for_rpc_client_to_start(self, timeout=60):
        self.restore_account_and_login()
        start_time = time.time()
        # ToDo: change this part for waiting for `node.login` signal when websockets are migrated to StatusBackend
        while time.time() - start_time <= timeout:
            try:
                self.rpc_valid_request(method='accounts_getKeypairs')
                return
            except AssertionError:
                time.sleep(3)
        raise TimeoutError(f"RPC client was not started after {timeout} seconds")

    def start_messenger(self, params=[]):
        method = "wakuext_startMessenger"
        response = self.rpc_request(method, params)
        json_response = response.json()

        if 'error' in json_response:
            assert json_response['error']['code'] == -32000
            assert json_response['error']['message'] == "messenger already started"
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
