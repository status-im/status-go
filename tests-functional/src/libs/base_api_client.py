import requests
import json
from tenacity import retry, stop_after_delay, wait_fixed
from src.libs.custom_logger import get_custom_logger

logger = get_custom_logger(__name__)

class BaseAPIClient:
    def __init__(self, base_url):
        self.base_url = base_url

    @retry(stop=stop_after_delay(10), wait=wait_fixed(0.5), reraise=True)
    def send_post_request(self, endpoint, payload=None, headers=None, timeout=10):
        if headers is None:
            headers = {"Content-Type": "application/json"}
        if payload is None:
            payload = {}

        url = f"{self.base_url}/{endpoint}"
        logger.info(f"Sending POST request to {url} with payload: {json.dumps(payload)}")
        try:
            response = requests.post(url, headers=headers, data=json.dumps(payload), timeout=timeout)
            response.raise_for_status()
            logger.info(f"Response received: {response.status_code} - {response.text}")
            return response.json()
        except requests.exceptions.RequestException as e:
            logger.error(f"Request to {url} failed: {str(e)}")
            raise
