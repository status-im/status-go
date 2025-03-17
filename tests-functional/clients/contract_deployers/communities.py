from clients.smart_contract_runner import SmartContractRunner
from resources.constants import DEPLOYER_ACCOUNT


class CommunitiesDeployer:

    def __init__(self, smart_contract_runner: SmartContractRunner):
        self.deploy_output = smart_contract_runner.clone_and_run(
            github_org="status-im",
            github_repo="communities-contracts",
            smart_contract_dir="script",
            smart_contract_filename="DeployContracts.s.sol",
            private_key=DEPLOYER_ACCOUNT.private_key,
            sender_address=DEPLOYER_ACCOUNT.address,
        )
