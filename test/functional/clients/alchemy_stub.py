import json
import logging
import threading
from http.server import BaseHTTPRequestHandler, HTTPServer

ZERO_HASH_PREFIX = "0x" + "00" * 31


def _transfer(n, category, sender, recipient, raw_value=None, contract=None, decimal=None, token_id=None, asset="ETH"):
    tx_hash = f"{ZERO_HASH_PREFIX}{n:02x}"
    return {
        "blockNum": hex(0x100 + n),
        "uniqueId": f"{tx_hash}:{'external' if category == 'external' else 'log:1'}",
        "hash": tx_hash,
        "from": sender,
        "to": recipient,
        "value": None,
        "erc1155Metadata": None,
        "tokenId": token_id,
        "asset": asset,
        "category": category,
        "rawContract": {"value": raw_value, "address": contract, "decimal": decimal},
        "metadata": {"blockTimestamp": f"2024-01-{n:02d}T00:00:00.000Z"},
    }


PEER = "0x0adb6caa256a5375c638c00e2ff80a9ac1b2d3a7"
TOKEN = "0x744d70fdbe2ba4cf95131626614a1763df805b9e"
NFT = "0x9f64932be34d5d897c4253d17707b50921f372b6"


def transfers_for(address, outgoing):
    """Transfers the stub returns for one direction of one address."""
    if outgoing:
        return [_transfer(2, "erc20", address, PEER, raw_value="0x1e240", contract=TOKEN, decimal="0x6", asset="TKN")]
    return [
        _transfer(1, "external", PEER, address, raw_value="0xde0b6b3a7640000", decimal="0x12"),
        _transfer(3, "erc721", PEER, address, contract=NFT, token_id="0x33", asset="NFT"),
    ]


class AlchemyRequestHandler(BaseHTTPRequestHandler):
    """Answers eth_blockNumber and alchemy_getAssetTransfers like an Alchemy RPC endpoint."""

    requests = []

    def log_message(self, format, *args):
        logging.debug(f"alchemy stub request: {format % args}")

    def _result(self, request, result):
        body = json.dumps({"jsonrpc": "2.0", "id": request.get("id"), "result": result}).encode("utf-8")
        self.send_response(200)
        self.send_header("Content-type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def _error(self, request, message):
        body = json.dumps({"jsonrpc": "2.0", "id": request.get("id"), "error": {"code": -32601, "message": message}}).encode("utf-8")
        self.send_response(200)
        self.send_header("Content-type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_POST(self):
        content_length = int(self.headers["Content-Length"])
        request = json.loads(self.rfile.read(content_length).decode("utf-8"))
        if isinstance(request, list):
            self._error({}, "batch requests are not supported")
            return

        self.__class__.requests.append(request)
        method = request.get("method")
        if method == "eth_blockNumber":
            self._result(request, hex(0x1000))
        elif method == "alchemy_getAssetTransfers":
            params = request["params"][0]
            outgoing = params.get("fromAddress") is not None
            address = (params.get("fromAddress") or params.get("toAddress")).lower()
            transfers = transfers_for(address, outgoing)
            categories = set(params.get("category", []))
            self._result(request, {"transfers": [t for t in transfers if t["category"] in categories], "pageKey": ""})
        else:
            self._error(request, f"method {method} is not supported")


class AlchemyStub:
    def __init__(self, address="0.0.0.0", port=0):
        AlchemyRequestHandler.requests = []
        self.server = HTTPServer((address, port), AlchemyRequestHandler)
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.thread.start()

    @property
    def port(self):
        return self.server.server_port

    @property
    def requests(self):
        return AlchemyRequestHandler.requests

    def close(self):
        self.server.shutdown()
        self.server.server_close()
