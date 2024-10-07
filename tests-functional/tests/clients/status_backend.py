import jsonschema
import json
import requests
from datetime import datetime
from conftest import option, user_1
from clients.signals import SignalClient


class RpcClient:

    def __init__(
        self, rpc_url, client=requests.Session()
    ):

        self.client = client
        self.rpc_url = rpc_url

    def _try_except_JSONDecodeError_KeyError(self, response, key: str):
        try:
            return response.json()[key]
        except json.JSONDecodeError:
            raise AssertionError(
                f"Invalid JSON in response: {response.content}")
        except KeyError:
            raise AssertionError(
                f"Key '{key}' not found in the JSON response: {response.content}")

    def verify_is_valid_json_rpc_response(self, response, _id=None):
        assert response.status_code == 200
        assert response.content
        self._try_except_JSONDecodeError_KeyError(response, "result")

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
        self._try_except_JSONDecodeError_KeyError(response, "error")

    def rpc_request(self, method, params=[], _id=None, url=None):
        url = url if url else self.rpc_url
        data = {"jsonrpc": "2.0", "method": method}
        if params:
            data["params"] = params
        data["id"] = _id if _id else 13
        response = self.client.post(url, json=data)

        return response

    def rpc_valid_request(self, method, params=[], _id=None, url=None):
        response = self.rpc_request(method, params, _id, url)
        self.verify_is_valid_json_rpc_response(response, _id)
        return response

    def verify_json_schema(self, response, method):
        with open(f"{option.base_dir}/schemas/{method}", "r") as schema:
            jsonschema.validate(instance=response.json(),
                                schema=json.load(schema))


class StatusBackend(RpcClient, SignalClient):

    def __init__(self, await_signals):

        self.api_url = f"{option.rpc_url_status_backend}/statusgo"
        self.ws_url = f"{option.ws_url_status_backend}"
        self.rpc_url = f"{option.rpc_url_status_backend}/statusgo/CallRPC"

        RpcClient.__init__(self, self.rpc_url)
        SignalClient.__init__(self, self.ws_url, await_signals)

    def api_request(self, method, data, url=None):
        url = url if url else self.api_url
        url = f"{url}/{method}"
        response = requests.post(url, json=data)
        return response

    def verify_is_valid_api_response(self, response):
        assert response.status_code == 200
        assert response.content
        try:
            assert not response.json()["error"]
        except json.JSONDecodeError:
            raise AssertionError(
                f"Invalid JSON in response: {response.content}")
        except KeyError:
            pass

    def api_valid_request(self, method, data, url=None):
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
