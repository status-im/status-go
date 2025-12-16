from clients.foundry import Foundry
from resources.constants import DEPLOYER_ACCOUNT
import os
import tarfile
import io


class ERC721Deployer:

    def __init__(self, foundry: Foundry, total_supply: int = 10):
        constructor_args = [total_supply]
        erc721_tar = self._create_erc721_tar()
        self.mock_erc721_address = foundry.put_and_deploy(
            data=erc721_tar,
            contract_path="MockERC721.sol",
            contract_name="MockERC721",
            constructor_args=constructor_args,
            private_key=DEPLOYER_ACCOUNT.private_key,
            sender_address=DEPLOYER_ACCOUNT.address,
        )

    def _create_erc721_tar(self):
        root_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(__file__))))
        sol_path = os.path.join(root_dir, "contracts", "erc721", "MockERC721.sol")
        tar_buffer = io.BytesIO()
        with tarfile.open(fileobj=tar_buffer, mode="w|") as tar:
            tar.add(sol_path, arcname="MockERC721.sol")
        tar_buffer.seek(0)
        return tar_buffer.getvalue()
