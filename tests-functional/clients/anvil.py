import logging
import time
import docker

from utils.config import Config
from tenacity import retry, wait_fixed, stop_after_attempt
from web3 import Web3
from web3.types import (
    TxData,
    TxReceipt,
    RPCEndpoint,
)
from eth_typing import (
    HexStr,
)
from typing import (
    Union,
)
from hexbytes import HexBytes


class Anvil(Web3):

    def __init__(self):
        self.docker_client = docker.from_env()
        self.docker_project_name = Config.docker_project_name
        self.network_name = f"{self.docker_project_name}_default"

        container_name_prefix = f"{self.docker_project_name}-anvil"
        self.container_name = self.find_container_name(self.network_name, container_name_prefix)

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
            if self.is_connected(show_traceback=True):
                logging.info(f"Anvil is healthy after {time.time() - start_time} seconds")
                return
            else:
                time.sleep(0.1)
        raise TimeoutError(f"Anvil was not healthy after {timeout} seconds")

    def get_transaction(self, tx_hash: str) -> TxData:
        return self.eth.get_transaction(HexStr(tx_hash))

    def transaction_receipt(self, tx_hash: str) -> TxReceipt:
        return self.eth.get_transaction_receipt(HexStr(tx_hash))

    def send_raw_transaction(self, transaction: Union[HexStr, bytes]) -> HexBytes:
        return self.eth.send_raw_transaction(transaction)

    def set_balance(self, address: str, raw_amount: int):
        return self.provider.make_request(RPCEndpoint("anvil_setBalance"), [address, hex(raw_amount)])
