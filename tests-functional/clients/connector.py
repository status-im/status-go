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
        self._request_id = 0
        self.wrapped_request_id = 0
        self.name = ""

    @property
    def request_id(self) -> int:
        self._request_id += 1
        return self._request_id

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
        self._send("eth_accounts")

    def get_chain_id(self):
        self._send("eth_chainId")

    def get_block_number(self):
        self._send("eth_blockNumber")

    def revoke_permissions(self):
        self._send("wallet_revokePermissions")

    def _send(self, method, params=None):
        assert self.ws_conn is not None

        request_id = self.request_id
        request = {
            "jsonrpc": "2.0",
            "id": request_id,
            "name": self.name,
            "url": "http://localhost/",
            "method": method,
        }
        if params is not None:
            request["params"] = params

        wrapped_request = {
            "jsonrpc": "2.0",
            "id": request_id,
            "method": "connector_callRPC",
            "params": [json.dumps(request)],
        }

        logging.debug(f"Sending Connector request with data: {json.dumps(wrapped_request, sort_keys=True)}")
        self.ws_conn.send(json.dumps(wrapped_request), websocket.ABNF.OPCODE_TEXT)

    def receive(self):
        assert self.ws_conn is not None

        response = self.ws_conn.recv()
        logging.debug(f"Got Connector response: {json.dumps(response, sort_keys=True)}")

        return json.loads(response)
