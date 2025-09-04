import requests
from clients.api import ApiClient


class RpcClient(ApiClient):

    def __init__(self, client=requests.Session()):
        self.client = client
        self.request_counter = 0

    def validate_json_rpc_response(self, response, _id=None):
        # Must contain exactly one of 'result' or 'error'
        has_result = "result" in response
        has_error = "error" in response

        if not (has_result ^ has_error):  # True only if exactly one is True
            raise AssertionError(f"Invalid structure: must contain exactly one of 'result' or 'error', got: {response}")

        if _id:
            try:
                if _id != response["id"]:
                    raise AssertionError(f"got id: {response['id']} instead of expected id: {_id}")
            except KeyError:
                raise AssertionError(f"no id in response {response}")
        return response

    def rpc_valid_request(self, method, params=None, _id=None):
        if not _id:
            request_id = self.request_counter
            self.request_counter += 1
        else:
            request_id = _id

        if params is None:
            params = []
        data = {"jsonrpc": "2.0", "method": method, "id": request_id}
        if params:
            data["params"] = params
        response = self.api_request_json("CallRPC", data)
        self.validate_json_rpc_response(response, request_id)
        return response
