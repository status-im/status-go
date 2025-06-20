import json
import logging
import threading
import time
from http.server import BaseHTTPRequestHandler, HTTPServer

import requests


class GorushRequestHandler(BaseHTTPRequestHandler):
    """HTTP request handler for gorush stub"""

    # Class-level variable to store requests for debugging
    push_requests = []

    def log_message(self, format, *args):
        logging.debug(f"gorush stub request: {format % args}")

    def _set_response(self, status_code=200, content_type="application/json"):
        self.send_response(status_code)
        self.send_header("Content-type", content_type)
        self.end_headers()

    def do_GET(self):
        if self.path == "/healthz":
            self._set_response()
            self.wfile.write(json.dumps({"status": "ok"}).encode("utf-8"))
        else:
            self._set_response(404)
            self.wfile.write(json.dumps({"error": "Not found"}).encode("utf-8"))

    def do_POST(self):
        content_length = int(self.headers["Content-Length"])
        post_data = self.rfile.read(content_length).decode("utf-8")

        if self.path == "/api/push":
            # Store request for debugging
            self.__class__.push_requests.append(post_data)

            # Decode post_data
            try:
                request = json.loads(post_data)
            except json.JSONDecodeError:
                self._set_response(400)
                self.wfile.write(json.dumps({"error": "Invalid JSON"}).encode("utf-8"))
                return

            # Return a successful response
            self._set_response()
            response = {
                "success": "ok",
                "counts": {
                    "total": len(request["notifications"]),
                },
            }
            self.wfile.write(json.dumps(response).encode("utf-8"))
        else:
            self._set_response(404)
            self.wfile.write(json.dumps({"error": "Not found"}).encode("utf-8"))


class GorushStub:
    """Client for interacting with gorush-stub service"""

    def __init__(self, address="localhost", port=8088):
        """Initialize GorushStub client

        Args:
            port: Port to expose gorush-stub on the host
        """
        # Setup logging
        self.logger = logging.getLogger(__name__)

        # Initialize the HTTP server
        self.base_url = ""
        self.server = None
        self.server_thread = None

        # Clear previous debug requests
        GorushRequestHandler.push_requests = []

        # Create and start the HTTP server
        self.server = HTTPServer((address, port), GorushRequestHandler)
        self.server_thread = threading.Thread(target=self.server.serve_forever)
        self.server_thread.daemon = True
        self.server_thread.start()

        # Create base URL for API requests
        self.base_url = f"http://{address}:{self.server.server_port}"

        # Wait for the server to start
        self._wait_for_service()
        self.logger.info(f"gorush-stub initialized at {self.base_url}")

    def _wait_for_service(self, timeout=30, interval=1):
        """Wait for gorush-stub service to be available

        Args:
            timeout: Maximum time to wait in seconds
            interval: Interval between attempts in seconds

        Returns:
            bool: True if service is available, False otherwise
        """
        start_time = time.time()
        while time.time() - start_time < timeout:
            try:
                response = requests.get(f"{self.base_url}/healthz")
                if response.status_code == 200:
                    return True
            except requests.RequestException:
                pass
            time.sleep(interval)

        self.logger.error(f"gorush-stub service not available after {timeout} seconds")
        return False

    def get_requests(self):
        """Get debug requests from gorush-stub

        Returns:
            list: List of recorded requests
        """
        # Get a copy of the current push_requests
        requests = GorushRequestHandler.push_requests.copy()

        # Clear the original list
        GorushRequestHandler.push_requests = []
        return requests

    def wait_for_requests(self, timeout=10):
        start_time = time.time()
        push_requests = []

        while len(push_requests) == 0:
            if time.time() - start_time > timeout:
                assert False, "Timeout waiting for push notifications requests"

            time.sleep(1)

            for req in self.get_requests():
                for notification in json.loads(req)["notifications"]:
                    push_requests.append(notification)

        return push_requests

    def close(self):
        """Stop the gorush-stub HTTP server"""
        if not self.server:
            return True

        try:
            self.server.shutdown()
            self.server.server_close()
            self.server = None
            self.server_thread = None
            return True
        except Exception as e:
            self.logger.error(f"Failed to close gorush-stub server: {e}")
            return False
