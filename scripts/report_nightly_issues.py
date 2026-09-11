#!/usr/bin/env python3
"""Aggregate nightly RPC failures into one open GitHub issue."""

import argparse
from datetime import datetime, timezone
import hashlib
import html
import json
import os
from pathlib import Path
import urllib.error
import urllib.request
import xml.etree.ElementTree as ET


REPOSITORY = "status-im/status-go"
API = f"https://api.github.com/repos/{REPOSITORY}"
TITLE_PREFIX = "[Status functional tests using Logos Delivery nightly] - Failing tests"
ASSIGNEES = ["igor-sirotin"]


def failed_tests(report_dir):
    reports = sorted(Path(report_dir).glob("*.xml"))
    failures = {}
    for report in reports:
        for case in ET.parse(report).iter("testcase"):
            # Ignore rerunFailure and skipped: only final failures are actionable.
            for error in list(case.findall("failure")) + list(case.findall("error")):
                failure = {
                    "test": f"{case.get('classname', '')}::{case.get('name', '')}",
                    "kind": error.tag,
                    "message": error.get("message") or (error.text or "").strip() or "No error message recorded",
                    "details": (error.text or "").strip(),
                }
                failures[tuple(failure.values())] = failure
    return reports, sorted(failures.values(), key=lambda failure: (failure["message"], failure["test"]))


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

    def pages(self, path):
        separator = "&" if "?" in path else "?"
        page = 1
        while True:
            items = self.request("GET", f"{path}{separator}per_page=100&page={page}")
            yield from items
            if len(items) < 100:
                return
            page += 1

    def publish(self, marker, title, body):
        # Use the title prefix so manually created/edited reports can also be reused.
        for issue in self.pages("/issues?state=open&sort=created&direction=asc"):
            if "pull_request" in issue or TITLE_PREFIX not in issue.get("title", ""):
                continue
            if marker in (issue.get("body") or ""):
                return "existing", issue["html_url"]
            for comment in self.pages(f"/issues/{issue['number']}/comments"):
                if marker in (comment.get("body") or ""):
                    return "existing", issue["html_url"]
            self.request(
                "POST",
                f"/issues/{issue['number']}/comments",
                {"body": "Failures have been detected on a newer run as well.\n\n" + body},
            )
            return "commented", issue["html_url"]
        issue = self.request("POST", "/issues", {"title": title, "body": body, "assignees": ASSIGNEES})
        assigned = {user["login"].lower() for user in issue.get("assignees", [])}
        if not {user.lower() for user in ASSIGNEES}.issubset(assigned):
            print(f"WARNING: GitHub did not assign the requested user. Check {issue['html_url']}", flush=True)
            return "created-assignment-incomplete", issue["html_url"]
        return "created", issue["html_url"]


def failure_details(failures):
    groups = {}
    for failure in failures:
        groups.setdefault((failure["kind"], failure["message"]), []).append(failure)
    sections = []
    for (kind, message), cases in groups.items():
        names = "\n".join(f"- <code>{html.escape(case['test'])}</code>" for case in cases)
        trace = html.escape(cases[0]["details"][:2000])
        sections.append(
            f"### {kind.capitalize()} ({len(cases)} affected tests)\n\n"
            f"<pre>{html.escape(message)}</pre>\n\n{names}\n\n"
            f"<details><summary>Example traceback (up to 2,000 characters)</summary>\n\n<pre>{trace}</pre>\n\n</details>"
        )
    return "\n\n".join(sections)


def report(mode, report_dir, output_dir, environ=None, github_factory=GitHub):
    environ = os.environ if environ is None else environ
    output = Path(output_dir)
    output.mkdir(parents=True, exist_ok=True)
    url_path = output / "issue-url.txt"
    url_path.unlink(missing_ok=True)
    summary = {"mode": mode, "repository": REPOSITORY, "action": "pending", "issue_url": None}
    try:
        build_url = environ.get("BUILD_URL", "").rstrip("/") + "/"
        if build_url == "/":
            raise ValueError("BUILD_URL is required so every reported issue links to its Jenkins run")
        reports, failures = failed_tests(report_dir)
        summary["failed_tests"] = failures
        if not reports:
            summary.update(action="no-test-report", message="No JUnit report: inspect setup/build logs. No issue created.")
            return 0
        if not failures:
            summary["action"] = "no-failures"
            return 0
        marker = f"<!-- logos-delivery-nightly:run:{hashlib.sha256(build_url.encode()).hexdigest()} -->"
        title = f"{TITLE_PREFIX} {datetime.now(timezone.utc):%d.%m.%Y}"
        body = (
            f"{marker}\n\nJenkins build: {build_url}\n\n"
            f"Test report: {build_url}testReport/\n\n"
            f"Full failure details: {build_url}artifact/nightly-report/issue-report.json\n\n"
            f"Build result: {environ.get('NIGHTLY_BUILD_RESULT', 'unknown')}\n\n"
            f"status-go commit: `{environ.get('GIT_COMMIT', 'not recorded')}`\n\n"
            f"Delivery image: `{environ.get('WAKU_IMAGE', 'not recorded')}`\n\n" + failure_details(failures)
        )
        # Preserve the complete structured report as an artifact even for unusually large runs.
        if len(body) > 60000:
            body = body[:59000] + f"\n\nReport truncated. All failures: {build_url}artifact/nightly-report/issue-report.json\n"
        (output / "issue-preview.md").write_text(f"# {title}\n\n{body}")
        if mode == "preview":
            summary["action"] = "preview"
            return 0
        token = environ.get("GH_TOKEN", "")
        if not token:
            raise ValueError("No GitHub token bound. Check the existing status-im-auto Jenkins username/token credential.")
        action, url = github_factory(token).publish(marker, title, body)
        summary.update(action=action, issue_url=url)
        url_path.write_text(url + "\n")
        print(f"\nGITHUB ISSUE {action.upper()}: {url}\n", flush=True)
        return 0
    except (ValueError, ET.ParseError, OSError, urllib.error.URLError) as exc:
        if isinstance(exc, urllib.error.HTTPError):
            message = f"GitHub HTTP {exc.code}. Check credential permissions and requested assignees for {REPOSITORY}."
        elif isinstance(exc, urllib.error.URLError):
            message = "Could not reach the GitHub API. Check Jenkins network access."
        else:
            message = str(exc)
        summary.update(action="error", message=message)
        print(f"GitHub issue reporting failed: {message}", flush=True)
        return 1
    finally:
        (output / "issue-report.json").write_text(json.dumps(summary, indent=2) + "\n")
        print(json.dumps({key: value for key, value in summary.items() if key != "failed_tests"}, indent=2), flush=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--mode", choices=("preview", "publish"), default="preview")
    parser.add_argument("--report-dir", default="test/functional/reports")
    parser.add_argument("--output-dir", default="nightly-report")
    args = parser.parse_args()
    return report(args.mode, args.report_dir, args.output_dir)


if __name__ == "__main__":
    raise SystemExit(main())
