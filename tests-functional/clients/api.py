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
        url = f"{url}/{method}"
        if not quiet:
            logging.debug(f"Sending POST request to url {url} with data: {json.dumps(data, sort_keys=True)}")
        response = self.client.post(url, json=data)
        if not quiet:
            logging.debug(f"Got response: {response.content}")
        return response

    def verify_is_valid_api_response(self, response):
        assert response.status_code == 200, f"Got response {response.content}, status code {response.status_code}"
        assert response.content
        logging.debug(f"Got response: {response.content}")
        try:
            error = response.json()["error"]
            assert not error, f"Error: {error}"
        except JSONDecodeError:
            raise AssertionError(f"Invalid JSON in response: {response.content}")
        except KeyError:
            # No 'error' key in response JSON, treat as valid
            pass

    def api_valid_request(self, method, data, url=None):
        response = self.api_request(method, data, url=url)
        self.verify_is_valid_api_response(response)
        return response
