import uuid as uuid_lib

import pytest
import logging
import resources.constants as constants

from steps.status_backend import StatusBackendSteps
from clients.signals import SignalType
from utils import wallet_utils


@pytest.mark.rpc
@pytest.mark.transaction
@pytest.mark.wallet
class TestRouter(StatusBackendSteps):
    await_signals = [
        SignalType.NODE_LOGIN.value,
        SignalType.WALLET.value,
        SignalType.WALLET_SUGGESTED_ROUTES.value,
        SignalType.WALLET_ROUTER_SIGN_TRANSACTIONS.value,
        SignalType.WALLET_ROUTER_SENDING_TRANSACTIONS_STARTED.value,
        SignalType.WALLET_ROUTER_TRANSACTIONS_SENT.value,
    ]

    def test_tx_from_route(self):
        uuid = str(uuid_lib.uuid4())
        amount_in = "0xde0b6b3a7640000"

        params = {
            "uuid": uuid,
            "sendType": 0,
            "addrFrom": constants.user_1.address,
            "addrTo": constants.user_2.address,
            "amountIn": amount_in,
            "amountOut": "0x0",
            "tokenID": "ETH",
            "tokenIDIsOwnerToken": False,
            "toTokenID": "",
            "disabledFromChainIDs": [1, 10, 42161],
            "disabledToChainIDs": [1, 10, 42161],
            "gasFeeMode": 1,
        }

        routes = wallet_utils.get_suggested_routes(self.rpc_client, **params)
        assert len(routes["Best"]) > 0
        wallet_router_sign_transactions = wallet_utils.build_transactions_from_route(self.rpc_client, **params)
        transaction_hashes = wallet_router_sign_transactions["signingDetails"]["hashes"]
        tx_signatures = wallet_utils.sign_messages(self.rpc_client, transaction_hashes, constants.user_1.address)
        tx_status = wallet_utils.send_router_transactions_with_signatures(self.rpc_client, uuid, tx_signatures)
        wallet_utils.check_tx_details(self.rpc_client, tx_status["hash"], self.network_id, constants.user_2.address, amount_in)

    def test_setting_different_fee_modes(self):
        uuid = str(uuid_lib.uuid4())
        gas_fee_mode = constants.gas_fee_mode_medium
        amount_in = "0xde0b6b3a7640000"

        router_input_params = {
            "uuid": uuid,
            "sendType": 0,
            "addrFrom": constants.user_1.address,
            "addrTo": constants.user_2.address,
            "amountIn": amount_in,
            "amountOut": "0x0",
            "tokenID": "ETH",
            "tokenIDIsOwnerToken": False,
            "toTokenID": "",
            "disabledFromChainIDs": [1, 10, 42161],
            "disabledToChainIDs": [1, 10, 42161],
            "gasFeeMode": gas_fee_mode,
        }

        logging.info("Step: getting the best route")
        routes = wallet_utils.get_suggested_routes(self.rpc_client, **router_input_params)
        assert len(routes["Best"]) > 0
        wallet_utils.check_fees_for_path(constants.processor_name_transfer, gas_fee_mode, routes["Best"][0]["ApprovalRequired"], routes["Best"])

        logging.info("Step: update gas fee mode without providing path tx identity params via wallet_setFeeMode endpoint")
        method = "wallet_setFeeMode"
        response = self.rpc_client.rpc_request(method, [None, gas_fee_mode])
        self.rpc_client.verify_is_json_rpc_error(response)

        logging.info("Step: update gas fee mode with incomplete details for path tx identity params via wallet_setFeeMode endpoint")
        tx_identity_params = {
            "routerInputParamsUuid": uuid,
        }
        response = self.rpc_client.rpc_request(method, [tx_identity_params, gas_fee_mode])
        self.rpc_client.verify_is_json_rpc_error(response)

        logging.info("Step: update gas fee mode to low")
        gas_fee_mode = constants.gas_fee_mode_low
        tx_identity_params = {
            "routerInputParamsUuid": uuid,
            "pathName": routes["Best"][0]["ProcessorName"],
            "chainID": routes["Best"][0]["FromChain"]["chainId"],
            "isApprovalTx": routes["Best"][0]["ApprovalRequired"],
        }
        self.rpc_client.prepare_wait_for_signal("wallet.suggested.routes", 1)
        _ = self.rpc_client.rpc_valid_request(method, [tx_identity_params, gas_fee_mode])
        response = self.rpc_client.wait_for_signal("wallet.suggested.routes")
        routes = response["event"]
        assert len(routes["Best"]) > 0
        wallet_utils.check_fees_for_path(constants.processor_name_transfer, gas_fee_mode, routes["Best"][0]["ApprovalRequired"], routes["Best"])

        logging.info("Step: update gas fee mode to high")
        gas_fee_mode = constants.gas_fee_mode_high
        self.rpc_client.prepare_wait_for_signal("wallet.suggested.routes", 1)
        _ = self.rpc_client.rpc_valid_request(method, [tx_identity_params, gas_fee_mode])
        response = self.rpc_client.wait_for_signal("wallet.suggested.routes")
        routes = response["event"]
        assert len(routes["Best"]) > 0
        wallet_utils.check_fees_for_path(constants.processor_name_transfer, gas_fee_mode, routes["Best"][0]["ApprovalRequired"], routes["Best"])

        logging.info("Step: try to set custom gas fee mode via wallet_setFeeMode endpoint")
        gas_fee_mode = constants.gas_fee_mode_custom
        response = self.rpc_client.rpc_request(method, [tx_identity_params, gas_fee_mode])
        self.rpc_client.verify_is_json_rpc_error(response)

    def test_setting_custom_fee_mode(self):
        uuid = str(uuid_lib.uuid4())
        gas_fee_mode = constants.gas_fee_mode_medium
        amount_in = "0xde0b6b3a7640000"

        router_input_params = {
            "uuid": uuid,
            "sendType": 0,
            "addrFrom": constants.user_1.address,
            "addrTo": constants.user_2.address,
            "amountIn": amount_in,
            "amountOut": "0x0",
            "tokenID": "ETH",
            "tokenIDIsOwnerToken": False,
            "toTokenID": "",
            "disabledFromChainIDs": [1, 10, 42161],
            "disabledToChainIDs": [1, 10, 42161],
            "gasFeeMode": gas_fee_mode,
        }

        logging.info("Step: getting the best route")
        routes = wallet_utils.get_suggested_routes(self.rpc_client, **router_input_params)
        assert len(routes["Best"]) > 0
        wallet_utils.check_fees_for_path(constants.processor_name_transfer, gas_fee_mode, routes["Best"][0]["ApprovalRequired"], routes["Best"])

        logging.info("Step: try to set custom tx details with empty params via wallet_setCustomTxDetails endpoint")
        method = "wallet_setCustomTxDetails"
        response = self.rpc_client.rpc_request(method, [None, None])
        self.rpc_client.verify_is_json_rpc_error(response)

        logging.info("Step: try to set custom tx details with incomplete details for path tx identity params via wallet_setCustomTxDetails endpoint")
        tx_identity_params = {
            "routerInputParamsUuid": uuid,
        }
        response = self.rpc_client.rpc_request(method, [tx_identity_params, None])
        self.rpc_client.verify_is_json_rpc_error(response)

        logging.info("Step: try to set custom tx details providing other than the custom gas fee mode via wallet_setCustomTxDetails endpoint")
        tx_identity_params = {
            "routerInputParamsUuid": uuid,
            "pathName": routes["Best"][0]["ProcessorName"],
            "chainID": routes["Best"][0]["FromChain"]["chainId"],
            "isApprovalTx": routes["Best"][0]["ApprovalRequired"],
        }
        tx_custom_params = {
            "gasFeeMode": constants.gas_fee_mode_low,
        }
        response = self.rpc_client.rpc_request(method, [tx_identity_params, tx_custom_params])
        self.rpc_client.verify_is_json_rpc_error(response)

        logging.info("Step: try to set custom tx details without providing maxFeesPerGas via wallet_setCustomTxDetails endpoint")
        tx_custom_params = {
            "gasFeeMode": gas_fee_mode,
        }
        response = self.rpc_client.rpc_request(method, [tx_identity_params, tx_custom_params])
        self.rpc_client.verify_is_json_rpc_error(response)

        logging.info("Step: try to set custom tx details without providing PriorityFee via wallet_setCustomTxDetails endpoint")
        tx_custom_params = {
            "gasFeeMode": gas_fee_mode,
            "maxFeesPerGas": "0x77359400",
        }
        response = self.rpc_client.rpc_request(method, [tx_identity_params, tx_custom_params])
        self.rpc_client.verify_is_json_rpc_error(response)

        logging.info("Step: try to set custom tx details via wallet_setCustomTxDetails endpoint")
        gas_fee_mode = constants.gas_fee_mode_custom
        tx_nonce = 4
        tx_gas_amount = 30000
        tx_max_fees_per_gas = "0x77359400"
        tx_priority_fee = "0x1DCD6500"
        tx_identity_params = {
            "routerInputParamsUuid": uuid,
            "pathName": routes["Best"][0]["ProcessorName"],
            "chainID": routes["Best"][0]["FromChain"]["chainId"],
            "isApprovalTx": routes["Best"][0]["ApprovalRequired"],
        }
        tx_custom_params = {
            "gasFeeMode": gas_fee_mode,
            "nonce": tx_nonce,
            "gasAmount": tx_gas_amount,
            "maxFeesPerGas": tx_max_fees_per_gas,
            "priorityFee": tx_priority_fee,
        }
        self.rpc_client.prepare_wait_for_signal("wallet.suggested.routes", 1)
        _ = self.rpc_client.rpc_valid_request(method, [tx_identity_params, tx_custom_params])
        response = self.rpc_client.wait_for_signal("wallet.suggested.routes")
        routes = response["event"]
        assert len(routes["Best"]) > 0
        tx_nonce_int = int(routes["Best"][0]["TxNonce"], 16)
        assert tx_nonce_int == tx_nonce
        assert routes["Best"][0]["TxGasAmount"] == tx_gas_amount
        assert routes["Best"][0]["TxMaxFeesPerGas"].upper() == tx_max_fees_per_gas.upper()
        assert routes["Best"][0]["TxPriorityFee"].upper() == tx_priority_fee.upper()
        wallet_utils.check_fees_for_path(constants.processor_name_transfer, gas_fee_mode, routes["Best"][0]["ApprovalRequired"], routes["Best"])
