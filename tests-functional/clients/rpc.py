import requests
from clients.api import ApiClient


class RpcClient(ApiClient):

    def __init__(self, client=requests.Session()):
        self.client = client
        self._request_id = 0

    @property
    def request_id(self) -> int:
        self._request_id += 1
        return self._request_id

    def validate_json_rpc_response(self, response, _id):
        # Must contain exactly one of 'result' or 'error'
        has_result = "result" in response
        has_error = "error" in response

        if not (has_result ^ has_error):  # True only if exactly one is True
            raise AssertionError(f"Invalid structure: must contain exactly one of 'result' or 'error', got: {response}")

        try:
            if _id != response["id"]:
                raise AssertionError(f"got id: {response['id']} instead of expected id: {_id}")
        except KeyError:
            raise AssertionError(f"no id in response {response}")

        return response

    def rpc_valid_request(self, method, params=None):
        request_id = self.request_id

        if params is None:
            params = []
        data = {"jsonrpc": "2.0", "method": method, "id": request_id}
        if params:
            data["params"] = params
        response = self.api_request_json("CallRPC", data)
        self.validate_json_rpc_response(response, request_id)
        return response.get("result")
