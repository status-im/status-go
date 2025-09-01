import json
import requests
from tenacity import retry, stop_after_delay, wait_fixed

from clients.api import ApiClient


class RpcClient(ApiClient):

    def __init__(self, rpc_url, client=requests.Session()):
        self.client = client
        self.rpc_url = rpc_url
        self.request_counter = 0

    def _check_decode_and_key_errors_in_response(self, response, key):
        try:
            data = response.json()
        except json.JSONDecodeError:
            raise AssertionError(f"Invalid JSON in response: {response.content}")

        if key in data:
            # If 'result' is present, 'error' must NOT be present
            if key == "result" and "error" in data:
                raise AssertionError(f"Invalid structure: both 'result' and 'error' present in response: {data}")
            return

        # Allow missing 'result' if 'error' is present
        if key == "result" and "error" in data:
            return
        raise AssertionError(f"Key '{key}' not found in the JSON response: {response.content}")

    def verify_is_valid_json_rpc_response(self, response, _id=None):
        self._check_decode_and_key_errors_in_response(response, "result")
        if _id:
            try:
                if _id != response.json()["id"]:
                    raise AssertionError(f"got id: {response.json()['id']} instead of expected id: {_id}")
            except KeyError:
                raise AssertionError(f"no id in response {response.json()}")
        return response

    def verify_is_json_rpc_error(self, response):
        assert response.status_code == 200
        assert response.content
        self._check_decode_and_key_errors_in_response(response, "error")

    @retry(stop=stop_after_delay(10), wait=wait_fixed(0.5), reraise=True)
    def rpc_valid_request(self, method, params=None, _id=None, url=None, enable_logging=True):
        if not _id:
            request_id = self.request_counter
            self.request_counter += 1
        else:
            request_id = _id

        if params is None:
            params = []
        url = url if url else self.rpc_url
        data = {"jsonrpc": "2.0", "method": method, "id": request_id}
        if params:
            data["params"] = params
        response = self.api_valid_request("", data, url=url)
        self.verify_is_valid_json_rpc_response(response, request_id)
        return response
