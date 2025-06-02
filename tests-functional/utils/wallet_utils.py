import json
import logging
import jsonschema
import resources.constants as constants
from clients.signals import SignalType, WalletEventType

from conftest import option


def verify_json_schema(response, method):
    with open(f"{option.base_dir}/schemas/{method}", "r") as schema:
        jsonschema.validate(instance=response, schema=json.load(schema))


def get_suggested_routes(rpc_client, **kwargs):
    method = "wallet_getSuggestedRoutesAsync"
    required_params = ["uuid", "sendType", "addrFrom", "addrTo", "amountIn", "tokenID", "gasFeeMode"]
    input_params = {}

    for key, new_value in kwargs.items():
        input_params[key] = new_value

    for key in required_params:
        if key not in input_params:
            logging.info(f"Warning: The key '{key}' does not exist in the input_params parameters and will be ignored.")

    params = [input_params]

    rpc_client.prepare_wait_for_signal("wallet.suggested.routes", 1)
    _ = rpc_client.rpc_valid_request(method, params)

    routes_signal = rpc_client.wait_for_signal("wallet.suggested.routes")
    routes = routes_signal["event"]

    return routes


def build_transactions_from_route(rpc_client, uuid):
    method = "wallet_buildTransactionsFromRoute"
    if uuid is None or uuid == "":
        logging.info(f"Warning: provided '{uuid}' does not exist or is empty")

    params = [uuid]
    _ = rpc_client.rpc_valid_request(method, params)

    wallet_router_sign_transactions_signal = rpc_client.wait_for_signal("wallet.router.sign-transactions")
    wallet_router_sign_transactions = wallet_router_sign_transactions_signal["event"]

    assert "signingDetails" in wallet_router_sign_transactions
    assert wallet_router_sign_transactions["signingDetails"]["signOnKeycard"] is False
    transaction_hashes = wallet_router_sign_transactions["signingDetails"]["hashes"]

    assert transaction_hashes, "Transaction hashes are empty!"

    return wallet_router_sign_transactions


def sign_messages(rpc_client, hashes, address):
    tx_signatures = {}

    for hash in hashes:

        method = "wallet_signMessage"
        params = [hash, address, option.password]

        response = rpc_client.rpc_valid_request(method, params)

        result = response.json().get("result")
        assert result and result.startswith("0x"), f"Invalid transaction signature for hash {hash}: {result}"

        tx_signature = result[2:]

        signature = {
            "r": tx_signature[:64],
            "s": tx_signature[64:128],
            "v": tx_signature[128:],
        }

        tx_signatures[hash] = signature
    return tx_signatures


def check_tx_details(rpc_client, tx_hash, network_id, address_to, expected_amount_in):
    method = "ethclient_transactionByHash"
    params = [network_id, tx_hash]

    response = rpc_client.rpc_valid_request(method, params)
    tx_details = response.json()["result"]["tx"]

    assert tx_details["value"] == expected_amount_in
    assert tx_details["to"].upper() == address_to.upper()


def check_fees(fee_mode, base_fee, max_priority_fee_per_gas, max_fee_per_gas, suggested_fee_levels):
    assert base_fee.startswith("0x")
    assert max_priority_fee_per_gas.startswith("0x")
    assert max_fee_per_gas.startswith("0x")

    base_fee_int = int(base_fee, 16)
    max_priority_fee_per_gas_int = int(max_priority_fee_per_gas, 16)
    max_fee_per_gas_int = int(max_fee_per_gas, 16)

    low_max_fee_per_gas = int(suggested_fee_levels["low"], 16)
    low_priority_max_fee_per_gas = int(suggested_fee_levels["lowPriority"], 16)
    medium_max_fee_per_gas = int(suggested_fee_levels["medium"], 16)
    medium_priority_max_fee_per_gas = int(suggested_fee_levels["mediumPriority"], 16)
    high_max_fee_per_gas = int(suggested_fee_levels["high"], 16)
    high_priority_max_fee_per_gas = int(suggested_fee_levels["highPriority"], 16)

    if fee_mode == constants.gas_fee_mode_low:
        assert max_fee_per_gas_int == low_max_fee_per_gas
        assert max_priority_fee_per_gas_int == low_priority_max_fee_per_gas
        assert base_fee_int + max_priority_fee_per_gas_int == max_fee_per_gas_int
    elif fee_mode == constants.gas_fee_mode_medium:
        # calculate variadic fees from high max fees per gas
        variadic_fee = high_max_fee_per_gas - high_priority_max_fee_per_gas - 2 * base_fee_int

        assert max_fee_per_gas_int == medium_max_fee_per_gas
        assert max_priority_fee_per_gas_int == medium_priority_max_fee_per_gas
        assert base_fee_int + variadic_fee + max_priority_fee_per_gas_int == max_fee_per_gas_int
    elif fee_mode == constants.gas_fee_mode_high:
        # calculate variadic fees from medium max fees per gas
        variadic_fee = medium_max_fee_per_gas - medium_priority_max_fee_per_gas - base_fee_int

        assert max_fee_per_gas_int == high_max_fee_per_gas
        assert max_priority_fee_per_gas_int == high_priority_max_fee_per_gas
        assert 2 * base_fee_int + variadic_fee + max_priority_fee_per_gas_int == max_fee_per_gas_int
    elif fee_mode == constants.gas_fee_mode_custom:
        assert base_fee_int + max_priority_fee_per_gas_int == max_fee_per_gas_int
    else:
        assert False, "Invalid gas fee mode"


def check_fees_for_path(path_name, gas_fee_mode, check_approval, route):
    for path_tx in route:
        if path_tx["ProcessorName"] != path_name:
            continue
        if check_approval:
            assert path_tx["ApprovalRequired"]
            check_fees(
                gas_fee_mode,
                path_tx["ApprovalBaseFee"],
                path_tx["ApprovalPriorityFee"],
                path_tx["ApprovalMaxFeesPerGas"],
                path_tx["SuggestedLevelsForMaxFeesPerGas"],
            )
            return
        check_fees(
            gas_fee_mode, path_tx["TxBaseFee"], path_tx["TxPriorityFee"], path_tx["TxMaxFeesPerGas"], path_tx["SuggestedLevelsForMaxFeesPerGas"]
        )


def send_router_transactions_with_signatures(rpc_client, uuid, tx_signatures):
    method = "wallet_sendRouterTransactionsWithSignatures"
    params = [{"uuid": uuid, "Signatures": tx_signatures}]
    rpc_client.prepare_wait_for_signal(
        SignalType.WALLET.value,
        1,
        lambda signal: signal["event"]["type"] == WalletEventType.TRANSACTIONS_PENDING_TRANSACTION_STATUS_CHANGED.value,
    )
    _ = rpc_client.rpc_valid_request(method, params)
    event_response = rpc_client.wait_for_signal(SignalType.WALLET.value)["event"]
    tx_status = json.loads(event_response["message"].replace("'", '"'))

    assert tx_status["status"] == "Success"

    return tx_status


def send_router_transaction(rpc_client, **kwargs):
    routes = get_suggested_routes(rpc_client, **kwargs)
    assert "Best" in routes, f"No best route found: {routes}"

    build_tx = build_transactions_from_route(rpc_client, kwargs.get("uuid"))

    tx_signatures = sign_messages(rpc_client, build_tx["signingDetails"]["hashes"], kwargs.get("addrFrom"))

    tx_status = send_router_transactions_with_signatures(rpc_client, routes["Uuid"], tx_signatures)
    return {
        "routes": routes,
        "build_tx": build_tx,
        "tx_signatures": tx_signatures,
        "tx_status": tx_status,
    }
