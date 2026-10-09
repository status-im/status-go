import pytest

import resources.constants as constants
from clients.services.wallet_send_type import WalletSendType
from utils import wallet_utils

MINTED = 10 * 10**18
AMOUNT_IN = "0xde0b6b3a7640000"


@pytest.mark.rpc
@pytest.mark.transaction
@pytest.mark.wallet
class TestRouterErc20:

    @pytest.fixture(autouse=True)
    def setup_backend(
        self,
        funded_new_profile,
        anvil_client,
        foundry_client,
        multicall3_deployer,
        snt_token_overrides,
        snt_addresses,
    ):
        self.anvil_client = anvil_client
        self.foundry_client = foundry_client
        self.snt_address = snt_addresses["snt"]
        self.rpc_client, self.wallet_address = funded_new_profile(
            name="erc20_user",
            token_overrides=snt_token_overrides,
            multicall_contract_address=multicall3_deployer.contract_address,
        )
        minted = foundry_client.generate_tokens(
            snt_addresses["controller"],
            self.wallet_address,
            str(MINTED),
            constants.DEPLOYER_ACCOUNT.private_key,
        )
        assert minted.exit_code == 0, minted.output

    def _erc20_balance(self, owner):
        raw = self.foundry_client.get_erc20_balance(self.snt_address, owner)
        return int(raw.output.decode().strip(), 16)

    def test_erc20_transfer(self):
        amount = int(AMOUNT_IN, 16)
        burn_before = self._erc20_balance(constants.BURN_ADDRESS)

        token_key = wallet_utils.get_token_key(constants.ANVIL_NETWORK_ID, self.snt_address)
        result = wallet_utils.send_token_transfer(
            self.rpc_client,
            WalletSendType.TRANSFER,
            self.wallet_address,
            token_key,
            AMOUNT_IN,
        )

        assert result["routes"]["Route"][0]["ProcessorName"] == constants.processor_name_transfer

        tx_data = self.anvil_client.get_transaction(result["tx_status"]["hash"])
        assert tx_data["value"] == 0
        assert tx_data["to"].lower() == self.snt_address.lower()
        assert tx_data["input"][:4] == bytes.fromhex("a9059cbb")

        assert self._erc20_balance(constants.BURN_ADDRESS) - burn_before == amount
        assert self._erc20_balance(self.wallet_address) == MINTED - amount
