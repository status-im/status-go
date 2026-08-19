import logging
import threading
import time
from dataclasses import dataclass
from typing import Optional

import requests


@dataclass
class GoMemoryStats:
    """Memory statistics from Go's expvar /debug/vars endpoint"""

    alloc_bytes: int  # Currently allocated bytes
    total_alloc_bytes: int  # Total bytes allocated (cumulative)
    sys_bytes: int  # Bytes obtained from OS
    mallocs: int  # Number of mallocs
    frees: int  # Number of frees
    heap_alloc_bytes: int  # Heap allocated bytes
    heap_sys_bytes: int  # Heap system bytes
    heap_idle_bytes: int  # Heap idle bytes
    heap_in_use_bytes: int  # Heap in-use bytes
    heap_released_bytes: int  # Heap released bytes
    heap_objects: int  # Number of heap objects
    gc_cpu_fraction: float  # GC CPU fraction
    num_gc: int  # Number of GC runs


class ExpvarClient:
    """Client for collecting Go memory metrics via /debug/vars endpoint"""

    def __init__(self, base_url: str):
        """
        Initialize expvar client

        Args:
            base_url: Base URL of the application (e.g., "http://localhost:8080")
        """
        self.base_url = base_url.rstrip("/")
        self.go_metrics = []
        self._stop_monitoring = None
        self.monitor_thread = None

    def get_expvars(self, timeout: int = 10) -> Optional[dict]:
        """
        Get memory statistics from /debug/vars endpoint

        Args:
            timeout: Request timeout in seconds

        Returns:
            MemoryStats object or None if request fails
        """
        try:
            response = requests.get(f"{self.base_url}/debug/vars", timeout=timeout)
            response.raise_for_status()
            data = response.json()
            data["timestamp"] = time.time()
            return data

        except requests.exceptions.RequestException as e:
            raise e
        except Exception as e:
            logging.error(f"Error parsing /debug/vars response: {e}")
            return None

    @staticmethod
    def parse_expvars(data):
        memstats = data.get("memstats", {})
        if not memstats:
            raise ValueError("memstats not found in /debug/vars response")

        return GoMemoryStats(
            alloc_bytes=memstats.get("Alloc", 0),
            total_alloc_bytes=memstats.get("TotalAlloc", 0),
            sys_bytes=memstats.get("Sys", 0),
            mallocs=memstats.get("Mallocs", 0),
            frees=memstats.get("Frees", 0),
            heap_alloc_bytes=memstats.get("HeapAlloc", 0),
            heap_sys_bytes=memstats.get("HeapSys", 0),
            heap_idle_bytes=memstats.get("HeapIdle", 0),
            heap_in_use_bytes=memstats.get("HeapInuse", 0),
            heap_released_bytes=memstats.get("HeapReleased", 0),
            heap_objects=memstats.get("HeapObjects", 0),
            gc_cpu_fraction=memstats.get("GCCPUFraction", 0.0),
            num_gc=memstats.get("NumGC", 0),
        )

    @staticmethod
    def parse_num_goroutines(data):
        return data.get("numGoroutine", 0)

    @staticmethod
    def parse_num_threads(data):
        return data.get("numThreads", 0)

    @staticmethod
    def parse_timestamp(data):
        return data.get("timestamp", 0)

    def start_monitoring(self, interval: float = 1.0):
        """Start independent Go metrics monitoring thread

        Args:
            interval: Monitoring interval in seconds
        """
        self.go_metrics = []
        self._stop_monitoring = threading.Event()

        def monitor_go_metrics():
            while self._stop_monitoring and not self._stop_monitoring.is_set():
                try:
                    expvars = self.get_expvars()
                    if expvars:
                        self.go_metrics.append(expvars)

                    # Wait for the specified interval or until stop event is set
                    self._stop_monitoring.wait(timeout=interval)
                except Exception as e:
                    logging.error(f"Go metrics monitoring error: {e}")
                    self._stop_monitoring.set()

        self._stop_monitoring.clear()
        self.monitor_thread = threading.Thread(target=monitor_go_metrics, daemon=True)
        self.monitor_thread.start()
        logging.info("Started Go metrics monitoring")

    def stop_monitoring(self):
        """Stop the Go metrics monitoring thread and return collected metrics"""
        if self._stop_monitoring:
            self._stop_monitoring.set()

        if self.monitor_thread and self.monitor_thread.is_alive():
            self.monitor_thread.join(timeout=10)
            if self.monitor_thread.is_alive():
                logging.warning("Go metrics monitoring thread didn't stop gracefully")

        return self.go_metrics
