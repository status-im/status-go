import json
import logging
import jsonschema
import uuid
import threading
import time

from conftest import option
from constants import user_1, user_2

from clients.signals import SignalClient

def verify_json_schema(response, method):
    with open(f"{option.base_dir}/schemas/{method}", "r") as schema:
        jsonschema.validate(instance=response,
                            schema=json.load(schema))
        
def get_suggested_routes(rpc_client, signal_client, **kwargs):
    _uuid = str(uuid.uuid4())
    amount_in = "0xde0b6b3a7640000"

    method = "wallet_getSuggestedRoutesAsync"
    input_params = {
        "uuid": _uuid,
        "sendType": 0,
        "addrFrom": user_1.address,
        "addrTo": user_2.address,
        "amountIn": amount_in,
        "amountOut": "0x0",
        "tokenID": "ETH",
        "tokenIDIsOwnerToken": False,
        "toTokenID": "",
        "disabledFromChainIDs": [10, 42161],
        "disabledToChainIDs": [10, 42161],
        "gasFeeMode": 1,
        "fromLockedAmount": {}
    }
    for key, new_value in kwargs.items():
        if key in input_params:
            input_params[key] = new_value
        else:
            logging.info(
                f"Warning: The key '{key}' does not exist in the input_params parameters and will be ignored.")
    params = [input_params]

    signal_client.prepare_wait_for_signal("wallet.suggested.routes", 1)
    _ = rpc_client.rpc_valid_request(method, params)

    routes = signal_client.wait_for_signal("wallet.suggested.routes")
    assert routes['event']['Uuid'] == _uuid

    return routes['event']

def build_transactions_from_route(rpc_client, signal_client, uuid, **kwargs):
    method = "wallet_buildTransactionsFromRoute"
    build_tx_params = {
        "uuid": uuid,
        "slippagePercentage": 0
    }
    for key, new_value in kwargs.items():
        if key in build_tx_params:
            build_tx_params[key] = new_value
        else:
            logging.info(
                f"Warning: The key '{key}' does not exist in the build_tx_params parameters and will be ignored.")
    params = [build_tx_params]
    _ = rpc_client.rpc_valid_request(method, params)

    wallet_router_sign_transactions = signal_client.wait_for_signal("wallet.router.sign-transactions")

    assert wallet_router_sign_transactions['event']['signingDetails']['signOnKeycard'] == False
    transaction_hashes = wallet_router_sign_transactions['event']['signingDetails']['hashes']

    assert transaction_hashes, "Transaction hashes are empty!"

    return wallet_router_sign_transactions['event']

def sign_messages(rpc_client, hashes):
    tx_signatures = {}

    for hash in hashes:

        method = "wallet_signMessage"
        params = [
            hash,
            user_1.address,
            option.password
        ]

        response = rpc_client.rpc_valid_request(method, params)

        if response.json()["result"].startswith("0x"):
            tx_signature = response.json()["result"][2:]

        signature = {
            "r": tx_signature[:64],
            "s": tx_signature[64:128],
            "v": tx_signature[128:]
        }

        tx_signatures[hash] = signature
    return tx_signatures

def send_router_transactions_with_signatures(rpc_client, signal_client, uuid, tx_signatures):
    method = "wallet_sendRouterTransactionsWithSignatures"
    params = [
        {
            "uuid": uuid,
            "Signatures": tx_signatures
        }
    ]
    _ = rpc_client.rpc_valid_request(method, params)

    tx_status = signal_client.wait_for_signal(
        "wallet.transaction.status-changed")

    assert tx_status["event"]["status"] == "Success"

    return tx_status["event"]

def send_router_transaction(rpc_client, signal_client, **kwargs):
    routes = get_suggested_routes(rpc_client, signal_client, **kwargs)
    build_tx = build_transactions_from_route(rpc_client, signal_client, routes['Uuid'])
    tx_signatures = sign_messages(rpc_client, build_tx['signingDetails']['hashes'])
    tx_status = send_router_transactions_with_signatures(rpc_client, signal_client, routes['Uuid'], tx_signatures)
    return {
        "routes": routes,
        "build_tx": build_tx,
        "tx_signatures": tx_signatures,
        "tx_status": tx_status
    }
