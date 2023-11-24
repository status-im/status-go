#!/usr/bin/env python

import glob
import xml.etree.ElementTree as ET
from collections import defaultdict

test_stats = defaultdict(lambda: defaultdict(int))

for file in glob.glob("**/report.xml", recursive=True):
    tree = ET.parse(file)
    root = tree.getroot()
    for testcase in root.iter("testcase"):
        test_name = testcase.attrib["name"]

        test_stats[test_name]["total_runs"] += 1

        if testcase.find("failure") is not None:
            test_stats[test_name]["failed_runs"] += 1
        elif testcase.find("error") is not None:
            test_stats[test_name]["failed_runs"] += 1

failing_test_stats = [
    {
        "name": name,
        "failure_rate": stats["failed_runs"] / stats["total_runs"],
        "failed_runs": stats["failed_runs"],
        "total_runs": stats["total_runs"]
    }
    for name, stats in test_stats.items() if stats["failed_runs"] != 0
]

sorted_failing_test_stats = sorted(failing_test_stats,
                                   key=lambda x: x["failure_rate"],
                                   reverse=True)

print("---")
print("Failing tests stats")
print("---")
for test_stat in sorted_failing_test_stats:
    print("{}: {}% ({} of {} failed)".format(
        test_stat['name'],
        test_stat['failure_rate'] * 100,
        test_stat['failed_runs'],
        test_stat['total_runs']
    ))
