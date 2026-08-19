import json
import logging
from json import JSONDecodeError

import requests


class ApiError(Exception):
    def __init__(self, message, *, method=None, payload=None):
        super().__init__(message)
        self.method = method
        self.payload = payload


class ApiHTTPError(ApiError):
    def __init__(self, message, *, method=None, status=None, payload=None):
        super().__init__(message, method=method, payload=payload)
        self.status = status


class ApiDecodeError(ApiError):
    pass


class ApiResponseError(ApiError):
    pass


class ApiClient:
    def __init__(self, api_url, client=requests.Session()):
        self.client = client
        self.api_url = api_url

    def method_url(self, method):
        return f"{self.api_url}/{method}"

    def api_request(self, method, data, url=None, quiet=False, **kwargs):
        url = url if url else self.api_url
        url = f"{url}/{method}" if method else url
        if not quiet:
            logging.debug(f"Sending POST request to url {url} with data: {json.dumps(data, sort_keys=True)}")
        response = self.client.post(url, json=data, **kwargs)

        if response.status_code != 200:
            raise ApiHTTPError(
                f"HTTP {response.status_code}",
                method=method,
                status=response.status_code,
                payload=getattr(response, "text", None),
            )

        if not response.content:
            raise ApiHTTPError("Empty response body", method=method, status=response.status_code)

        if not quiet:
            logging.debug(f"Got response: {response.content}")
        return response

    def api_request_json(self, method, data, **kwargs):
        response = self.api_request(method, data, **kwargs)
        try:
            json_response = response.json()
        except JSONDecodeError:
            raise ApiDecodeError("Invalid JSON in response", method=method, payload=response.content)

        err = json_response.get("error")
        if err:
            raise ApiResponseError(str(err), method=method, payload=json_response)

        return json_response
