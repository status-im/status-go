import json
import logging
import requests
from json import JSONDecodeError


class ApiClient:
    def __init__(self, api_url, client=requests.Session()):
        self.client = client
        self.api_url = api_url

    def api_request(self, method, data, url=None, quiet=False):
        url = url if url else self.api_url
        url = f"{url}/{method}" if method else url
        if not quiet:
            logging.debug(f"Sending POST request to url {url} with data: {json.dumps(data, sort_keys=True)}")
        response = self.client.post(url, json=data)
        assert response.status_code == 200, f"Got response {response.content}, status code {response.status_code}"
        assert response.content

        if not quiet:
            logging.debug(f"Got response: {response.content}")
        return response

    def api_request_json(self, method, data, check_error=True):
        response = self.api_request(method, data)
        try:
            json_response = response.json()
            if check_error:
                assert not json_response.get("error", None), f"Found error in response: {json_response}"
            return json_response
        except JSONDecodeError:
            raise AssertionError(f"Invalid JSON in response: {response.content}")
