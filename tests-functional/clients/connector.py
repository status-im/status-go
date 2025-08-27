import json
import logging
import requests
import websocket
from websocket import WebSocket
from websocket import create_connection


class ConnectorClient:
    def __init__(self, url: str):
        self.url = url
        self.ws_conn: WebSocket | None = None
        self.request_id = 0
        self.wrapped_request_id = 0
        self.name = ""

    def connect(self):
        http_url = self.url.replace("ws", "http")
        logging.debug(f"ConnectorClient: sending initial HEAD request to {http_url}")
        response = requests.head(http_url, timeout=5)
        assert response.status_code == 404

        logging.debug(f"ConnectorClient: connecting to {self.url}")
        self.ws_conn = create_connection(self.url, origin="http://localhost")
        assert self.ws_conn is not None
        assert self.ws_conn.sock is not None

        # Use a random name for dApp name
        port = self.ws_conn.sock.getsockname()[1]
        self.name = f"status-go-functional-tests-{port}"

    def disconnect(self):
        if self.ws_conn is not None:
            self.ws_conn.close()

    def get_accounts(self):
        assert self.ws_conn is not None

        self.request_id += 1

        request = {
            "jsonrpc": "2.0",
            "id": self.request_id,
            "name": self.name,
            "url": "http://localhost/",
            "method": "eth_accounts",
        }
        wrapped_request = {
            "jsonrpc": "2.0",
            "id": self.request_id,
            "method": "connector_callRPC",
            "params": [json.dumps(request)],
        }

        self.ws_conn.send(json.dumps(wrapped_request), websocket.ABNF.OPCODE_TEXT)

    def receive(self):
        assert self.ws_conn is not None
        return self.ws_conn.recv()
