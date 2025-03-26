import io
import json
import logging
import tarfile
import tempfile
import time
import docker
import docker.errors
import os

from conftest import option
from resources.constants import user_1, ANVIL_NETWORK_ID
from tenacity import retry, wait_fixed, stop_after_attempt


class SmartContractRunner:

    container = None

    def __init__(self):
        self.docker_client = docker.from_env()
        self.docker_project_name = option.docker_project_name
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
        host_output_path = self._extract_data(container_output_path)
        if not host_output_path:
            raise Exception(f"Failed to extract data from {container_output_path}")
        with open(host_output_path, "r") as f:
            output = json.load(f)
        return output["returns"]

    def _extract_data(self, path: str):
        if not self.container:
            return path

        try:
            stream, _ = self.container.get_archive(path)
        except docker.errors.NotFound:
            return None

        temp_dir = tempfile.mkdtemp()
        tar_bytes = io.BytesIO(b"".join(stream))

        with tarfile.open(fileobj=tar_bytes) as tar:
            tar.extractall(path=temp_dir)
            # If the tar contains a single file, return the path to that file
            # Otherwise it's a directory, just return temp_dir.
            if len(tar.getmembers()) == 1:
                return os.path.join(temp_dir, tar.getmembers()[0].name)

        return temp_dir
