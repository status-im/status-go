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
