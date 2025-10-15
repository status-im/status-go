import io
import json
import logging
import tarfile
import tempfile
import time
import docker
import docker.errors
import os

from utils.config import Config
from resources.constants import user_1, ANVIL_NETWORK_ID
from tenacity import retry, wait_fixed, stop_after_attempt


class Foundry:

    container = None

    def __init__(self):
        self.docker_client = docker.from_env()
        self.docker_project_name = Config.docker_project_name
        self.network_name = f"{self.docker_project_name}_default"

        container_name_prefix = f"{self.docker_project_name}-foundry"
        self.container_name = self.find_container_name(self.network_name, container_name_prefix)

        if not self.container_name:
            raise Exception("Foundry container not found")
        self.container = self.docker_client.containers.get(self.container_name)
        self.wait_for_healthy()

    @retry(stop=stop_after_attempt(10), wait=wait_fixed(0.1), reraise=True)
    def find_container_name(self, network_name, searched_container):
        network = self.docker_client.networks.get(network_name)

        for container in network.containers:
            container_name = container.name
            if container_name is not None and searched_container in container_name:
                return container_name

        return None

    def wait_for_healthy(self, timeout=10):
        start_time = time.time()
        while time.time() - start_time <= timeout:
            if self.is_connected():
                logging.info(f"Foundry is healthy after {time.time() - start_time} seconds")
                return
            else:
                time.sleep(0.1)
        raise TimeoutError(f"Foundry was not healthy after {timeout} seconds")

    def is_connected(self):
        if not self.container:
            return False

        exec_result = self.container.exec_run("cast chain-id")
        exit_code = exec_result.exit_code
        if exit_code != 0:
            logging.info(f"Exit code: {exit_code}")
            return False
        output = exec_result.output.decode().strip()
        if output != str(ANVIL_NETWORK_ID):
            logging.info(f"ChainID comparison error. Expected: {output}, Actual:{ANVIL_NETWORK_ID}")
            return False
        return True

    def clone_and_run(self, **kwargs):
        if not self.container:
            raise Exception("Container not found")

        github_org = kwargs.get("github_org", "status-im")
        github_repo = kwargs.get("github_repo")
        if not github_repo:
            raise ValueError("github_repo is required")
        smart_contract_dir = kwargs.get("smart_contract_dir")
        if not smart_contract_dir:
            raise ValueError("smart_contract_dir is required")
        smart_contract_filename = kwargs.get("smart_contract_filename")
        if not smart_contract_filename:
            raise ValueError("smart_contract_filename is required")
        private_key = kwargs.get("private_key", user_1.private_key)
        sender_address = kwargs.get("sender_address", user_1.address)

        cmd = f"/app/clone_and_run.sh {github_org} {github_repo} {smart_contract_dir} {smart_contract_filename} {private_key} {sender_address}"
        logging.info(f"Running command: {cmd}")

        exec_result = self.container.exec_run(
            f"{cmd}",
            workdir="/app",
        )
        logging.info(f"Exit code: {exec_result.exit_code}")
        logging.info(f"Result: {exec_result.output.decode().strip()}")
        if exec_result.exit_code != 0:
            raise Exception(f"Failed to clone and run {github_repo}")

        container_output_path = f"/app/{github_repo}/broadcast/{smart_contract_filename}/{ANVIL_NETWORK_ID}/run-latest.json"
        host_output_path = self.get_archive(container_output_path)
        if not host_output_path:
            raise Exception(f"Failed to extract data from {container_output_path}")
        with open(host_output_path, "r") as f:
            output = json.load(f)
        return output["returns"]

    def put_and_deploy(self, data, contract_path, contract_name, **kwargs):
        if not self.container:
            raise Exception("Container not found")

        container_path = self.put_archive(data, **kwargs)

        private_key = kwargs.get("private_key", user_1.private_key)
        sender_address = kwargs.get("sender_address", user_1.address)

        cmd = f"""forge create {container_path}/{contract_path}:{contract_name}
            --rpc-url 'http://anvil:8545'
            --from {sender_address}
            --private-key {private_key}
            --broadcast"""
        constructor_args = kwargs.get("constructor_args")
        if constructor_args:
            cmd += f" --constructor-args {constructor_args}"

        logging.info(f"Running command: {cmd}")
        exec_result = self.container.exec_run(
            f"{cmd}",
            workdir="/app",
        )
        exit_code = exec_result.exit_code
        output = exec_result.output.decode()

        logging.info(f"Exit code: {exit_code}")
        logging.info(f"Result: {output.strip()}")
        if exit_code != 0:
            raise Exception(f"Failed to deploy {contract_name}")

        # Extract contract address from output
        for line in output.splitlines():
            if "Deployed to:" in line:
                contract_address = line.split("Deployed to:")[1].strip()
                print(f"Contract deployed at: {contract_address}")
                return contract_address
        raise Exception("Contract address not found in output.")

    def put_archive(self, data, **kwargs):
        if not self.container:
            raise Exception("Container not found")

        container_path = kwargs.get("container_path")
        if not container_path:
            # Create a temporary directory
            temp_dir_name = tempfile.mktemp(prefix="temp_", dir="/app").split("/")[-1]
            temp_dir_path = f"/app/{temp_dir_name}"  # Directory path inside the container

            # Create the temporary directory in the container
            create_dir_cmd = f"mkdir -p {temp_dir_path}"
            exec_response = self.container.exec_run(create_dir_cmd)
            if exec_response.exit_code != 0:
                raise Exception(f"Failed to create directory: {exec_response.output.decode()}")

            container_path = temp_dir_path
        logging.info(f"Putting archive in path: {container_path}")

        try:
            self.container.put_archive(container_path, data)
        except docker.errors.NotFound:
            raise Exception(f"Path '{container_path}' not found in container {self.container.name}")

        return container_path

    def get_archive(self, container_path):
        if not self.container:
            raise Exception("Container not found")

        try:
            stream, _ = self.container.get_archive(container_path)
        except docker.errors.NotFound:
            raise Exception(f"Path '{container_path}' not found in container {self.container.name}")

        temp_dir = tempfile.mkdtemp()
        tar_bytes = io.BytesIO(b"".join(stream))

        with tarfile.open(fileobj=tar_bytes) as tar:
            tar.extractall(path=temp_dir)
            # If the tar contains a single file, return the path to that file
            # Otherwise it's a directory, just return temp_dir.
            if len(tar.getmembers()) == 1:
                return os.path.join(temp_dir, tar.getmembers()[0].name)

        return temp_dir
