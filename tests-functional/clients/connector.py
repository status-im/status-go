import json
import logging
import requests
import websocket
from websocket import WebSocketApp
from websocket import WebSocket
from websocket import create_connection


class ConnectorClient:
    def __init__(self, http_url: str, ws_url: str):
        self.http_url = http_url
        self.ws_url = ws_url
        self.wsapp: WebSocketApp | None = None
        self.ws_conn: WebSocket | None = None
        self.request_id = 0
        self.wrapped_request_id = 0
        self.name = ""

    def connect(self):
        url = self.ws_url.replace("ws", "http")
        logging.info(f"Initial HEAD request to {url}")
        response = requests.head(url, timeout=5)
        assert response.status_code == 404

        logging.info(f"Connecting to {self.ws_url}")
        self.ws_conn = create_connection(self.ws_url, origin="http://localhost")
        assert self.ws_conn is not None
        assert self.ws_conn.sock is not None

        port = self.ws_conn.sock.getsockname()[1]
        self.name = f"status-go-functional-tests-{port}"

    def _connect(self):
        self.wsapp = websocket.WebSocketApp(
            url=self.ws_url,
            on_message=self._on_message,
            on_error=self._on_error,
            on_open=self._on_open,
            on_close=self._on_close,
        )
        self.wsapp.run_forever()

    def _on_message(self, ws, message):
        logging.info(f"ConnectorService message received: {message}")

    def _on_error(self, ws, error):
        logging.error(f"ConnectorService: websocket error: {error}")

    def _on_close(self, ws, close_status_code, close_msg):
        logging.debug(f"ConnectorService: websocket connection closed: {close_status_code}, {close_msg}")

    def _on_open(self, ws):
        logging.debug("ConnectorService: websocket connection opened")

    def disconnect(self):
        if self.wsapp is None:
            return
        self.wsapp.close()

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
