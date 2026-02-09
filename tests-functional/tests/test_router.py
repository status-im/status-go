import re
import uuid as uuid_lib

import pytest
from web3.types import Wei  # type: ignore

import resources.constants as constants
from clients.api import ApiResponseError
from utils import wallet_utils


@pytest.mark.rpc
@pytest.mark.transaction
@pytest.mark.wallet
class TestRouter:

    @pytest.fixture(autouse=True)
    def setup_backend(self, backend_recovered_profile, anvil_client, multicall3_deployer):
        self.anvil_client = anvil_client
        self.rpc_client = backend_recovered_profile(
            name="main_user", user=constants.user_1, multicall_contract_address=multicall3_deployer.contract_address
        )

    def test_tx_from_route(self):
        uuid = str(uuid_lib.uuid4())
        gas_fee_mode = constants.gas_fee_mode_medium
        amount_in = "0xde0b6b3a7640000"
        amount_in_wei = Wei(1000000000000000000)

        native_token_key = wallet_utils.get_token_key(constants.ANVIL_NETWORK_ID, constants.NATIVE_TOKEN_ADDRESS)
        params = {
            "uuid": uuid,
            "sendType": 0,
            "addrFrom": constants.user_1.address,
            "addrTo": constants.user_2.address,
            "amountIn": amount_in,
            "amountOut": "0x0",
            "tokenKey": native_token_key,
            "tokenIDIsOwnerToken": False,
            "toTokenKey": native_token_key,
            "fromChainID": constants.ANVIL_NETWORK_ID,
            "toChainID": constants.ANVIL_NETWORK_ID,
            "gasFeeMode": gas_fee_mode,
        }

        routes = wallet_utils.get_suggested_routes(self.rpc_client, **params)
        assert len(routes["Route"]) > 0
        wallet_router_sign_transactions = wallet_utils.build_transactions_from_route(self.rpc_client, uuid)
        if wallet_router_sign_transactions is None:
            raise ValueError("wallet_router_sign_transactions is None")

        if "signingDetails" not in wallet_router_sign_transactions:
            raise ValueError("signingDetails not found in wallet_router_sign_transactions")

        transaction_hashes = wallet_router_sign_transactions["signingDetails"]["hashes"]
        tx_signatures = wallet_utils.sign_messages(self.rpc_client, transaction_hashes, constants.user_1.address)
        tx_status = wallet_utils.send_router_transactions_with_signatures(self.rpc_client, uuid, tx_signatures)

        # Check tx details
        tx_data = self.anvil_client.get_transaction(tx_status["hash"])

        assert tx_data.get("value", 0) == int(amount_in, 16)
        assert tx_data.get("value", 0) == amount_in_wei
        assert tx_data.get("to", "").upper() == constants.user_2.address.upper()

    def test_setting_different_fee_modes(self):
        uuid = str(uuid_lib.uuid4())
        gas_fee_mode = constants.gas_fee_mode_medium
        amount_in = "0xde0b6b3a7640000"

        native_token_key = wallet_utils.get_token_key(constants.ANVIL_NETWORK_ID, constants.NATIVE_TOKEN_ADDRESS)
        router_input_params = {
            "uuid": uuid,
            "sendType": 0,
            "addrFrom": constants.user_1.address,
            "addrTo": constants.user_2.address,
            "amountIn": amount_in,
            "amountOut": "0x0",
            "tokenKey": native_token_key,
            "tokenIDIsOwnerToken": False,
            "toTokenKey": native_token_key,
            "fromChainID": constants.ANVIL_NETWORK_ID,
            "toChainID": constants.ANVIL_NETWORK_ID,
            "gasFeeMode": gas_fee_mode,
        }

        # Step: getting the best route
        routes = wallet_utils.get_suggested_routes(self.rpc_client, **router_input_params)
        assert len(routes["Route"]) > 0
        wallet_utils.check_fees_for_path(constants.processor_name_transfer, gas_fee_mode, routes["Route"][0]["ApprovalRequired"], routes["Route"])

        # Step: update gas fee mode without providing path tx identity params via wallet_setFeeMode endpoint
        with pytest.raises(ApiResponseError, match=re.escape("transaction identity not provided")):
            self.rpc_client.wallet_service.set_fee_mode(None, gas_fee_mode)

        # Step: update gas fee mode with incomplete details for path tx identity params via wallet_setFeeMode endpoint
        tx_identity_params = {
            "routerInputParamsUuid": uuid,
        }
        with pytest.raises(ApiResponseError, match=re.escape("Error:Field validation")):
            self.rpc_client.wallet_service.set_fee_mode(tx_identity_params, gas_fee_mode)

        # Step: update gas fee mode to low
        gas_fee_mode = constants.gas_fee_mode_low
        tx_identity_params = {
            "routerInputParamsUuid": uuid,
            "pathName": routes["Route"][0]["ProcessorName"],
            "chainID": routes["Route"][0]["FromChain"]["chainId"],
            "isApprovalTx": routes["Route"][0]["ApprovalRequired"],
        }
        with self.rpc_client.expect_signal("wallet.suggested.routes") as exp:
            _ = self.rpc_client.wallet_service.set_fee_mode(tx_identity_params, gas_fee_mode)
        response = exp.result
        routes = response["event"]
        assert len(routes["Route"]) > 0
        wallet_utils.check_fees_for_path(constants.processor_name_transfer, gas_fee_mode, routes["Route"][0]["ApprovalRequired"], routes["Route"])

        # Step: update gas fee mode to high
        gas_fee_mode = constants.gas_fee_mode_high
        with self.rpc_client.expect_signal("wallet.suggested.routes") as exp:
            _ = self.rpc_client.wallet_service.set_fee_mode(tx_identity_params, gas_fee_mode)
        response = exp.result
        routes = response["event"]
        assert len(routes["Route"]) > 0
        wallet_utils.check_fees_for_path(constants.processor_name_transfer, gas_fee_mode, routes["Route"][0]["ApprovalRequired"], routes["Route"])

        # Step: try to set custom gas fee mode via wallet_setFeeMode endpoint
        gas_fee_mode = constants.gas_fee_mode_custom
        with pytest.raises(ApiResponseError, match=re.escape("custom fee mode cannot be set this way")):
            self.rpc_client.wallet_service.set_fee_mode(tx_identity_params, gas_fee_mode)

    def test_setting_custom_fee_mode(self):
        uuid = str(uuid_lib.uuid4())
        gas_fee_mode = constants.gas_fee_mode_medium
        amount_in = "0xde0b6b3a7640000"

        native_token_key = wallet_utils.get_token_key(constants.ANVIL_NETWORK_ID, constants.NATIVE_TOKEN_ADDRESS)
        router_input_params = {
            "uuid": uuid,
            "sendType": 0,
            "addrFrom": constants.user_1.address,
            "addrTo": constants.user_2.address,
            "amountIn": amount_in,
            "amountOut": "0x0",
            "tokenKey": native_token_key,
            "tokenIDIsOwnerToken": False,
            "toTokenKey": native_token_key,
            "fromChainID": constants.ANVIL_NETWORK_ID,
            "toChainID": constants.ANVIL_NETWORK_ID,
            "gasFeeMode": gas_fee_mode,
        }

        # Step: getting the best route
        routes = wallet_utils.get_suggested_routes(self.rpc_client, **router_input_params)
        assert len(routes["Route"]) > 0
        wallet_utils.check_fees_for_path(constants.processor_name_transfer, gas_fee_mode, routes["Route"][0]["ApprovalRequired"], routes["Route"])

        # Step: try to set custom tx details with empty params via wallet_setCustomTxDetails endpoint
        with pytest.raises(ApiResponseError, match=re.escape("transaction identity not provided")):
            self.rpc_client.wallet_service.set_custom_tx_details(None, None)

        # Step: try to set custom tx details with incomplete details for path tx identity params via wallet_setCustomTxDetails endpoint
        tx_identity_params = {
            "routerInputParamsUuid": uuid,
        }
        with pytest.raises(ApiResponseError, match=re.escape("Error:Field validation")):
            self.rpc_client.wallet_service.set_custom_tx_details(tx_identity_params, None)

        # Step: try to set custom tx details providing other than the custom gas fee mode via wallet_setCustomTxDetails endpoint
        tx_identity_params = {
            "routerInputParamsUuid": uuid,
            "pathName": routes["Route"][0]["ProcessorName"],
            "chainID": routes["Route"][0]["FromChain"]["chainId"],
            "isApprovalTx": routes["Route"][0]["ApprovalRequired"],
        }
        tx_custom_params = {
            "gasFeeMode": constants.gas_fee_mode_low,
        }
        with pytest.raises(ApiResponseError, match=re.escape("only custom fee mode can be set this way")):
            self.rpc_client.wallet_service.set_custom_tx_details(tx_identity_params, tx_custom_params)

        # Step: try to set custom tx details without providing maxFeesPerGas via wallet_setCustomTxDetails endpoint
        tx_custom_params = {
            "gasFeeMode": gas_fee_mode,
        }
        with pytest.raises(ApiResponseError, match=re.escape("only custom fee mode can be set this way")):
            self.rpc_client.wallet_service.set_custom_tx_details(tx_identity_params, tx_custom_params)

        # Step: try to set custom tx details without providing PriorityFee via wallet_setCustomTxDetails endpoint
        tx_custom_params = {
            "gasFeeMode": gas_fee_mode,
            "maxFeesPerGas": "0x77359400",
        }
        with pytest.raises(ApiResponseError, match=re.escape("only custom fee mode can be set this way")):
            self.rpc_client.wallet_service.set_custom_tx_details(tx_identity_params, tx_custom_params)

        # Step: try to set custom tx details via wallet_setCustomTxDetails endpoint
        gas_fee_mode = constants.gas_fee_mode_custom
        tx_nonce = 4
        tx_gas_amount = 30000
        tx_max_fees_per_gas = "0x77359400"
        tx_priority_fee = "0x1DCD6500"
        tx_identity_params = {
            "routerInputParamsUuid": uuid,
            "pathName": routes["Route"][0]["ProcessorName"],
            "chainID": routes["Route"][0]["FromChain"]["chainId"],
            "isApprovalTx": routes["Route"][0]["ApprovalRequired"],
        }
        tx_custom_params = {
            "gasFeeMode": gas_fee_mode,
            "nonce": tx_nonce,
            "gasAmount": tx_gas_amount,
            "maxFeesPerGas": tx_max_fees_per_gas,
            "priorityFee": tx_priority_fee,
        }
        with self.rpc_client.expect_signal("wallet.suggested.routes") as exp:
            _ = self.rpc_client.wallet_service.set_custom_tx_details(tx_identity_params, tx_custom_params)
        response = exp.result
        routes = response["event"]
        assert len(routes["Route"]) > 0
        tx_nonce_int = int(routes["Route"][0]["TxNonce"], 16)
        assert tx_nonce_int == tx_nonce
        assert routes["Route"][0]["TxGasAmount"] == tx_gas_amount
        assert routes["Route"][0]["TxMaxFeesPerGas"].upper() == tx_max_fees_per_gas.upper()
        assert routes["Route"][0]["TxPriorityFee"].upper() == tx_priority_fee.upper()
        wallet_utils.check_fees_for_path(constants.processor_name_transfer, gas_fee_mode, routes["Route"][0]["ApprovalRequired"], routes["Route"])
