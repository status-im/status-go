import json
import logging
import jsonschema
import requests
from tenacity import retry, stop_after_delay, wait_fixed
from conftest import option
from json import JSONDecodeError

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

    @retry(stop=stop_after_delay(10), wait=wait_fixed(0.5), reraise=True)
    def rpc_request(self, method, params=[], request_id=13, url=None):
        url = url if url else self.rpc_url
        data = {"jsonrpc": "2.0", "method": method, "id": request_id}
        if params:
            data["params"] = params
        logging.info(f"Sending POST request to url {url} with data: {json.dumps(data, sort_keys=True, indent=4)}")
        response = self.client.post(url, json=data)
        try:
            resp_json = response.json()
            logging.info(f"Got response: {json.dumps(resp_json, sort_keys=True, indent=4)}")
            if resp_json.get("error"):
                assert "JSON-RPC client is unavailable" != resp_json["error"]
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
