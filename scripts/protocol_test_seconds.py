#!/usr/bin/env python
"""Prints the weights run_unit_tests.sh uses to give every shard of the
internal/protocol tests a similar share of the work: the seconds each slow
top-level test took, from the JUnit reports of a unit test run, and one
`default` line for all the others.

    scripts/protocol_test_seconds.py report_*.xml > scripts/protocol_test_seconds.txt
"""
import sys
import xml.etree.ElementTree as ET

PACKAGE = "github.com/status-im/status-go/internal/protocol"
# Tests at least this slow get a line of their own. With ten shards the others
# move the longest one by a few seconds only.
SLOW_SECONDS = 30

seconds = {}
for path in sys.argv[1:]:
    for suite in ET.parse(path).getroot().iter("testsuite"):
        if suite.get("name") != PACKAGE:
            continue
        for case in suite.iter("testcase"):
            name = case.get("name")
            if "/" in name:
                continue
            # A rerun reports the test again; the slowest attempt is the safe estimate.
            seconds[name] = max(seconds.get(name, 0), float(case.get("time") or 0))

if not seconds:
    sys.exit(f"no {PACKAGE} tests in the given reports")

slow = {name: time for name, time in seconds.items() if time >= SLOW_SECONDS}
others = [time for name, time in seconds.items() if name not in slow]

print(f"# Seconds per top-level test of internal/protocol that takes {SLOW_SECONDS} s or more,")
print("# measured on a GitHub runner; `default` is the average of all the others.")
print("# Regenerate: scripts/protocol_test_seconds.py report_*.xml > scripts/protocol_test_seconds.txt")
for name in sorted(slow, key=lambda name: (-slow[name], name)):
    print(f"{round(slow[name])} {name}")
print(f"{max(1, round(sum(others) / len(others))) if others else 1} default")
