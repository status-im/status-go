from clients.foundry import Foundry


class Multicall3Deployer:

    def __init__(self, foundry: Foundry):
        multicall3_log = foundry.get_archive("/app/contracts/Multicall3.sol.log")

        with open(multicall3_log, "r") as f:
            output = f.read()
            for line in output.splitlines():
                if "Deployed to:" in line:
                    contract_address = line.split("Deployed to:")[1].strip()
                    print(f"Contract deployed at: {contract_address}")
                    self.contract_address = contract_address

        if not self.contract_address:
            raise Exception("Contract address not found in output.")
