import io
import logging
import os
import random
import tarfile
import tempfile
import threading

import docker
import docker.errors
from docker.errors import APIError

from clients.metrics import ContainerStats
from utils.config import Config

DATA_DIR = "/usr/status-user"


class StatusGoContainer:
    all_containers = []
    container = None

    def __init__(self, entrypoint, ports=None, privileged=False, container_name_suffix=""):
        if ports is None:
            ports = {}

        # Initialize stop event for monitoring thread
        self._stop_monitoring = threading.Event()
        self.health_monitor = None
        self._stop_perf_monitoring = threading.Event()
        self.perf_monitor = None

        # Initialize performance metrics container
        self.stats = list[ContainerStats]()

        # Prepare image and container name
        # NOTE: This part needs some love.
        #       There's magic with `docker_project_name`, `docker_image` and `identifier` variables.
        docker_project_name = Config.docker_project_name
        self.network_name = f"{docker_project_name}_default"
        git_commit = os.popen("git rev-parse --short HEAD").read().strip()
        identifier = os.environ.get("BUILD_ID") if os.environ.get("CI") else git_commit
        image_name = Config.docker_image or f"statusgo-{identifier}:latest"
        self.container_name = f"{docker_project_name}-{identifier}{container_name_suffix}"
        coverage_path = Config.codecov_dir if Config.codecov_dir else os.path.abspath("./coverage/binary")

        # Run the container
        logging.debug(f"Creating status-go container from image '{image_name}'")

        container_args = {
            "image": image_name,
            "detach": True,
            "privileged": privileged,
            "name": self.container_name,
            "labels": {"com.docker.compose.project": docker_project_name},  # TODO: Is this still needed?
            "environment": {
                "GOCOVERDIR": "/coverage/binary",
            },
            "volumes": {
                coverage_path: {
                    "bind": "/coverage/binary",
                    "mode": "rw",
                }
            },
            "extra_hosts": {
                "host.docker.internal": "host-gateway",
            },
            "entrypoint": entrypoint,
            "ports": ports,
            "stop_signal": "SIGINT",
        }

        if "FUNCTIONAL_TESTS_DOCKER_UID" in os.environ:
            container_args["user"] = os.environ["FUNCTIONAL_TESTS_DOCKER_UID"]

        self.docker_client = docker.from_env()

        try:
            self.docker_client.images.get(image_name)
        except docker.errors.ImageNotFound:
            raise RuntimeError(f"Docker image '{image_name}' not found")

        self.container = self.docker_client.containers.run(**container_args)
        StatusGoContainer.all_containers.append(self)

        logging.debug(f"Container {self.container.name} created. ID = {self.container.id}")

        network = self.docker_client.networks.get(self.network_name)
        network.connect(self.container)

    def __del__(self):
        self.stop()

    def data_dir(self):
        return DATA_DIR

    def id(self):
        return self.container.id if self.container else ""

    def short_id(self):
        return self.container.id[:8] if self.container else ""

    def name(self):
        return self.container.name if self.container else ""

    def _check_container_health(self):
        """Check if container is healthy"""
        if not self.container:
            raise RuntimeError("Container is not initialized")

        self.container.reload()
        if self.container.status != "running":
            logs = self.container.logs().decode("utf-8").splitlines()[-10:]
            logs = "\n".join(logs)
            raise RuntimeError(f"Container is not running. Status: {self.container.status}. Logs (last 10 lines):\n{logs}")
        return True

    def start_health_monitoring(self):
        """Start background health monitoring thread"""

        def monitor():
            while not self._stop_monitoring.is_set():
                try:
                    self._check_container_health()
                    # Wait for 5 seconds or until stop event is set
                    self._stop_monitoring.wait(timeout=1)
                except Exception as e:
                    logging.error(f"Container health check failed: {e}")
                    raise e  # This will kill the thread and fail the test

        self._stop_monitoring.clear()  # Reset the event
        self.health_monitor = threading.Thread(target=monitor, daemon=True)
        self.health_monitor.start()

    def start_performance_monitoring(self):
        """Start a background performance monitoring thread that collects CPU and memory metrics"""
        # Reset metrics storage
        self.stats = list[ContainerStats]()
        self._stop_perf_monitoring = threading.Event()

        def monitor_performance():
            stats_stream = self.docker_client.api.stats(self.id(), decode=True, stream=True)
            prev_stat = None  # Store previous stat for rate calculations

            for stat in stats_stream:
                # Create ContainerStats with previous stat for network rate calculation
                container_stats = ContainerStats(stat, prev_stat)
                self.stats.append(container_stats)

                # Store current stat as previous for the next iteration
                prev_stat = stat

                if self._stop_perf_monitoring.is_set():
                    break

            logging.debug(f"Performance monitoring stopped for container {self.name()}")

        self._stop_perf_monitoring.clear()  # Reset the event
        self.perf_monitor = threading.Thread(target=monitor_performance, daemon=True)
        self.perf_monitor.start()
        logging.info(f"Started performance monitoring for container {self.name()}")

    def stop_performance_monitoring(self):
        """Stop the performance monitoring thread and return the collected metrics"""
        self._stop_perf_monitoring.set()  # Signal the thread to stop
        if not self.perf_monitor or not self.perf_monitor.is_alive():
            return None

        self.perf_monitor.join(timeout=10)
        if self.perf_monitor.is_alive():
            logging.warning("Performance monitoring thread didn't stop gracefully")

        return self.stats

    def stop_health_monitoring(self):
        """Stop the health monitoring thread"""
        self._stop_monitoring.set()  # Signal the thread to stop
        if not self.health_monitor or not self.health_monitor.is_alive():
            return
        self.health_monitor.join(timeout=10)
        if self.health_monitor.is_alive():
            logging.warning("Health monitoring thread didn't stop gracefully")

    def shutdown(self):
        """
        Stops, saves logs, and removes a container with error handling.
        Args:
            container: The container object (should have stop, save_logs, remove methods)
            log_prefix: Optional string for logging context
        """
        if not self.container:
            logging.debug("No container to shutdown")
            return

        container_id = self.short_id()
        self.stop()
        self.save_logs()
        self.remove()
        logging.debug(f"Container '{container_id}' shutdown finished")

    def stop(self):
        """Stop the container and monitoring"""
        self.stop_health_monitoring()  # Stop health monitoring first
        if hasattr(self, "_stop_perf_monitoring"):
            self.stop_performance_monitoring()  # Stop performance monitoring if running
        if self.container:
            logging.debug(f"Stopping container {self.container.name}...")
            self.container.stop(timeout=10)
            logging.debug(f"Container {self.container.name} stopped.")

    def remove(self):
        """Remove the container"""
        if self.container:
            name = self.container.name
            logging.debug(f"Removing container {name}...")
            self.container.remove()
            self.container = None
            logging.debug(f"Container {name} removed.")

    def pause(self):
        if not self.container:
            raise RuntimeError("Container is not initialized.")
        self.container.pause()
        logging.info(f"Container {self.container.name} paused.")

    def unpause(self):
        if not self.container:
            raise RuntimeError("Container is not initialized.")
        self.container.unpause()
        logging.info(f"Container {self.container.name} unpaused.")

    def exec(self, command):
        if not self.container:
            raise RuntimeError("Container is not initialized.")
        try:
            exec_result = self.container.exec_run(cmd=["sh", "-c", command], stdout=True, stderr=True, tty=False)
            if exec_result.exit_code != 0:
                raise RuntimeError(f"Failed to execute command in container {self.container.id}:\n" f"OUTPUT: {exec_result.output.decode().strip()}")
            return exec_result.output.decode().strip()
        except APIError as e:
            raise RuntimeError(f"API error during container execution: {str(e)}") from e

    def extract_data(self, path: str):
        if not self.container:
            raise RuntimeError("Container is not initialized.")

        try:
            stream, _ = self.container.get_archive(path)
        except docker.errors.NotFound:
            logging.error(f"Path '{path}' not found in container {self.container.name}.")
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

    def import_data(self, src_path: str, dest_path: str):
        """
        Copy data from the host (src_path) into the container at dest_path.
        """
        if not self.container:
            raise RuntimeError("Container is not initialized.")

        if not os.path.exists(src_path):
            raise FileNotFoundError(f"Source path '{src_path}' does not exist.")

        # Create a tar archive of the source path
        tar_stream = io.BytesIO()
        with tarfile.open(fileobj=tar_stream, mode="w") as tar:
            arcname = os.path.basename(src_path)
            tar.add(src_path, arcname=arcname)
        tar_stream.seek(0)

        # Put the archive into the container at the destination path
        try:
            # Ensure destination directory exists in the container
            response = self.container.exec_run(cmd=["mkdir", "-p", dest_path])
            assert response.exit_code == 0, f"Failed to ensure directory exists: {response.output.decode().strip()}"
            success = self.container.put_archive(dest_path, tar_stream.getvalue())
            assert success, f"Failed to put archive to {dest_path} in container {self.container.name}"
        except Exception as e:
            logging.error(f"Failed to import data to container: {e}")
            raise

    def get_name(self):
        return self.container.name if self.container else None

    def save_logs(self):
        if not self.container:
            raise RuntimeError("Container is not initialized.")
        if Config.logs_dir == "":
            logging.debug("Save container logs skipped")
            return

        file_path = os.path.join(Config.logs_dir, f"container_{self.short_id()}.log")
        logging.info(f"Saving logs to {file_path}")

        with open(file_path, "wb") as f:
            logs = self.container.logs()
            f.write(logs)


class PushNotificationServerContainer(StatusGoContainer):
    def __init__(self, identity, gorush_port):
        entrypoint = [
            "push-notification-server",
            "--identity",
            identity,
            "--gorush-url",
            f"http://host.docker.internal:{gorush_port}",
            "--data-dir",
            DATA_DIR,
            "--log-level",
            "DEBUG",
            "--waku-fleet-config",
            Config.waku_fleets_config,
            "--waku-fleet",
            Config.waku_fleet,
        ]
        super().__init__(entrypoint, container_name_suffix=f"-push-notification-server-{gorush_port}")


class StatusBackendContainer(StatusGoContainer):
    def __init__(self, host_port: int, privileged=False, ipv6=False):
        container_port = 3333
        entrypoint = [
            "status-backend",
            "--address",
            f"0.0.0.0:{container_port}" if not ipv6 else f"[::]:{container_port}",
        ]

        self.ipv6 = ipv6
        self.url = f"http://{'[::1]' if ipv6 else '127.0.0.1'}:{host_port}"

        if ipv6:
            ports = {
                f"{container_port}/tcp": [
                    {"HostIp": "::", "HostPort": str(host_port)},
                ]
            }
        else:
            ports = {
                f"{container_port}/tcp": str(host_port),
            }

        super().__init__(entrypoint, ports, privileged, container_name_suffix=f"-status-backend-{host_port}")

    def _change_ip(self, new_ipv4=None, new_ipv6=None):
        if not self.container:
            raise RuntimeError("Container is not initialized.")

        # Get the network details
        network = self.docker_client.networks.get(self.network_name)

        # Ensure network has explicitly configured subnets
        ipam_config = network.attrs.get("IPAM", {}).get("Config", [])
        if not ipam_config:
            raise RuntimeError("Network does not have a user-defined subnet, cannot assign a custom IP.")

        self.container.reload()
        container_info = self.container.attrs["NetworkSettings"]["Networks"].get(self.network_name, {})
        current_ipv4 = container_info.get("IPAddress", "Unknown")
        current_ipv6 = container_info.get("GlobalIPv6Address", "Unknown")

        logging.info(f"Current IPs for {self.container.name} - IPv4: {current_ipv4}, IPv6: {current_ipv6}")

        # Generate new IPs based on mode
        for config in ipam_config:
            subnet = config.get("Subnet")

            if self.ipv6 and ":" in subnet and not new_ipv6:  # IPv6 Subnet
                base_ipv6 = subnet.rstrip("::/64")
                new_ipv6 = f"{base_ipv6}::{random.randint(1, 9999):x}:{random.randint(1, 9999):x}"
                logging.info(f"Generated new IPv6: {new_ipv6}")

            elif not self.ipv6 and "." in subnet and not new_ipv4:  # IPv4 Subnet
                new_ipv4 = subnet.rsplit(".", 1)[0] + f".{random.randint(2, 254)}"
                logging.info(f"Generated new IPv4: {new_ipv4}")

        # Disconnect and reconnect with only the needed IP type
        network.disconnect(self.container)
        if self.ipv6:
            network.connect(self.container, ipv6_address=new_ipv6)
        else:
            network.connect(self.container, ipv4_address=new_ipv4)

        self.container.reload()
        updated_info = self.container.attrs["NetworkSettings"]["Networks"].get(self.network_name, {})
        updated_ipv4 = updated_info.get("IPAddress", "Unknown")
        updated_ipv6 = updated_info.get("GlobalIPv6Address", "Unknown")

        if self.ipv6 and current_ipv6 == updated_ipv6:
            raise RuntimeError("IPV6 is the same after network reconnect")
        if not self.ipv6 and current_ipv4 == updated_ipv4:
            raise RuntimeError("IPV4 is the same after network reconnect")

        logging.info(f"Changed container {self.container.name} IPs - New IPv4: {updated_ipv4}, New IPv6: {updated_ipv6}")

    def change_ip(self, new_ipv4=None, new_ipv6=None):
        try:
            logging.info(f"Trying to change container {self.container_name} IPs (IPv6 Mode: {self.ipv6})")
            self._change_ip(new_ipv4, new_ipv6)
        except Exception as e:
            raise RuntimeError(f"Failed to change container IP: {e}")
