# Reliability Test Suite

This test suite, located under `tests-functional/tests/reliability`, is designed to evaluate the robustness and stability of the system under challenging network and runtime conditions. These tests go beyond typical functional checks and simulate real-world scenarios that stress the system's resilience.

## 🧪 What We Test

The suite includes tests that simulate:

* **Adverse network conditions**:

  * High latency
  * Packet loss
  * Low bandwidth
* **Node lifecycle disruptions**:

  * Runtime IP address changes
  * Node pause/unpause behavior
* **Network protocols**:

  * IPv4 and IPv6 coverage
* **System rate limiting**:

  * Testing throttling mechanisms under stress

## 🚀 How to Run

Running the reliability tests is nearly identical to running functional tests see [how-to-run guide](https://github.com/status-im/status-go/blob/develop/tests-functional/README.MD#how-to-run).
Example of running a test:

```bash
pytest tests-functional/tests/reliability/ -k test_one_to_one_message_with_latency
```

Note! To run locally you need [`tc`](https://man7.org/linux/man-pages/man8/tc.8.html) (ex: wasn't able to run it on `windows` or `wsl`) on your host.

## 🕒 CI Integration

These tests are executed:

* **Nightly** on a schedule
* **On demand**, via GitHub Actions

Workflow link: [test-reliability.yml](https://github.com/status-im/status-go/actions/workflows/test-reliability.yml)

Both **IPv4** and **IPv6** test runs are included in CI to ensure broad network compatibility.

## 📊 Reporting

Test results are available in the **GitHub Actions summary** of each run. For example:

🔗 [Example run report](https://github.com/status-im/status-go/actions/runs/15748011404)

While the reporting is currently minimal, it provides quick insight into which tests passed or failed, and under what conditions.
