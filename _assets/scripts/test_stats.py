#!/usr/bin/env python

import glob
import xml.etree.ElementTree as ET
from collections import defaultdict
import re

test_stats = defaultdict(lambda: defaultdict(int))
skipped_tests = {}  # Use a dictionary to store test names and their skip reasons

file_path = "**/report_*.xml"

for file in glob.glob(file_path, recursive=True):
    tree = ET.parse(file)
    root = tree.getroot()
    for testcase in root.iter("testcase"):
        test_name = testcase.attrib["name"]

        test_stats[test_name]["total_runs"] += 1

        if testcase.find("failure") is not None:
            test_stats[test_name]["failed_runs"] += 1
        elif testcase.find("error") is not None:
            test_stats[test_name]["failed_runs"] += 1

        # Check for skipped tests
        skipped_element = testcase.find("skipped")
        if skipped_element is not None:
            message = skipped_element.attrib.get("message", "")
            # Extract the real reason from the message
            match = re.search(r': (.*?)\s*--- SKIP', message)
            skip_reason = match.group(1).strip() if match else "unknown reason"
            skipped_tests[test_name] = skip_reason  # Store test name and skip reason

# Filter out root test cases if they have subtests
filtered_test_stats = {
    name: stats for name, stats in test_stats.items()
    if not any(subtest.startswith(name + "/") for subtest in test_stats)
}

failing_test_stats = [
    {
        "name": name,
        "failure_rate": stats["failed_runs"] / stats["total_runs"],
        "failed_runs": stats["failed_runs"],
        "total_runs": stats["total_runs"]
    }
    for name, stats in filtered_test_stats.items() if stats["failed_runs"] != 0
]

sorted_failing_test_stats = sorted(failing_test_stats,
                                   key=lambda x: x["failure_rate"],
                                   reverse=True)

flaky_skipped_count = sum(1 for reason in skipped_tests.values() if reason == "flaky test")

print("---")
print(f"Failing tests stats (total: {len(failing_test_stats)})")
print("---")
for test_stat in sorted_failing_test_stats:
    print("{}: {:.1f}% ({} of {} failed)".format(
        test_stat['name'],
        test_stat['failure_rate'] * 100,
        test_stat['failed_runs'],
        test_stat['total_runs']
    ))

print("---")
print(f"Skipped tests (total: {len(skipped_tests)}, skipped as flaky: {flaky_skipped_count})")
print("---")
for test_name, skip_reason in skipped_tests.items():
    print(f"{test_name}: {skip_reason}")
