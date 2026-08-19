from clients.foundry import Foundry
from resources.constants import DEPLOYER_ACCOUNT


class ENSDeployer:

    def __init__(self, foundry: Foundry):
        self.deploy_output = foundry.clone_and_run(
            github_org="status-im",
            github_repo="ens-usernames",
            smart_contract_dir="script",
            smart_contract_filename="Deploy.s.sol",
            private_key=DEPLOYER_ACCOUNT.private_key,
            sender_address=DEPLOYER_ACCOUNT.address,
        )

        self.registry_address = self._find_contract("ENSRegistry")
        self.resolver_address = self._find_contract("PublicResolver")
        self.token_address = self._find_contract("MiniMeToken")
        self.registrar_address = self._find_contract("UsernameRegistrar")

    @classmethod
    def from_file(cls, foundry: Foundry, container_file_path: str):
        """Load ENS addresses from a JSON file in the foundry container."""
        import json

        instance = cls.__new__(cls)
        instance.deploy_output = None

        host_file_path = foundry.get_archive(container_file_path)
        with open(host_file_path, "r") as f:
            addresses = json.load(f)

        instance.registry_address = addresses["registry"]
        instance.resolver_address = addresses["resolver"]
        instance.token_address = addresses["token"]
        instance.registrar_address = addresses["registrar"]

        return instance

    def _find_contract(self, contract_name):
        for deployment in self.deploy_output.values():
            if f"contract {contract_name}" in deployment.get("internal_type", ""):
                return deployment["value"]
        raise Exception(f"{contract_name} contract not found in deploy output")
