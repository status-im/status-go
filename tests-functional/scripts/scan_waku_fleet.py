#!/usr/bin/env python3

import argparse
import json
import sys
import os
import logging
import time

from urllib import request
from typing import Dict, List, Optional, Tuple


def print_config_options(args):
    logging.info("Configuration options:")
    logging.info(f"  Fleet name: {args.fleet_name}")
    logging.info(f"  Cluster ID: {args.cluster_id}")
    logging.info(f"  Static nodes: {args.static_nodes}")
    logging.info(f"  Bootstrap nodes: {args.bootstrap_nodes}")
    logging.info(f"  Store nodes: {args.store_nodes}")
    logging.info(f"  Output path: {args.output}")


def parse_args():
    parser = argparse.ArgumentParser(description="Scan Waku nodes and build wakufleetconfig.json using DNS-based ENRs")
    parser.add_argument("--fleet-name", required=True, help="Fleet name, e.g. status-go.test")
    parser.add_argument("--cluster-id", required=True, type=int, help="Cluster ID, e.g. 16")
    parser.add_argument(
        "--static-nodes",
        default="",
        help="Comma-separated list of static node hostnames (Docker DNS names), e.g. 'node-1,node-2'",
    )
    parser.add_argument(
        "--bootstrap-nodes",
        default="",
        help="Comma-separated list of bootstrap node hostnames, e.g. 'boot-1,boot-2'",
    )
    parser.add_argument(
        "--store-nodes",
        default="",
        help="Comma-separated list of store node hostnames, e.g. 'store-1,store-2'",
    )
    parser.add_argument("--output", required=True, help="Output path for wakufleetconfig.json inside the container")
    return parser.parse_args()


def _fetch_debug_info(host: str) -> Optional[Dict]:
    url = f"http://{host}:8645/debug/v1/info"
    logging.info(f"Fetching debug info from Waku node {url}")

    attempts = 15
    delay_seconds = 1.0

    for attempt in range(1, attempts + 1):
        try:
            logging.info(f"Attempt {attempt}/{attempts} to fetch debug info from {url}")
            with request.urlopen(url, timeout=5.0) as resp:
                if resp.status != 200:
                    logging.error(f"Unexpected status {resp.status} from {url}")
                    continue
                data = resp.read()
                return json.loads(data.decode("utf-8"))
        except Exception as e:  # noqa: BLE001
            logging.error(f"Error fetching debug info from {url}: {e}")
            if attempt < attempts:
                time.sleep(delay_seconds)

    return None


def _first_dns_addr(listen_addrs: List[str]) -> Optional[str]:
    for addr in listen_addrs:
        if addr.startswith("/dns4/") or addr.startswith("/dns6/"):
            return addr
    return listen_addrs[0] if listen_addrs else None


def _scan_hosts(hosts: List[str]) -> Dict[str, Tuple[str, Optional[str]]]:
    """
    Returns mapping host -> (enrUri, addr)
    """
    results: Dict[str, Tuple[str, Optional[str]]] = {}
    for host in hosts:
        host = host.strip()
        if not host:
            continue
        info = _fetch_debug_info(host)
        if not info:
            raise RuntimeError(f"Unable to gather ENR for host '{host}'")
        enr = info.get("enrUri")
        addrs = info.get("listenAddresses", []) or []
        addr = _first_dns_addr(addrs)
        if not enr:
            raise RuntimeError(f"No ENR in debug info for host '{host}'")
        results[host] = (enr, addr)
        logging.info(f"Host {host}: enr={enr}, addr={addr}")
    return results


def main():
    logging.basicConfig(level=logging.INFO, format="[scan_waku_fleet] %(message)s")
    args = parse_args()
    print_config_options(args)

    def split_list(s: str) -> List[str]:
        return [x.strip() for x in s.split(",") if x.strip()] if s else []

    static_hosts = split_list(args.static_nodes)
    bootstrap_hosts = split_list(args.bootstrap_nodes)
    store_hosts = split_list(args.store_nodes)

    all_hosts = list(dict.fromkeys(static_hosts + bootstrap_hosts + store_hosts))  # preserve order, unique

    if not all_hosts:
        logging.info("No hosts provided. Nothing to do.")
        return 0

    try:
        scanned = _scan_hosts(all_hosts)
    except Exception as e:
        logging.error(str(e))
        return 2

    # Assemble the config structure
    fleet_name = args.fleet_name
    cluster_id = int(args.cluster_id)

    # wakuNodes: use static hosts ENRs
    waku_nodes: List[str] = [scanned[h][0] for h in static_hosts if h in scanned]

    # discV5BootstrapNodes: from bootstrap hosts ENRs
    bootstrap_enrs: List[str] = [scanned[h][0] for h in bootstrap_hosts if h in scanned]

    # storeNodes: list of {id, enr, addr, fleet}
    store_nodes: List[Dict[str, str]] = []
    for h in store_hosts:
        if h not in scanned:
            continue
        enr, addr = scanned[h]
        node = {
            "id": h,
            "enr": enr,
            "fleet": fleet_name,
        }
        if addr:
            node["addr"] = addr
        store_nodes.append(node)

    output = {
        fleet_name: {
            "clusterId": cluster_id,
            "wakuNodes": waku_nodes,
            "discV5BootstrapNodes": bootstrap_enrs,
            "storeNodes": store_nodes,
        }
    }

    # Ensure destination directory exists
    os.makedirs(os.path.dirname(args.output or ".") or ".", exist_ok=True)

    with open(args.output, "w", encoding="utf-8") as f:
        json.dump(output, f, indent=2)
        f.write("\n")

    logging.info(f"Written fleet config for '{fleet_name}' to {args.output}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
