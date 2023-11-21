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

        test_stats[test_name]["total"] += 1

        if testcase.find("failure") is not None:
            test_stats[test_name]["failed"] += 1
        elif testcase.find("error") is not None:
            test_stats[test_name]["failed"] += 1

failing_test_stats = [
    {"name": name, "failure_rate": stats["failed"] / stats["total"]}
    for name, stats in test_stats.items() if stats["failed"] != 0
]

sorted_failing_test_stats = sorted(failing_test_stats,
                                   key=lambda x: x["failure_rate"],
                                   reverse=True)

print("---")
print("Failing tests stats")
print("(test name: failure rate)")
print("---")
for test_stat in sorted_failing_test_stats:
    print(f"{test_stat['name']}: {test_stat['failure_rate'] * 100}%")
