#!/usr/bin/env python
"""Prints how long each top-level test of internal/protocol took, from the JUnit
reports of a unit test run. run_unit_tests.sh reads the result to give every
shard a similar share of the work.

    scripts/protocol_test_seconds.py report_*.xml > scripts/protocol_test_seconds.txt
"""
import sys
import xml.etree.ElementTree as ET

PACKAGE = "github.com/status-im/status-go/internal/protocol"

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

print("# Seconds per top-level test of internal/protocol, measured on a GitHub runner.")
print("# Regenerate: scripts/protocol_test_seconds.py report_*.xml > scripts/protocol_test_seconds.txt")
for name in sorted(seconds):
    print(f"{max(1, round(seconds[name]))} {name}")
