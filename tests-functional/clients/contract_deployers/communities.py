import os
from clients.smart_contract_runner import SmartContractRunner


class CommunitiesDeployer:

    def __init__(self, smart_contract_runner: SmartContractRunner):
        output_path = os.path.join(smart_contract_runner.get_output_dir("communities-contracts"), "run-latest.json")
        self.deploy_output = smart_contract_runner.get_output(output_path)
