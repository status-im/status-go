import os

class Config:
    # Get the project root directory based on the location of the config file
    PROJECT_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))

    # Status Backend Configurations
    STATUS_BACKEND_URL = os.getenv("STATUS_BACKEND_URL", "http://127.0.0.1")
    API_REQUEST_TIMEOUT = int(os.getenv("API_REQUEST_TIMEOUT", "15"))

    # Paths (Relative to Project Root)
    DATA_DIR = os.path.join(PROJECT_ROOT, "local")
    LOCAL_DATA_DIR1 = os.path.join(DATA_DIR, "data1")
    LOCAL_DATA_DIR2 = os.path.join(DATA_DIR, "data2")
    RESOURCES_FOLDER = os.path.join(PROJECT_ROOT, "resources")

    # Payloads
    ACCOUNT_PAYLOAD_DEFAULTS = {
        "displayName": "user",
        "password": "test_password",
        "customizationColor": "primary"
    }

    # Commands (For network emulation)
    LATENCY_CMD = "sudo tc qdisc add dev eth0 root netem delay 1s 100ms distribution normal"
    PACKET_LOSS_CMD = "sudo tc qdisc add dev eth0 root netem loss 50%"
    LOW_BANDWIDTH_CMD = "sudo tc qdisc add dev eth0 root tbf rate 1kbit burst 1kbit"
    REMOVE_TC_CMD = "sudo tc qdisc del dev eth0 root"