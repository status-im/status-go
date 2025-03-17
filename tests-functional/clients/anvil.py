import logging
import time
import docker

from conftest import option
from web3 import Web3


class Anvil(Web3):

    def __init__(self):
        self.docker_client = docker.from_env()
        self.docker_project_name = option.docker_project_name
        self.network_name = f"{self.docker_project_name}_default"
        network = self.docker_client.networks.get(self.network_name)
        self.container_name = None
        for container in network.containers:
            container_name_prefix = f"{self.docker_project_name}-anvil"
            container_name = container.name
            if container_name is not None and container_name_prefix in container_name:
                self.container_name = container_name
                break
        if not self.container_name:
            raise Exception("Anvil container not found")
        self.container = self.docker_client.containers.get(self.container_name)
        network_info = self.container.attrs["NetworkSettings"]["Ports"].get("8545/tcp", [])
        if not network_info:
            raise Exception("Anvil exposed port not found")
        self.ip = network_info[0]["HostIp"]
        self.port = network_info[0]["HostPort"]
        self.anvil_url = f"http://{self.ip}:{self.port}"
        logging.info(f"Anvil URL: {self.anvil_url}")
        Web3.__init__(self, Web3.HTTPProvider(self.anvil_url))
        self.wait_for_healthy()

    def wait_for_healthy(self, timeout=10):
        start_time = time.time()
        while time.time() - start_time <= timeout:
            if self.is_connected(show_traceback=True):
                logging.info(f"Anvil is healthy after {time.time() - start_time} seconds")
                return
            else:
                time.sleep(0.1)
        raise TimeoutError(f"Anvil was not healthy after {timeout} seconds")
