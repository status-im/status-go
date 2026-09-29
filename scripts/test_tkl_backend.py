#!/usr/bin/env python3
"""Test libstatus reuse and invalidation with a small build-command fixture."""
import os
from pathlib import Path
import shutil
import signal
import subprocess
import tempfile
import time
import unittest


class BackendCacheTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.env = dict(os.environ)
        for key in ("TKL_LOCK_TIMEOUT", "TKL_TEST_BUILD_GATE"):
            self.env.pop(key, None)
        self.lock = self.root / "build/libstatus.tkl-lock"
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

    def build_args(self):
        return ["bash", "scripts/tkl_backend.sh", "build/libstatus", "sh", "-c",
                'if [ -n "${TKL_TEST_BUILD_GATE:-}" ]; then '
                'touch "$TKL_TEST_BUILD_GATE.ready"; '
                'while [ ! -f "$TKL_TEST_BUILD_GATE" ]; do sleep 0.05; done; fi; '
                'echo build >> build/count; echo library > build/libstatus']

    def build(self, check=True, **env):
        return subprocess.run(self.build_args(), cwd=self.root, text=True,
                              capture_output=True, check=check, timeout=15,
                              env=dict(self.env, **env))

    def start_build(self, **env):
        process = subprocess.Popen(self.build_args(), cwd=self.root,
                                   env=dict(self.env, **env), text=True,
                                   stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                   start_new_session=True)

        def cleanup():
            try:
                process.communicate(timeout=1)
            except subprocess.TimeoutExpired:
                os.killpg(process.pid, signal.SIGKILL)
                process.communicate()

        self.addCleanup(cleanup)
        return process

    def blocked_build(self):
        gate = self.root / "build/gate"
        process = self.start_build(TKL_TEST_BUILD_GATE=str(gate))
        deadline = time.monotonic() + 10
        while not gate.with_suffix(".ready").exists():
            if process.poll() is not None:
                self.fail(process.communicate()[1])
            self.assertLess(time.monotonic(), deadline, "build never reached the gate")
            time.sleep(0.02)
        return process, gate

    def assert_build_succeeds(self, process):
        _, stderr = process.communicate(timeout=15)
        self.assertEqual(process.returncode, 0, stderr)

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

    def test_recovers_empty_legacy_lock(self):
        self.lock.mkdir(parents=True)
        self.build()
        self.assertEqual(self.count(), 1)
        self.assertFalse(self.lock.exists())

    def test_recovers_killed_owner(self):
        process, _ = self.blocked_build()
        os.killpg(process.pid, signal.SIGKILL)
        process.communicate(timeout=5)
        self.assertTrue(self.lock.exists())
        self.build()
        self.assertEqual(self.count(), 1)
        self.assertFalse(self.lock.exists())

    def test_concurrent_build_waits_and_reuses_library(self):
        first, gate = self.blocked_build()
        second = self.start_build(TKL_TEST_BUILD_GATE=str(gate))
        time.sleep(0.3)
        self.assertIsNone(second.poll(), "concurrent build must wait for the owner")
        gate.touch()
        self.assert_build_succeeds(first)
        self.assert_build_succeeds(second)
        self.assertEqual(self.count(), 1)
        self.assertFalse(self.lock.exists())

    def test_concurrent_waiters_recover_stale_lock(self):
        exited = subprocess.Popen(["true"])
        exited.wait()
        (self.lock / str(exited.pid)).mkdir(parents=True)
        waiters = [self.start_build() for _ in range(6)]
        for waiter in waiters:
            self.assert_build_succeeds(waiter)
        self.assertEqual(self.count(), 1)
        self.assertFalse(self.lock.exists())

    def test_live_owner_times_out_without_removing_lock(self):
        owner = self.lock / str(os.getpid())
        owner.mkdir(parents=True)
        result = self.build(check=False, TKL_LOCK_TIMEOUT="1")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Timed out", result.stderr)
        self.assertTrue(owner.is_dir())
        self.assertFalse((self.root / "build/count").exists())

    def test_failed_build_releases_lock_and_has_no_success_stamp(self):
        self.build()
        result = subprocess.run(
            ["bash", "scripts/tkl_backend.sh", "build/libstatus", "false"],
            cwd=self.root, env=self.env, text=True, capture_output=True, timeout=15)
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(self.lock.exists())
        self.assertFalse((self.root / "build/libstatus.tkl-inputs").exists())
        self.build()
        self.assertEqual(self.count(), 2)


class BackendMakeTargetTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        shutil.copy(Path(__file__).resolve().parents[1] / "Makefile", self.root)
        scripts = self.root / "scripts"
        scripts.mkdir()
        # Observe the real Make recipe without invoking native or Go builds.
        (scripts / "tkl_env.sh").write_text('printf "%s\\n" "$@" > wrapper.args\n')
        (scripts / "check_tkl_exports.sh").write_text("exit 0\n")
        self.libs = self.root / "sds"
        self.libs.mkdir()

    def make(self, **overrides):
        variables = dict(NIM_SDS_LIB_DIR=str(self.libs),
                         NIM_SDS_INC_DIR=str(self.libs), LIB_EXT="so", **overrides)
        return subprocess.run(
            ["make", "--no-print-directory", "statusgo-shared-library-tkl"] +
            [f"{key}={value}" for key, value in variables.items()],
            cwd=self.root, text=True, capture_output=True, timeout=15)

    def test_missing_sds_fails_before_wrapper(self):
        result = self.make()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("libsds not found", result.stderr)
        self.assertIn("make build-libsds", result.stderr)
        self.assertFalse((self.root / "wrapper.args").exists())

    def test_existing_shared_or_static_library_is_forwarded(self):
        for names, selected in ((["libsds.so"], "libsds.so"),
                                (["libsds.a"], "libsds.a"),
                                (["libsds.so", "libsds.a"], "libsds.so")):
            with self.subTest(names=names):
                for old in self.libs.iterdir():
                    old.unlink()
                for name in names:
                    (self.libs / name).touch()
                result = self.make()
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertIn(f"LIBSDS={self.libs / selected}",
                              (self.root / "wrapper.args").read_text().splitlines())

    def test_explicit_library_override_is_preserved(self):
        library = self.root / "custom-libsds.so"
        library.touch()
        (self.libs / "libsds.a").touch()
        result = self.make(LIBSDS=str(library))
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn(f"LIBSDS={library}",
                      (self.root / "wrapper.args").read_text().splitlines())


if __name__ == "__main__":
    unittest.main()
