#!/usr/bin/env python3
"""Test libstatus reuse and invalidation with a small build-command fixture."""
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


class BackendCacheTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        script = Path(__file__).with_name("tkl_backend.sh")
        self.assertTrue(script.exists(), "backend cache helper is missing")
        (self.root / "scripts").mkdir()
        shutil.copy(script, self.root / "scripts")
        (self.root / ".gitignore").write_text("build/\n")
        (self.root / "main.go").write_text("package fixture\n")
        self.run_cmd("git", "init", "-q")
        self.run_cmd("git", "add", ".")
        self.run_cmd("git", "-c", "user.name=Test", "-c", "user.email=test@example.com",
                     "commit", "-qm", "fixture")

    def run_cmd(self, *args, **kwargs):
        return subprocess.run(args, cwd=self.root, text=True, capture_output=True,
                              check=True, **kwargs)

    def build(self, **env):
        return self.run_cmd("bash", "scripts/tkl_backend.sh", "build/libstatus",
                            "sh", "-c", "echo build >> build/count; echo library > build/libstatus",
                            env=dict(os.environ, **env))

    def count(self):
        return len((self.root / "build/count").read_text().splitlines())

    def test_unchanged_build_is_reused(self):
        self.build()
        self.build()
        self.assertEqual(self.count(), 1)

    def test_checksum_tool_fallback(self):
        for tool in ("sha256sum", "shasum", None):
            with self.subTest(tool=tool):
                if tool and not shutil.which(tool):
                    continue
                # Only expose the selected checksum tool, even on a host with both.
                bindir = self.root / "build" / (tool or "no-hash-tool")
                bindir.mkdir(parents=True)
                for name in ("bash", "sh", "dirname", "mkdir", "rmdir", "git",
                             "go", "sed", "xargs", "awk", "cat", "rm") + ((tool,) if tool else ()):
                    (bindir / name).symlink_to(shutil.which(name))
                if tool:
                    self.build(PATH=str(bindir))
                    count = self.count()
                    self.assertIn("Reusing tagged backend", self.build(PATH=str(bindir)).stdout)
                    self.assertEqual(self.count(), count)
                else:
                    result = subprocess.run(
                        [str(bindir / "bash"), "scripts/tkl_backend.sh", "build/libstatus", "true"],
                        cwd=self.root, env=dict(os.environ, PATH=str(bindir)),
                        text=True, capture_output=True)
                    self.assertNotEqual(result.returncode, 0)
                    self.assertIn("Install sha256sum or shasum", result.stderr)

    def test_changed_sources_and_native_inputs_rebuild(self):
        self.build()
        (self.root / "main.go").write_text("package changed\n")
        self.build()
        self.build(CGO_CFLAGS="-DTKL_BUILD_ID=123")
        self.build(CGO_CFLAGS="-DTKL_BUILD_ID=123", SENTRY_CONTEXT_VERSION="new-version")
        self.assertEqual(self.count(), 4)

    def test_missing_or_overwritten_library_rebuilds(self):
        self.build()
        (self.root / "build/libstatus").unlink()
        self.build()
        (self.root / "build/libstatus").write_text("an untagged replacement\n")
        self.build()
        self.assertEqual(self.count(), 3)


if __name__ == "__main__":
    unittest.main()
