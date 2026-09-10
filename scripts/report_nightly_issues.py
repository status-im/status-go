#!/usr/bin/env python3
"""Report nightly RPC failures; preview mode performs no GitHub requests."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import urllib.error
import urllib.request
import xml.etree.ElementTree as ET


REPOSITORY = "status-im/status-go"
API = f"https://api.github.com/repos/{REPOSITORY}"


def failed_tests(report_dir):
    reports = sorted(Path(report_dir).glob("*.xml"))
    failures = set()
    for report in reports:
        for case in ET.parse(report).iter("testcase"):
            if case.find("failure") is not None or case.find("error") is not None:
                failures.add(f"{case.get('classname', '')}::{case.get('name', '')}")
    return reports, sorted(failures)


class GitHub:
    def __init__(self, token):
        self.token = token

    def request(self, method, path, payload=None):
        data = None if payload is None else json.dumps(payload).encode()
        request = urllib.request.Request(
            API + path,
            data=data,
            method=method,
            headers={
                "Authorization": f"Bearer {self.token}",
                "Accept": "application/vnd.github+json",
                "Content-Type": "application/json",
                "X-GitHub-Api-Version": "2022-11-28",
            },
        )
        with urllib.request.urlopen(request, timeout=30) as response:
            return json.load(response)

    def publish(self, marker, title, body):
        # Reuse an open failure issue instead of opening one every night.
        page = 1
        while True:
            issues = self.request("GET", f"/issues?state=open&per_page=100&page={page}")
            for issue in issues:
                if "pull_request" not in issue and marker in (issue.get("body") or ""):
                    if issue.get("body") == body:
                        return "existing", issue["html_url"]
                    self.request("POST", f"/issues/{issue['number']}/comments", {"body": body})
                    return "commented", issue["html_url"]
            if len(issues) < 100:
                break
            page += 1
        issue = self.request("POST", "/issues", {"title": title, "body": body})
        return "created", issue["html_url"]


def read_optional(path):
    return path.read_text().strip() if path.exists() else "not recorded"


def report(mode, report_dir, output_dir, environ=None, github_factory=GitHub):
    environ = os.environ if environ is None else environ
    output = Path(output_dir)
    output.mkdir(parents=True, exist_ok=True)
    url_path = output / "issue-url.txt"
    url_path.unlink(missing_ok=True)
    summary = {"mode": mode, "repository": REPOSITORY, "action": "pending", "issue_url": None}
    try:
        build_url = environ.get("BUILD_URL", "")
        if not build_url:
            raise ValueError("BUILD_URL is required so every reported issue links to its Jenkins run")
        marker = "<!-- logos-delivery-nightly:failures -->"
        title = "Logos Delivery nightly: functional test failures"
        if mode == "auth-test":
            marker = f"<!-- logos-delivery-nightly:auth-test:{hashlib.sha256(build_url.encode()).hexdigest()} -->"
            title = "[TEST — Jenkins GitHub authentication] Logos Delivery nightly"
            details = "This is a deliberate authentication test, not a product bug. Please close it after verifying this integration."
        else:
            reports, failures = failed_tests(report_dir)
            summary["failed_tests"] = failures
            if not reports:
                summary["action"] = "no-test-report"
                summary["message"] = "No JUnit report: inspect setup/build logs. No test-failure issue created."
                return 0
            if not failures:
                summary["action"] = "no-failures"
                return 0
            details = "Failed tests (final JUnit results):\n\n" + "\n".join(f"- `{name[:500]}`" for name in failures[:50])
            if len(failures) > 50:
                details += f"\n\nPlus {len(failures) - 50} more; see the Jenkins test report."
        body = (
            f"{marker}\n\n{details}\n\n"
            f"Jenkins build: {build_url}\n\n"
            f"Test report: {build_url.rstrip('/')}/testReport/\n\n"
            f"Artifacts: {build_url.rstrip('/')}/artifact/\n\n"
            f"Build result before issue reporting: {environ.get('NIGHTLY_BUILD_RESULT', 'unknown')}\n\n"
            f"status-go commit: `{read_optional(output / 'status-go-sha.txt')}`\n\n"
            f"Delivery image: `{read_optional(output / 'delivery-image.txt')}`\n"
        )
        (output / "issue-preview.md").write_text(f"# {title}\n\n{body}")
        if mode == "preview":
            summary["action"] = "preview"
            summary["message"] = "No GitHub requests made. Review issue-preview.md before enabling publication."
            return 0
        token = environ.get("GH_TOKEN", "")
        if not token:
            raise ValueError("No GitHub token bound. Check binding of the existing status-im-auto Jenkins username/token credential.")
        action, url = github_factory(token).publish(marker, title, body)
        summary.update(action=action, issue_url=url)
        url_path.write_text(url + "\n")
        print(f"\nGITHUB ISSUE {action.upper()}: {url}\n", flush=True)
        return 0
    except (ValueError, ET.ParseError, OSError, urllib.error.URLError) as exc:
        # Do not include request headers or credentials in artifacts or exception output.
        if isinstance(exc, urllib.error.HTTPError):
            message = f"GitHub HTTP {exc.code}. Verify credential access and Issues write permission for {REPOSITORY}."
        elif isinstance(exc, urllib.error.URLError):
            message = "Could not reach the GitHub API. Check Jenkins network access."
        else:
            message = str(exc)
        summary.update(action="error", message=message)
        print(f"GitHub issue reporting failed: {message}", flush=True)
        return 1
    finally:
        (output / "issue-report.json").write_text(json.dumps(summary, indent=2) + "\n")
        print(json.dumps(summary, indent=2), flush=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--mode", choices=("preview", "publish", "auth-test"), default="preview")
    parser.add_argument("--report-dir", default="test/functional/reports")
    parser.add_argument("--output-dir", default="nightly-report")
    args = parser.parse_args()
    return report(args.mode, args.report_dir, args.output_dir)


if __name__ == "__main__":
    raise SystemExit(main())
