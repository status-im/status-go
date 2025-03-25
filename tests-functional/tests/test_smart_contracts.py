import pytest

from clients.smart_contract_runner import SmartContractRunner
from resources.constants import DEPLOYER_ACCOUNT


@pytest.mark.contracts
class TestDeploySmartContracts:
    def test_deployer(self):
        smart_contract_runner = SmartContractRunner()
        smart_contract_runner.clone_and_run(
            github_org="status-im",
            github_repo="status-network-token-v2",
            smart_contract_dir="script",
            smart_contract_filename="Deploy.s.sol",
            private_key=DEPLOYER_ACCOUNT.private_key,
            sender_address=DEPLOYER_ACCOUNT.address,
        )

        smart_contract_runner.clone_and_run(
            github_org="status-im",
            github_repo="communities-contracts",
            smart_contract_dir="script",
            smart_contract_filename="DeployContracts.s.sol",
            private_key=DEPLOYER_ACCOUNT.private_key,
            sender_address=DEPLOYER_ACCOUNT.address,
        )
