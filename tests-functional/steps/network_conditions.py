from contextlib import contextmanager
import logging


class NetworkConditionsSteps:

    @contextmanager
    def add_latency(self, node, latency=300, jitter=50):
        logging.info("Entering context manager: add_latency")
        node.container_exec(f"apk add iproute2 && tc qdisc add dev eth0 root netem delay {latency}ms {jitter}ms distribution normal")
        try:
            yield
        finally:
            logging.info("Exiting context manager: add_latency")
            node.container_exec("tc qdisc del dev eth0 root")

    @contextmanager
    def add_packet_loss(self, node, packet_loss=2):
        logging.info("Entering context manager: add_packet_loss")
        node.container_exec(f"apk add iproute2 && tc qdisc add dev eth0 root netem loss {packet_loss}%")
        try:
            yield
        finally:
            logging.info("Exiting context manager: add_packet_loss")
            node.container_exec("tc qdisc del dev eth0 root netem")

    @contextmanager
    def add_low_bandwith(self, node, rate="1mbit", burst="32kbit", limit="12500"):
        logging.info("Entering context manager: add_low_bandwith")
        node.container_exec(f"apk add iproute2 && tc qdisc add dev eth0 root tbf rate {rate} burst {burst} limit {limit}")
        try:
            yield
        finally:
            logging.info("Exiting context manager: add_low_bandwith")
            node.container_exec("tc qdisc del dev eth0 root")

    @contextmanager
    def node_pause(self, node):
        logging.info("Entering context manager: node_pause")
        node.container_pause()
        try:
            yield
        finally:
            logging.info("Exiting context manager: node_pause")
            node.container_unpause()
