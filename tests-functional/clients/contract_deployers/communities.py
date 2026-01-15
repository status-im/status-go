from clients.foundry import Foundry
from resources.constants import DEPLOYER_ACCOUNT


class CommunitiesDeployer:

    def __init__(self, foundry: Foundry):
        self.deploy_output = foundry.clone_and_run(
            github_org="status-im",
            github_repo="communities-contracts",
            smart_contract_dir="script",
            smart_contract_filename="DeployContracts.s.sol",
            private_key=DEPLOYER_ACCOUNT.private_key,
            sender_address=DEPLOYER_ACCOUNT.address,
        )

    @classmethod
    def from_file(cls, foundry: Foundry, container_file_path: str):
        """Load Communities contract addresses from a JSON file in the foundry container."""
        import json

        instance = cls.__new__(cls)

        # Read the JSON file from the container
        host_file_path = foundry.get_archive(container_file_path)
        with open(host_file_path, "r") as f:
            deploy_output = json.load(f)

        instance.deploy_output = deploy_output

        return instance
