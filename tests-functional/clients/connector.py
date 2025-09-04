import json
import logging
import requests
from typing import Any, Dict, List
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

    def eth_chain_id(self):
        self._send("eth_chainId")

    def eth_accounts(self):
        self._send("eth_accounts")

    def eth_request_accounts(self):
        self._send("eth_requestAccounts")

    def eth_block_number(self):
        self._send("eth_blockNumber")

    def eth_get_balance(self, address: str):
        self._send("eth_getBalance", [address, "latest"])

    def eth_get_transaction_count(self, address: str):
        self._send("eth_getTransactionCount", [address, "latest"])

    def eth_call(self, call_object: Dict[str, Any]):
        params: List[Any] = [call_object, "latest"]
        self._send("eth_call", params)

    def eth_estimate_gas(self, tx_object: Dict[str, Any]):
        params: List[Any] = [tx_object]
        self._send("eth_estimateGas", params)

    def eth_get_transaction_receipt(self, tx_hash: str):
        self._send("eth_getTransactionReceipt", [tx_hash])

    def eth_send_transaction(self, tx_object: Dict[str, Any]):
        self._send("eth_sendTransaction", [tx_object])

    def wallet_switch_ethereum_chain(self, chain_id: int):
        hex_chain_id = hex(chain_id)
        self._send("wallet_switchEthereumChain", [{"chainId": hex_chain_id}])

    def wallet_revoke_permissions(self):
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
