import json
import logging
import os
import statistics
import time
from dataclasses import dataclass

import matplotlib
import matplotlib.pyplot as plt

from clients.expvar import ExpvarClient

matplotlib.use("Agg")  # Use non-interactive backend
logging.getLogger("matplotlib.font_manager").setLevel(logging.WARNING)


@dataclass
class CPUMetrics:
    cpu_percent: float
    cpu_count: int


@dataclass
class RAMMetrics:
    memory_usage_mb: float
    memory_max_usage_mb: float


@dataclass
class NetworkMetrics:
    rx_bytes: int  # Received bytes
    tx_bytes: int  # Transmitted bytes
    rx_packets: int  # Received packets
    tx_packets: int  # Transmitted packets
    rx_dropped: int  # Received packets dropped
    tx_dropped: int  # Transmitted packets dropped
    rx_errors: int  # Receive errors
    tx_errors: int  # Transmit errors
    rx_bytes_per_sec: float = 0  # Bytes per second received
    tx_bytes_per_sec: float = 0  # Bytes per second transmitted


@dataclass
class GoMemStats:
    idle_memory_mb: float  # Heap idle memory in MB
    heap_alloc_mb: float  # Currently allocated heap memory in MB
    heap_sys_mb: float  # Heap system memory in MB
    heap_in_use_mb: float  # Heap in-use memory in MB
    num_gc: int  # Number of GC runs
    gc_cpu_fraction: float  # GC CPU fraction


class Events:
    def __init__(self):
        self.events = {}

    def append(self, event: str):
        logging.info(f"Metrics event: {event}")
        self.events[event] = time.time()

    def __iter__(self):
        return iter(self.events)

    def to_dict(self):
        return self.events


def calculate_cpu_metrics(stats):
    # CPU Usage fields
    cpu_stats = stats["cpu_stats"]
    precpu_stats = stats["precpu_stats"]

    # Total CPU usage in nanoseconds
    cpu_total = cpu_stats["cpu_usage"]["total_usage"]
    cpu_total_prev = precpu_stats["cpu_usage"]["total_usage"]

    # System CPU usage in nanoseconds
    system_total = cpu_stats.get("system_cpu_usage", 0)
    system_total_prev = precpu_stats.get("system_cpu_usage", 0)

    # CPU cores
    try:
        try:
            cpu_count = len(cpu_stats["cpu_usage"]["percpu_usage"])
        except KeyError:
            cpu_count = cpu_stats["online_cpus"]
    except KeyError:
        cpu_count = 1

    # Calculate deltas
    cpu_delta = cpu_total - cpu_total_prev
    system_delta = system_total - system_total_prev

    # Calculate percentages
    cpu_percent = 0.0
    if system_delta > 0 and cpu_delta > 0:
        cpu_percent = (cpu_delta / system_delta) * cpu_count * 100.0

    return CPUMetrics(
        cpu_percent=cpu_percent,
        cpu_count=cpu_count,
    )


def calculate_memory_metrics(stats):
    usage = stats["memory_stats"]["usage"]
    max_usage = stats["memory_stats"].get("max_usage", usage)  # Use current usage as fallback

    # Convert to MB for readability
    mb = 1024 * 1024
    return RAMMetrics(
        memory_usage_mb=usage / mb,
        memory_max_usage_mb=max_usage / mb,
    )


def calculate_network_metrics(stats, prev_stats=None):
    """Calculate network metrics from Docker stats

    Args:
        stats: Current Docker stats containing network information
        prev_stats: Previous stats for calculating rates (optional)

    Returns:
        NetworkMetrics: Network statistics
    """
    network_stats = stats.get("networks", {})

    # Initialize totals
    total_rx_bytes = 0
    total_tx_bytes = 0
    total_rx_packets = 0
    total_tx_packets = 0
    total_rx_dropped = 0
    total_tx_dropped = 0
    total_rx_errors = 0
    total_tx_errors = 0

    # Sum up all network interfaces
    for interface_name, interface_stats in network_stats.items():
        total_rx_bytes += interface_stats.get("rx_bytes", 0)
        total_tx_bytes += interface_stats.get("tx_bytes", 0)
        total_rx_packets += interface_stats.get("rx_packets", 0)
        total_tx_packets += interface_stats.get("tx_packets", 0)
        total_rx_dropped += interface_stats.get("rx_dropped", 0)
        total_tx_dropped += interface_stats.get("tx_dropped", 0)
        total_rx_errors += interface_stats.get("rx_errors", 0)
        total_tx_errors += interface_stats.get("tx_errors", 0)

    # Calculate rates if previous stats are available
    rx_bytes_per_sec = 0
    tx_bytes_per_sec = 0

    if prev_stats is not None:
        prev_network_stats = prev_stats.get("networks", {})
        prev_rx_bytes = 0
        prev_tx_bytes = 0

        for interface_name, interface_stats in prev_network_stats.items():
            prev_rx_bytes += interface_stats.get("rx_bytes", 0)
            prev_tx_bytes += interface_stats.get("tx_bytes", 0)

        # Calculate time difference
        current_time = stats.get("read", "")
        prev_time = prev_stats.get("read", "")

        if current_time and prev_time:
            try:
                from datetime import datetime

                current_dt = datetime.fromisoformat(current_time.replace("Z", "+00:00"))
                prev_dt = datetime.fromisoformat(prev_time.replace("Z", "+00:00"))
                time_diff = (current_dt - prev_dt).total_seconds()
            except (ValueError, AttributeError):
                # If timestamp parsing fails, use a default time difference
                time_diff = 1.0
        else:
            # Fallback: assume 1 second interval if no timestamps
            time_diff = 1.0

        # Calculate rates
        if time_diff > 0:
            rx_bytes_per_sec = (total_rx_bytes - prev_rx_bytes) / time_diff
            tx_bytes_per_sec = (total_tx_bytes - prev_tx_bytes) / time_diff
        else:
            # If time_diff is 0 or negative, use the difference as bytes per second
            rx_bytes_per_sec = total_rx_bytes - prev_rx_bytes
            tx_bytes_per_sec = total_tx_bytes - prev_tx_bytes

    return NetworkMetrics(
        rx_bytes=total_rx_bytes,
        tx_bytes=total_tx_bytes,
        rx_packets=total_rx_packets,
        tx_packets=total_tx_packets,
        rx_dropped=total_rx_dropped,
        tx_dropped=total_tx_dropped,
        rx_errors=total_rx_errors,
        tx_errors=total_tx_errors,
        rx_bytes_per_sec=max(0, rx_bytes_per_sec),  # Ensure non-negative
        tx_bytes_per_sec=max(0, tx_bytes_per_sec),  # Ensure non-negative
    )


@dataclass
class ContainerStats:
    """Container stats object"""

    def __init__(self, stat, prev_stat=None, go_memory_stats=None):
        self.timestamp = time.time()
        self.cpu = calculate_cpu_metrics(stat)
        self.ram = calculate_memory_metrics(stat)
        self.network = calculate_network_metrics(stat, prev_stat)
        self.expvars = go_memory_stats


@dataclass
class StatusGoMetrics:
    # Container for performance monitoring metrics
    duration = 0
    samples = 0
    cpu_median = 0
    cpu_max = 0
    ram_median = 0
    ram_max = 0
    rx_bytes_per_sec_median = 0
    rx_bytes_per_sec_max = 0
    rx_total_bytes = 0
    ex_total_packets = 0
    tx_bytes_per_sec_median = 0
    tx_bytes_per_sec_max = 0
    tx_total_bytes = 0
    tx_total_packets = 0
    total_network_errors = 0

    # Expvars metrics
    total_memory_median = 0
    total_memory_max = 0
    idle_memory_median = 0  # "Idle" that is kepy by Go and not released to OS
    idle_memory_max = 0
    final_gc_count = 0
    timestamp = 0
    version = ""

    def __init__(
        self,
        container_stats: list[ContainerStats] | None = None,
        go_metrics: list[dict] | None = None,
        events: Events | None = None,
        version: str = "",
        stats: list[ContainerStats] | None = None,
    ):
        """
        Initialize PerformanceMetrics with independent arrays

        Args:
            container_stats: List of container statistics with their own timestamps
            go_metrics: List of Go memory statistics with their own timestamps
            events: Events tracker
            version: Version string
            stats: Legacy parameter for backward compatibility
        """
        # Handle backward compatibility
        if stats is not None and container_stats is None:
            container_stats = stats

        self.container_stats = container_stats or []
        self.go_metrics = go_metrics or []
        self._memory_stats = [ExpvarClient.parse_expvars(metric) for metric in self.go_metrics]
        self.events = events or Events()
        self.timestamp = time.time()
        self.version = version

        self._calculate_metrics()

    def _calculate_container_metrics(self):
        # Calculate duration from container stats
        self.duration = self.container_stats[-1].timestamp - self.container_stats[0].timestamp

        # Extract CPU and RAM metrics
        cpu_percents = [stat.cpu.cpu_percent for stat in self.container_stats]
        ram_usage = [stat.ram.memory_usage_mb for stat in self.container_stats]

        # Extract network metrics
        rx_bytes_per_sec = [stat.network.rx_bytes_per_sec for stat in self.container_stats]
        tx_bytes_per_sec = [stat.network.tx_bytes_per_sec for stat in self.container_stats]

        self.samples = len(self.container_stats)
        self.cpu_median = statistics.median(cpu_percents)
        self.cpu_max = max(cpu_percents)
        self.ram_median = statistics.median(ram_usage)
        self.ram_max = max(ram_usage)

        # Network metrics
        self.rx_bytes_per_sec_median = statistics.median(rx_bytes_per_sec)
        self.rx_bytes_per_sec_max = max(rx_bytes_per_sec)
        self.tx_bytes_per_sec_median = statistics.median(tx_bytes_per_sec)
        self.tx_bytes_per_sec_max = max(tx_bytes_per_sec)

        # Total network statistics from the last sample
        last_stat = self.container_stats[-1]
        self.rx_total_bytes = last_stat.network.rx_bytes
        self.tx_total_bytes = last_stat.network.tx_bytes
        self.ex_total_packets = last_stat.network.rx_packets
        self.tx_total_packets = last_stat.network.tx_packets
        self.total_network_errors = (
            last_stat.network.rx_errors + last_stat.network.tx_errors + last_stat.network.rx_dropped + last_stat.network.tx_dropped
        )

    def _calculate_memory_stats(self):
        """Calculate memory statistics from collected go metrics"""

        # Convert to MB for consistency
        mb = 1024 * 1024

        total_memory = [metric.sys_bytes - metric.heap_released_bytes for metric in self._memory_stats]
        idle_memory = [metric.heap_idle_bytes - metric.heap_released_bytes for metric in self._memory_stats]

        if total_memory:
            self.total_memory_median = statistics.median(total_memory) / mb
            self.total_memory_max = max(total_memory) / mb
        if idle_memory:
            self.idle_memory_median = statistics.median(idle_memory) / mb
            self.idle_memory_max = max(idle_memory) / mb

        # Final GC count from the last sample
        self.final_gc_count = self._memory_stats[-1].num_gc

    def _calculate_go_metrics(self):
        """Calculate summary metrics from collected Go memory data"""
        self._calculate_memory_stats()
        self._num_goroutines = [ExpvarClient.parse_num_goroutines(metric) for metric in self.go_metrics]
        self.num_goroutines_max = max(self._num_goroutines)
        self._num_threads = [ExpvarClient.parse_num_threads(metric) for metric in self.go_metrics]
        self.num_threads_max = max(self._num_threads)

    def _calculate_metrics(self):
        """Calculate summary metrics from collected data"""
        if self.container_stats:
            self._calculate_container_metrics()
        if self.go_metrics:
            self._calculate_go_metrics()

    def to_dict(self):
        """Convert PerformanceMetrics to a JSON-serializable dictionary"""
        return {
            "timestamp": self.timestamp,
            "version": self.version,
            "events": self.events.to_dict(),
            "metrics": {
                "cpu": {
                    "median": self.cpu_median,
                    "max": self.cpu_max,
                },
                "ram": {
                    "median": self.ram_median,
                    "max": self.ram_max,
                },
                "network": {
                    "rx": {
                        "bytes_per_sec": {
                            "median": self.rx_bytes_per_sec_median,
                            "max": self.rx_bytes_per_sec_max,
                        },
                        "total_bytes": self.rx_total_bytes,
                        "total_packets": self.ex_total_packets,
                    },
                    "tx": {
                        "bytes_per_sec": {
                            "median": self.tx_bytes_per_sec_median,
                            "max": self.tx_bytes_per_sec_max,
                        },
                        "total_bytes": self.tx_total_bytes,
                        "total_packets": self.tx_total_packets,
                    },
                    "total_errors": self.total_network_errors,
                },
                "expvar": {
                    "idle_memory_mb": {
                        "median": self.idle_memory_median,
                        "max": self.idle_memory_max,
                    },
                    "total_memory_mb": {
                        "median": self.total_memory_median,
                        "max": self.total_memory_max,
                    },
                    "gc_count": self.final_gc_count,
                    "num_goroutines_max": self.num_goroutines_max,
                    "num_threads_max": self.num_threads_max,
                },
            },
        }

    def save_performance_chart(self, title: str, output_path=None):
        """Generate and save a performance chart as a PNG image

        Args:
            title: Chart title
            output_path: Path to save the chart. If None, saves to ./performance_metrics_{container_id}.png

        Returns:
            str: Path to the saved chart file
        """
        if not self.container_stats or not self.go_metrics:
            raise ValueError("No performance data to generate chart")

        mb = 1024 * 1024

        # Create a figure with four subplots (CPU, Memory, Network, and Accumulated Network)
        fig, (ax1, ax2, ax3, ax4, ax5, ax6) = plt.subplots(6, 1, figsize=(12, 18), sharex=True)
        fig.suptitle(title, fontsize=16, y=0.98)

        # Extract data from container stats
        container_timestamps = [stat.timestamp for stat in self.container_stats]
        cpu_values = [stat.cpu.cpu_percent for stat in self.container_stats]
        ram_values = [stat.ram.memory_usage_mb for stat in self.container_stats]
        rx_values = [stat.network.rx_bytes_per_sec / mb for stat in self.container_stats]
        tx_values = [stat.network.tx_bytes_per_sec / mb for stat in self.container_stats]

        # Convert to relative time
        start_time = container_timestamps[0]
        container_time_points = [t - start_time for t in container_timestamps]

        # Extract accumulated network data
        rx_bytes = [stat.network.rx_bytes for stat in self.container_stats]
        tx_bytes = [stat.network.tx_bytes for stat in self.container_stats]
        rx_bytes_mb = [bytes / mb for bytes in rx_bytes]
        tx_bytes_mb = [bytes / mb for bytes in tx_bytes]

        # Extract data from Go metrics independently
        go_timestamps = [ExpvarClient.parse_timestamp(metric) for metric in self.go_metrics]
        sys_values = [metric.sys_bytes / mb for metric in self._memory_stats]
        actual_memory_values = [(metric.sys_bytes - metric.heap_released_bytes) / mb for metric in self._memory_stats]
        could_be_released = [(metric.heap_idle_bytes - metric.heap_released_bytes) / mb for metric in self._memory_stats]

        # Convert to relative time (use container start time if available, otherwise Go metrics start time)
        go_start_time = start_time if self.container_stats else go_timestamps[0]
        go_time_points = [t - go_start_time for t in go_timestamps]

        # CPU usage plot
        cpu_median = statistics.median(cpu_values)
        cpu_max = max(cpu_values)
        ax1.plot(container_time_points, cpu_values, "b-", label="CPU Usage (%)")
        ax1.set_ylabel("CPU Usage (%)")
        ax1.set_title("CPU Usage Over Time")
        ax1.grid(True)
        ax1.set_xlim(left=0)
        ax1.set_ylim(bottom=0)
        ax1.legend(loc="best")

        # Memory usage plot with independent arrays
        median_memory = statistics.median(ram_values)
        max_memory = max(ram_values)
        ax2.plot(container_time_points, ram_values, "m-", label="Container Memory (MB)")

        sys_median = statistics.median(sys_values)
        sys_max = max(sys_values)
        ax2.plot(go_time_points, sys_values, "orange", label="Go Sys Memory (MB)", linewidth=2)

        actual_memory_median = statistics.median(actual_memory_values)
        actual_memory_max = max(actual_memory_values)
        ax2.plot(go_time_points, actual_memory_values, "g-", label="Go Actual Memory Usage (MB)", linewidth=2)
        ax2.plot(go_time_points, could_be_released, "b", label="Go Idle Memory (MB)", linewidth=2)

        ax2.set_ylabel("Memory Usage (MB)")
        ax2.set_title("Memory Usage Over Time")
        ax2.grid(True)
        ax2.set_xlim(left=0)
        ax2.set_ylim(bottom=0)
        ax2.legend(loc="best")

        # Network usage plot
        rx_median = statistics.median(rx_values)
        tx_median = statistics.median(tx_values)
        rx_max = max(rx_values)
        tx_max = max(tx_values)
        ax3.plot(container_time_points, rx_values, "c-", label="Download (MB/s)", linewidth=2)
        ax3.plot(container_time_points, tx_values, "r-", label="Upload (MB/s)", linewidth=2)
        ax3.set_ylabel("Network Throughput (MB/s)")
        ax3.set_title("Network Activity Over Time")
        ax3.grid(True)
        ax3.set_xlim(left=0)
        ax3.set_ylim(bottom=0)
        ax3.legend(loc="best", labelspacing=2)

        # Accumulated network usage plot
        rx_total_bytes = rx_bytes[-1]
        tx_total_bytes = tx_bytes[-1]
        ax4.plot(container_time_points, rx_bytes_mb, "c-", label=f"Download (MB), total: {rx_total_bytes / mb:.2f} MB", linewidth=2)
        ax4.plot(container_time_points, tx_bytes_mb, "r-", label=f"Upload (MB), total: {tx_total_bytes / mb:.2f} MB", linewidth=2)
        ax4.set_xlabel("Time (seconds)")
        ax4.set_ylabel("Total Data Transferred (MB)")
        ax4.set_title("Accumulated Network Data Over Time")
        ax4.grid(True)
        ax4.set_xlim(left=0)
        ax4.set_ylim(bottom=0)
        ax4.legend(loc="best")

        # Number of goroutines plot
        ax5.plot(go_time_points, self._num_goroutines, "g-", label="Number of Goroutines", linewidth=2)
        ax5.plot(go_time_points, self._num_threads, "b-", label="Number of Threads", linewidth=2)
        ax5.set_xlabel("Time (seconds)")
        ax5.set_ylabel("Numbers")
        ax5.set_title("Various Numbers Over Time")
        ax5.grid(True)
        ax5.set_xlim(left=0)
        ax5.set_ylim(bottom=0)
        ax5.legend(loc="best")

        # Add vertical lines for events across all plots
        if self.events and hasattr(self.events, "events") and self.events.events:
            for event_name, event_timestamp in self.events.events.items():
                # Convert the event timestamp to relative time (seconds from start)
                event_time = event_timestamp - start_time

                # Only add lines for events that occur within our time range
                if container_time_points and 0 <= event_time <= max(container_time_points):
                    # Add vertical line to all subplots
                    for ax in [ax1, ax2, ax3, ax4, ax5]:
                        ax.axvline(x=event_time, color="black", linestyle="--", alpha=0.7, linewidth=1)

                    # Add an event label to the top plot (CPU) to avoid cluttering
                    ax1.text(
                        event_time,
                        ax1.get_ylim()[1] * 0.95,
                        event_name,
                        rotation=90,
                        verticalalignment="top",
                        horizontalalignment="right",
                        fontsize=8,
                        color="black",
                        alpha=0.8,
                    )

        # Create consolidated statistical summary outside plots
        stats_text = "Performance Statistics:\n"
        stats_text += f"- CPU Usage: median = {cpu_median:.2f}%, max = {cpu_max:.2f}%\n"
        stats_text += f"- Container Memory: median = {median_memory:.2f} MB, max = {max_memory:.2f} MB\n"
        stats_text += f"- Go Sys Memory: median = {sys_median:.2f} MB, max = {sys_max:.2f} MB\n"
        stats_text += f"- Go Actual Memory: median = {actual_memory_median:.2f} MB, max = {actual_memory_max:.2f} MB\n"
        stats_text += f"- Network Download: median = {rx_median:.2f} MB/s, max = {rx_max:.2f} MB/s\n"
        stats_text += f"- Network Upload: median = {tx_median:.2f} MB/s, max = {tx_max:.2f} MB/s"

        # Adjust layout to make room for the statistics text at the bottom
        plt.tight_layout(rect=(0, 0.15, 1, 1))

        ax6.axis("off")
        ax6.invert_yaxis()
        ax6.text(0.5, 0.5, stats_text, verticalalignment="top")

        # Save the figure
        if output_path is None:
            timestamp = time.strftime("%Y%m%d-%H%M%S")
            output_path = f"./performance_metrics_{timestamp}.png"

        # Ensure directory exists
        os.makedirs(os.path.dirname(os.path.abspath(output_path)), exist_ok=True)

        # Save figure
        plt.savefig(output_path, dpi=100, bbox_inches="tight")
        plt.close(fig)

        logging.info(f"Performance chart saved to {output_path}")
        return output_path

    def save_to_file(self, filename: str):
        metrics = self.to_dict()
        os.makedirs(os.path.dirname(filename), exist_ok=True)
        with open(filename, "w") as f:
            json.dump(metrics, f, indent=2)
        logging.info(f"Performance report saved to {filename}")
