import io
import tarfile
from pathlib import Path

import pytest
from web3 import Web3

import resources.constants as constants
from clients.services.wallet_send_type import WalletSendType
from utils import wallet_utils

CONTRACTS_DIR = Path(__file__).resolve().parents[1] / "contracts"
TOKEN_ID = 1


def _deploy_collectible(foundry_client, filename, contract_name, wallet_address):
    sol_path = CONTRACTS_DIR / filename
    archive = io.BytesIO()
    with tarfile.open(fileobj=archive, mode="w") as tar:
        tar.add(sol_path, arcname=sol_path.name)
    return foundry_client.put_and_deploy(
        archive.getvalue(),
        sol_path.name,
        contract_name,
        constructor_args=[wallet_address],
    )


def _address_from_cast(raw: str) -> str:
    return Web3.to_checksum_address(f"0x{raw.strip()[-40:]}")


def _uint_from_cast(raw: str) -> int:
    return int(raw.strip(), 16)


@pytest.mark.rpc
@pytest.mark.transaction
@pytest.mark.wallet
class TestRouterCollectibles:

    @pytest.fixture(autouse=True)
    def setup_backend(self, funded_new_profile, anvil_client, foundry_client, multicall3_deployer):
        self.anvil_client = anvil_client
        self.foundry_client = foundry_client
        self.rpc_client, self.wallet_address = funded_new_profile(
            name="collectibles_user",
            multicall_contract_address=multicall3_deployer.contract_address,
        )

    def _send(self, filename, contract_name, send_type, amount_in, processor_name):
        contract = _deploy_collectible(self.foundry_client, filename, contract_name, self.wallet_address)
        result = wallet_utils.send_collectible_transfer(
            self.rpc_client,
            send_type,
            self.wallet_address,
            contract,
            TOKEN_ID,
            amount_in,
        )
        assert result["routes"]["Route"][0]["ProcessorName"] == processor_name
        tx_data = self.anvil_client.get_transaction(result["tx_status"]["hash"])
        assert tx_data["value"] == 0
        assert tx_data["to"].lower() == contract.lower()
        return contract

    def test_erc721_transfer(self):
        contract = self._send(
            "MinimalERC721.sol",
            "MinimalERC721",
            WalletSendType.ERC721_TRANSFER,
            "0x1",
            "ERC721Transfer",
        )
        owner = self.foundry_client.get_erc721_owner(contract, TOKEN_ID)
        assert _address_from_cast(owner.output.decode()) == Web3.to_checksum_address(constants.BURN_ADDRESS)

    def test_erc1155_transfer(self):
        contract = self._send(
            "MinimalERC1155.sol",
            "MinimalERC1155",
            WalletSendType.ERC1155_TRANSFER,
            "0x2",
            "ERC1155Transfer",
        )
        sender_balance = self.foundry_client.get_erc1155_balance(contract, self.wallet_address, TOKEN_ID)
        recipient_balance = self.foundry_client.get_erc1155_balance(contract, constants.BURN_ADDRESS, TOKEN_ID)
        assert _uint_from_cast(sender_balance.output.decode()) == 3
        assert _uint_from_cast(recipient_balance.output.decode()) == 2
