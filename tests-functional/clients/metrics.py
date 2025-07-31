import json
import logging
import os
import time
import statistics

from dataclasses import dataclass

import matplotlib
import matplotlib.pyplot as plt

from clients.expvar import GoMemoryStats

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
        go_metrics: list[GoMemoryStats] | None = None,
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

    def _calculate_go_metrics(self):
        # Convert to MB for consistency
        mb = 1024 * 1024

        total_memory = [metric.sys_bytes - metric.heap_released_bytes for metric in self.go_metrics]
        idle_memory = [metric.heap_idle_bytes - metric.heap_released_bytes for metric in self.go_metrics]

        if total_memory:
            self.total_memory_median = statistics.median(total_memory) / mb
            self.total_memory_max = max(total_memory) / mb
        if idle_memory:
            self.idle_memory_median = statistics.median(idle_memory) / mb
            self.idle_memory_max = max(idle_memory) / mb

        # Final GC count from the last sample
        self.final_gc_count = self.go_metrics[-1].num_gc

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
                    "heap_alloc_mb": {
                        "median": self.total_memory_median,
                        "max": self.total_memory_max,
                    },
                    "gc_count": self.final_gc_count,
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
        if not self.container_stats and not self.go_metrics:
            logging.warning("No performance data to generate chart")
            return None

        mb = 1024 * 1024
        start_time = 0

        try:
            # Create a figure with four subplots (CPU, Memory, Network, and Accumulated Network)
            fig, (ax1, ax2, ax3, ax4) = plt.subplots(4, 1, figsize=(12, 16), sharex=True)
            fig.suptitle(title)

            # Extract data from container stats
            if self.container_stats:
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
            else:
                container_time_points = []
                cpu_values = []
                ram_values = []
                rx_values = []
                tx_values = []
                rx_bytes = []
                tx_bytes = []
                rx_bytes_mb = []
                tx_bytes_mb = []

            # Extract data from Go metrics independently
            go_timestamps = [metric.timestamp for metric in self.go_metrics]
            sys_values = [metric.sys_bytes / mb for metric in self.go_metrics]
            actual_memory_values = [(metric.sys_bytes - metric.heap_released_bytes) / mb for metric in self.go_metrics]
            could_be_released = [(metric.heap_idle_bytes - metric.heap_released_bytes) / mb for metric in self.go_metrics]

            # Convert to relative time (use container start time if available, otherwise Go metrics start time)
            go_start_time = start_time if self.container_stats else go_timestamps[0]
            go_time_points = [t - go_start_time for t in go_timestamps]

            # CPU usage plot
            if cpu_values:
                cpu_median = statistics.median(cpu_values)
                cpu_max = max(cpu_values)
                ax1.plot(container_time_points, cpu_values, "b-", label=f"CPU Usage (%)\nmedian = {cpu_median:.2f}%\nmax = {cpu_max:.2f}%")

            ax1.set_ylabel("CPU Usage (%)")
            ax1.set_title("CPU Usage Over Time")
            ax1.grid(True)
            ax1.set_xlim(left=0)
            ax1.set_ylim(bottom=0)
            ax1.legend(loc="best")

            # Memory usage plot with independent arrays
            if ram_values:
                median_memory = statistics.median(ram_values)
                max_memory = max(ram_values)
                ax2.plot(
                    container_time_points,
                    ram_values,
                    "m-",
                    label=f"Container Memory (MB)\nmedian = {median_memory:.2f} MB\nmax = {max_memory:.2f} MB",
                )

            if sys_values:
                sys_median = statistics.median(sys_values)
                sys_max = max(sys_values)
                ax2.plot(
                    go_time_points,
                    sys_values,
                    "orange",
                    label=f"Go Sys Memory (MB)\nmedian = {sys_median:.2f} MB\nmax = {sys_max:.2f} MB",
                    linewidth=2,
                )

            actual_memory_median = statistics.median(actual_memory_values)
            actual_memory_max = max(actual_memory_values)
            ax2.plot(
                go_time_points,
                actual_memory_values,
                "g-",
                label=f"Go Actual Memory Usage (MB)\nmedian = {actual_memory_median:.2f} MB\nmax = {actual_memory_max:.2f} MB",
                linewidth=2,
            )

            ax2.plot(
                go_time_points,
                could_be_released,
                "b",
                label="Go Idle Memory (MB)",
                linewidth=2,
            )

            ax2.set_ylabel("Memory Usage (MB)")
            ax2.set_title("Memory Usage Over Time")
            ax2.grid(True)
            ax2.set_xlim(left=0)
            ax2.set_ylim(bottom=0)
            ax2.legend(loc="best")

            # Network usage plot
            if rx_values and tx_values:
                rx_median = statistics.median(rx_values)
                tx_median = statistics.median(tx_values)
                rx_max = max(rx_values)
                tx_max = max(tx_values)
                ax3.plot(
                    container_time_points,
                    rx_values,
                    "c-",
                    label=f"Download (MB/s)\nmedian = {rx_median:.2f} MB/s\nmax = {rx_max:.2f} MB/s",
                    linewidth=2,
                )
                ax3.plot(
                    container_time_points,
                    tx_values,
                    "r-",
                    label=f"Upload (MB/s)\nmedian = {tx_median:.2f} MB/s\nmax = {tx_max:.2f} MB/s",
                    linewidth=2,
                )
            ax3.set_ylabel("Network Throughput (MB/s)")
            ax3.set_title("Network Activity Over Time")
            ax3.grid(True)
            ax3.set_xlim(left=0)
            ax3.set_ylim(bottom=0)
            ax3.legend(loc="best", labelspacing=2)

            # Accumulated network usage plot
            if rx_bytes and tx_bytes:
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

            # Add vertical lines for events across all plots
            if self.events and hasattr(self.events, "events") and self.events.events:
                for event_name, event_timestamp in self.events.events.items():
                    # Convert the event timestamp to relative time (seconds from start)
                    if self.container_stats:
                        start_time = self.container_stats[0].timestamp
                        event_time = event_timestamp - start_time

                        # Only add lines for events that occur within our time range
                        if container_time_points and 0 <= event_time <= max(container_time_points):
                            # Add vertical line to all subplots
                            for ax in [ax1, ax2, ax3, ax4]:
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

            # Adjust layout
            plt.tight_layout(rect=(0, 0, 1, 1))

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

        except Exception as e:
            logging.error(f"Error generating performance chart: {e}")
            return None

    def save_go_metrics_chart(self, title: str, output_path=None):
        """Generate and save a dedicated Go metrics chart as a PNG image

        Args:
            title: Chart title
            output_path: Path to save the chart. If None, saves to ./go_metrics_{timestamp}.png

        Returns:
            str: Path to the saved chart file
        """
        if not self.go_metrics:
            logging.warning("No Go metrics data to generate chart")
            return None

        if output_path is None:
            output_path = f"./go_metrics_{int(time.time())}.png"

        mb = 1024 * 1024

        try:
            # Create a figure with three subplots for Go metrics
            fig, (ax1, ax2, ax3) = plt.subplots(3, 1, figsize=(12, 16), sharex=True)
            fig.suptitle(title)

            # Extract data from Go metrics
            go_timestamps = [metric.timestamp for metric in self.go_metrics]
            start_time = go_timestamps[0]
            go_time_points = [t - start_time for t in go_timestamps]

            # Memory-related metrics (in MB)
            alloc_values = [metric.alloc_bytes / mb for metric in self.go_metrics]
            sys_bytes_values = [metric.sys_bytes / mb for metric in self.go_metrics]
            heap_alloc_values = [metric.heap_alloc_bytes / mb for metric in self.go_metrics]
            heap_sys_values = [metric.heap_sys_bytes / mb for metric in self.go_metrics]
            heap_idle_values = [metric.heap_idle_bytes / mb for metric in self.go_metrics]
            heap_in_use_values = [metric.heap_in_use_bytes / mb for metric in self.go_metrics]
            heap_released_values = [metric.heap_released_bytes / mb for metric in self.go_metrics]

            # Count-based metrics
            heap_objects_values = [metric.heap_objects for metric in self.go_metrics]
            mallocs_values = [metric.mallocs for metric in self.go_metrics]
            frees_values = [metric.frees for metric in self.go_metrics]
            num_gc_values = [metric.num_gc for metric in self.go_metrics]

            # GC metrics
            gc_cpu_fraction_values = [metric.gc_cpu_fraction * 100 for metric in self.go_metrics]  # Convert to percentage

            # Plot 1: Memory metrics
            if alloc_values:
                alloc_median = statistics.median(alloc_values)
                alloc_max = max(alloc_values)
                ax1.plot(
                    go_time_points, alloc_values, "r-", label=f"Alloc (MB)\nmedian = {alloc_median:.2f} MB\nmax = {alloc_max:.2f} MB", linewidth=2
                )

            if heap_alloc_values:
                heap_alloc_median = statistics.median(heap_alloc_values)
                heap_alloc_max = max(heap_alloc_values)
                ax1.plot(
                    go_time_points,
                    heap_alloc_values,
                    "b-",
                    label=f"Heap Alloc (MB)\nmedian = {heap_alloc_median:.2f} MB\nmax = {heap_alloc_max:.2f} MB",
                    linewidth=2,
                )

            if heap_sys_values:
                heap_sys_median = statistics.median(heap_sys_values)
                heap_sys_max = max(heap_sys_values)
                ax1.plot(
                    go_time_points,
                    heap_sys_values,
                    "orange",
                    label=f"Heap Sys (MB)\nmedian = {heap_sys_median:.2f} MB\nmax = {heap_sys_max:.2f} MB",
                    linewidth=2,
                )

            if heap_idle_values:
                heap_idle_median = statistics.median(heap_idle_values)
                heap_idle_max = max(heap_idle_values)
                ax1.plot(
                    go_time_points,
                    heap_idle_values,
                    "c--",
                    label=f"Heap Idle (MB)\nmedian = {heap_idle_median:.2f} MB\nmax = {heap_idle_max:.2f} MB",
                    linewidth=1,
                )

            if heap_in_use_values:
                heap_in_use_median = statistics.median(heap_in_use_values)
                heap_in_use_max = max(heap_in_use_values)
                ax1.plot(
                    go_time_points,
                    heap_in_use_values,
                    "g--",
                    label=f"Heap In Use (MB)\nmedian = {heap_in_use_median:.2f} MB\nmax = {heap_in_use_max:.2f} MB",
                    linewidth=1,
                )

            if heap_released_values:
                heap_released_median = statistics.median(heap_released_values)
                heap_released_max = max(heap_released_values)
                ax1.plot(
                    go_time_points,
                    heap_released_values,
                    "purple",
                    linestyle="--",
                    label=f"Heap Released (MB)\nmedian = {heap_released_median:.2f} MB\nmax = {heap_released_max:.2f} MB",
                    linewidth=1,
                )

            if sys_bytes_values:
                total_alloc_median = statistics.median(sys_bytes_values)
                total_alloc_max = max(sys_bytes_values)
                ax1.plot(
                    go_time_points,
                    sys_bytes_values,
                    "brown",
                    linestyle=":",
                    label=f"Sys (MB)\nmedian = {total_alloc_median:.2f} MB\nmax = {total_alloc_max:.2f} MB",
                    linewidth=1,
                )

            ax1.set_ylabel("Memory Usage (MB)")
            ax1.set_title("Go Memory Metrics Over Time")
            ax1.grid(True)
            ax1.set_xlim(left=0)
            ax1.set_ylim(bottom=0)
            ax1.legend(loc="best")

            # Plot 2: Count-based metrics
            if heap_objects_values:
                heap_objects_median = statistics.median(heap_objects_values)
                heap_objects_max = max(heap_objects_values)
                ax2.plot(
                    go_time_points,
                    heap_objects_values,
                    "pink",
                    label=f"Heap Objects\nmedian = {heap_objects_median:.0f}\nmax = {heap_objects_max:.0f}",
                    linewidth=2,
                )

            if mallocs_values:
                mallocs_median = statistics.median(mallocs_values)
                mallocs_max = max(mallocs_values)
                ax2.plot(
                    go_time_points, mallocs_values, "gray", label=f"Mallocs\nmedian = {mallocs_median:.0f}\nmax = {mallocs_max:.0f}", linewidth=2
                )

            if frees_values:
                frees_median = statistics.median(frees_values)
                frees_max = max(frees_values)
                ax2.plot(go_time_points, frees_values, "lightgray", label=f"Frees\nmedian = {frees_median:.0f}\nmax = {frees_max:.0f}", linewidth=2)

            if num_gc_values:
                num_gc_median = statistics.median(num_gc_values)
                num_gc_max = max(num_gc_values)
                ax2.plot(go_time_points, num_gc_values, "black", label=f"GC Count\nmedian = {num_gc_median:.0f}\nmax = {num_gc_max:.0f}", linewidth=2)

            ax2.set_ylabel("Count")
            ax2.set_title("Go Count Metrics Over Time")
            ax2.grid(True)
            ax2.set_xlim(left=0)
            ax2.set_ylim(bottom=0)
            ax2.legend(loc="best")

            # Plot 3: GC CPU fraction
            if gc_cpu_fraction_values:
                gc_cpu_median = statistics.median(gc_cpu_fraction_values)
                gc_cpu_max = max(gc_cpu_fraction_values)
                ax3.plot(
                    go_time_points,
                    gc_cpu_fraction_values,
                    "red",
                    label=f"GC CPU Fraction (%)\nmedian = {gc_cpu_median:.4f}%\nmax = {gc_cpu_max:.4f}%",
                    linewidth=2,
                )

            ax3.set_ylabel("GC CPU Fraction (%)")
            ax3.set_title("Go GC CPU Usage Over Time")
            ax3.set_xlabel("Time (seconds)")
            ax3.grid(True)
            ax3.set_xlim(left=0)
            ax3.set_ylim(bottom=0)
            ax3.legend(loc="best")

            plt.tight_layout()
            plt.savefig(output_path, dpi=150, bbox_inches="tight")
            plt.close()

            logging.info(f"Go metrics chart saved to {output_path}")
            return output_path

        except Exception as e:
            logging.error(f"Error generating Go metrics chart: {e}")
            return None

    def save_to_file(self, filename: str):
        metrics = self.to_dict()
        os.makedirs(os.path.dirname(filename), exist_ok=True)
        with open(filename, "w") as f:
            json.dump(metrics, f, indent=2)
        logging.info(f"Performance report saved to {filename}")
