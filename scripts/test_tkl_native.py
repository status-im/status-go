#!/usr/bin/env python3
"""Exercise cache and cleanup without resolving or compiling Nim."""
import os
import hashlib
from pathlib import Path
import shutil
import signal
import subprocess
import tempfile
import time
import unittest


class NativeDependencyTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.repo = self.root / "status-go"
        (self.repo / "scripts").mkdir(parents=True)
        helper = Path(__file__).with_name("tkl_native.sh")
        self.assertTrue(helper.exists(), "native dependency lifecycle helper is missing")
        shutil.copy(helper, self.repo / "scripts")
        shutil.copy(helper.with_name("tkl_env.sh"), self.repo / "scripts")
        (self.repo / "nim.mk").write_text("")
        (self.repo / "Makefile").write_text(
            "nimble.paths:\n\tprintf -- '--noNimblePath\\n' > $@\n"
            "nim-libtkl:\n\tTKL_NIM_FLAGS='$(TKL_NIM_FLAGS)' TKL_PLATFORM='$(TKL_PLATFORM)' ARCH='$(ARCH)' "
            "bash scripts/fake_task.sh '$(NIM_LIB_DIR)'\n")
        # Names the output after the target it is asked for, as the real build would.
        (self.repo / "scripts/fake_task.sh").write_text(
            '#!/bin/bash\nset -eu\nout="$1"\nmkdir -p "$out/link"\n'
            'case "$TKL_PLATFORM" in\n'
            '  ANDROID) name="android-$ARCH" ;;\n'
            '  IOS)\n'
            '    name="ios-$ARCH"\n'
            '    [ "${IPHONE_SDK:-iphoneos}" = iphoneos ] || name="ios-simulator-$ARCH" ;;\n'
            '  *)\n'
            '    name=build\n'
            '    if [ -n "${TKL_TEST_BUILD_GATE:-}" ]; then\n'
            '      touch "$TKL_TEST_BUILD_GATE.ready"\n'
            '      while [ ! -f "$TKL_TEST_BUILD_GATE" ]; do sleep 0.05; done\n'
            '    fi\n'
            '    printf "%s\\n" "$TKL_NIM_FLAGS" > "$out/compiler-flags" ;;\n'
            'esac\n'
            'printf "%s" "$name" > "$out/link/libtkl.a"\n'
            'printf "header\\n" > "$out/link/tkl.h"\n'
            'echo "$name" >> build-count\n')
        self.pin = "0123456789abcdef0123456789abcdef01234567"
        (self.repo / "status_go.nimble").write_text(
            f'requires "https://github.com/fixture/nim-token-lists#{self.pin}"\n')
        (self.repo / "go.mod").write_text(
            "module fixture\nrequire github.com/status-im/nim-token-lists/go/tkl "
            f"v0.0.0-20261005000000-{self.pin[:12]}\n")
        self.env = dict(os.environ)
        for key in ("NIM_TKL_INC_DIR", "NIM_TKL_LIB_DIR", "GOOS", "GOARCH", "OUT",
                    "CGO_CFLAGS", "CGO_LDFLAGS", "TKL_BASE_CGO_CFLAGS",
                    "TKL_BASE_CGO_LDFLAGS", "TKL_TEST_BUILD_GATE"):
            self.env.pop(key, None)
        self.source = self.repo / "build/tkl"
        self.count = self.repo / "build-count"
        target = self.run_cmd("go", "env", "GOHOSTOS", "GOHOSTARCH").stdout.split()
        self.out = self.source / "-".join(target)

    def run_cmd(self, *args, **kwargs):
        return subprocess.run(args, text=True, stdout=subprocess.PIPE,
                              stderr=subprocess.PIPE, check=True, **kwargs)

    def helper(self, action="prepare", check=True, **env):
        return subprocess.run(["bash", str(self.repo / "scripts/tkl_native.sh"), action],
                              env=dict(self.env, **env), text=True, capture_output=True,
                              check=check, timeout=15)

    def link(self, **env):
        return Path(self.helper(**env).stdout.strip())

    def start_helper(self, action="prepare", **env):
        process = subprocess.Popen(
            ["bash", str(self.repo / "scripts/tkl_native.sh"), action],
            env=dict(self.env, **env), text=True, stdout=subprocess.PIPE,
            stderr=subprocess.PIPE, start_new_session=True)

        def cleanup():
            try:
                process.communicate(timeout=1)
            except subprocess.TimeoutExpired:
                os.killpg(process.pid, signal.SIGKILL)
                process.communicate()

        self.addCleanup(cleanup)
        return process

    def blocked_build(self):
        gate = self.root / "build-gate"
        process = self.start_helper(TKL_TEST_BUILD_GATE=str(gate))
        deadline = time.monotonic() + 10
        while not gate.with_suffix(".ready").exists():
            if process.poll() is not None:
                self.fail(process.communicate()[1])
            self.assertLess(time.monotonic(), deadline, "build never reached the gate")
            time.sleep(0.02)
        return process, gate

    def assert_helper_succeeds(self, process):
        _, stderr = process.communicate(timeout=15)
        self.assertEqual(process.returncode, 0, stderr)

    def test_reuses_archive_then_clean_rebuilds(self):
        self.helper()
        self.assertEqual(self.count.read_text().splitlines(), ["build"])
        self.helper()
        self.assertEqual(self.count.read_text().splitlines(), ["build"])
        self.helper("clean")
        self.assertFalse(self.source.exists())
        self.assertTrue((self.link() / "libtkl.a").is_file())

    def test_missing_archive_rebuilds(self):
        self.helper()
        (self.link() / "libtkl.a").unlink()
        self.helper()
        self.assertEqual(len(self.count.read_text().splitlines()), 2)

    def test_changed_nim_params_rebuild(self):
        self.helper()
        self.helper(NIM_PARAMS="--mm:orc -d:release")
        self.assertEqual(len(self.count.read_text().splitlines()), 2)

    def test_prebuilt_mode_does_not_build(self):
        lib = self.root / "prebuilt"
        lib.mkdir()
        (lib / "libtkl.a").write_text("archive")
        (lib / "tkl.h").write_text("header")
        host = "/".join(self.run_cmd("go", "env", "GOHOSTOS", "GOHOSTARCH").stdout.split())
        (lib / "libtkl.target").write_text(host + "\n")
        self.helper(NIM_TKL_LIB_DIR=str(lib), NIM_TKL_INC_DIR=str(lib))
        self.assertFalse(self.source.exists())
        self.helper("clean", NIM_TKL_LIB_DIR=str(lib), NIM_TKL_INC_DIR=str(lib))
        self.assertTrue((lib / "libtkl.a").exists())

    def test_prebuilt_target_metadata_is_checked(self):
        lib = self.root / "prebuilt"
        lib.mkdir()
        (lib / "libtkl.a").write_text("archive")
        (lib / "tkl.h").write_text("header")
        cases = (("android/arm64", "android", "arm64", "", True),
                 ("linux/arm64", "android", "arm64", "", False),
                 ("android/amd64", "android", "arm64", "", False),
                 ("ios/arm64/iphoneos", "ios", "arm64", "iphoneos", True),
                 ("ios/arm64/iphoneos", "ios", "arm64", "iphonesimulator", False))
        for built, goos, goarch, sdk, accepted in cases:
            with self.subTest(built=built, goos=goos, goarch=goarch, sdk=sdk):
                (lib / "libtkl.target").write_text(built + "\n")
                result = self.helper(check=False, GOOS=goos, GOARCH=goarch, IPHONE_SDK=sdk,
                                     NIM_TKL_LIB_DIR=str(lib), NIM_TKL_INC_DIR=str(lib))
                self.assertEqual(result.returncode == 0, accepted, result.stderr)
                if not accepted:
                    self.assertIn("target mismatch", result.stderr)
                self.assertFalse(self.source.exists())

    def test_concurrent_builds_publish_one_archive(self):
        first, gate = self.blocked_build()
        second = self.start_helper()
        gate.touch()
        outputs = [p.communicate(timeout=15) for p in (first, second)]
        for process, (_, stderr) in zip((first, second), outputs):
            self.assertEqual(process.returncode, 0, stderr)
        self.assertEqual(outputs[0][0], outputs[1][0])
        self.assertTrue((Path(outputs[0][0].strip()) / "libtkl.a").is_file())
        self.assertEqual([p.name for p in self.out.iterdir()],
                         [Path(outputs[0][0].strip()).parent.name])

    def test_sourced_and_nested_environments_do_not_duplicate_flags(self):
        for cflags, ldflags in (("", ""), ("-DUSER_FLAG=1", "-lm")):
            with self.subTest(cflags=cflags):
                result = self.run_cmd(
                    "bash", "-c", '''
source "$1"
printf '%s\\n' "$CGO_CFLAGS" "$CGO_LDFLAGS"
source "$1"
printf '%s\\n' "$CGO_CFLAGS" "$CGO_LDFLAGS"
bash "$1" bash -c 'printf "%s\\n" "$CGO_CFLAGS" "$CGO_LDFLAGS"'
''', "bash", str(self.repo / "scripts/tkl_env.sh"),
                    env=dict(self.env, CGO_CFLAGS=cflags, CGO_LDFLAGS=ldflags))
                lines = result.stdout.splitlines()
                self.assertEqual(len(lines), 6)
                self.assertEqual(lines[:2], lines[2:4])
                self.assertEqual(lines[:2], lines[4:])
                self.assertTrue(lines[0].startswith(cflags + ' "-I'))
                self.assertTrue(lines[1].startswith(ldflags + ' "-L'))

    def test_managed_reentry_uses_only_original_compiler_flags(self):
        for cflags in ("", "-DUSER_FLAG=1"):
            with self.subTest(cflags=cflags):
                self.helper("clean")
                self.count.unlink(missing_ok=True)
                self.run_cmd(
                    "bash", "-c", '''
source "$1"
archive="$NIM_TKL_LIB_DIR/libtkl.a"
unset NIM_TKL_LIB_DIR NIM_TKL_INC_DIR
bash scripts/tkl_native.sh
rm "$archive"
bash scripts/tkl_native.sh
''', "bash", str(self.repo / "scripts/tkl_env.sh"),
                    env=dict(self.env, CGO_CFLAGS=cflags), cwd=self.repo)
                self.assertEqual(len(self.count.read_text().splitlines()), 2)
                flags = (next(self.out.iterdir()) / "compiler-flags").read_text()
                self.assertNotIn("TKL_BUILD_ID", flags)
                self.assertNotIn("--passC:-I", flags)
                if cflags:
                    self.assertIn("--passC:-DUSER_FLAG=1", flags)

    def test_changed_caller_compiler_flags_rebuild(self):
        self.helper(CGO_CFLAGS="-DUSER_FLAG=1")
        self.helper(CGO_CFLAGS="-DUSER_FLAG=2")
        self.assertEqual(len(self.count.read_text().splitlines()), 2)
        self.assertIn("--passC:-DUSER_FLAG=2",
                      (self.link(CGO_CFLAGS="-DUSER_FLAG=2").parent / "compiler-flags").read_text())

    def test_corrupt_archive_is_rebuilt(self):
        archive = self.link() / "libtkl.a"
        archive.write_bytes(b"")
        self.helper()
        self.assertEqual(archive.read_text(), "build")
        self.assertEqual(len(self.count.read_text().splitlines()), 2)

    def test_android_archives_are_cached_separately_and_api_invalidates(self):
        ndk = self.root / "ndk"
        ndk.mkdir()
        env = dict(GOOS="android", GOARCH="arm64", ANDROID_NDK_ROOT=str(ndk))
        self.helper(**env)
        self.helper(**dict(env, GOARCH="amd64"))
        self.helper(**env)
        self.assertEqual(self.count.read_text().splitlines(),
                         ["android-arm64", "android-x86_64"])
        self.helper(**dict(env, ANDROID_API="29"))
        self.assertEqual(len(self.count.read_text().splitlines()), 3)
        for target in ("android-arm64", "android-x86_64"):
            self.assertEqual((next((self.source / target).iterdir()) / "link/libtkl.a").read_text(), target)
        # The environment wrapper's exported paths must remain usable on a
        # second invocation, including for a managed cross-target archive.
        result = subprocess.run(
            ["bash", "-c", 'source "$1"; source "$1"', "fixture",
             str(self.repo / "scripts/tkl_env.sh")],
            env=dict(self.env, **env), text=True, capture_output=True, timeout=15)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual((next((self.source / "android-arm64").iterdir()) / "link/libtkl.target").read_text(),
                         "android/arm64\n")
        self.helper("clean")
        self.assertFalse(self.source.exists())

    def test_unsupported_architecture_fails_before_building(self):
        result = self.helper(check=False, GOOS="android", GOARCH="386")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Unsupported", result.stderr)
        self.assertFalse(self.source.exists())

    @unittest.skipUnless(shutil.which("xcrun"), "requires Xcode SDK discovery")
    def test_ios_device_and_simulator_keep_distinct_archives(self):
        env = dict(GOOS="ios", GOARCH="arm64", IOS_TARGET="14.0")
        self.helper(**dict(env, IPHONE_SDK="iphoneos"))
        self.helper(**dict(env, IPHONE_SDK="iphonesimulator"))
        self.helper(**dict(env, IPHONE_SDK="iphoneos"))
        self.assertEqual(self.count.read_text().splitlines(),
                         ["ios-arm64", "ios-simulator-arm64"])
        self.helper(**dict(env, IPHONE_SDK="iphoneos", IOS_TARGET="15.0"))
        self.assertEqual(len(self.count.read_text().splitlines()), 3)

    def test_prebuilt_hash_tools(self):
        lib = self.root / "prebuilt"
        lib.mkdir()
        archive = lib / "libtkl.a"
        header = lib / "tkl.h"
        archive.write_text("archive")
        header.write_text("header")
        host = "/".join(self.run_cmd("go", "env", "GOHOSTOS", "GOHOSTARCH").stdout.split())
        (lib / "libtkl.target").write_text(host + "\n")
        checksums = "".join(
            f"{hashlib.sha256(path.read_bytes()).hexdigest()}  {path}\n"
            for path in (archive, header))
        expected = hashlib.sha256(checksums.encode()).hexdigest()
        for tool in ("sha256sum", "shasum", None):
            with self.subTest(tool=tool):
                if tool and not shutil.which(tool):
                    continue  # Each host exercises its available checksum tools.
                bindir = self.root / (tool or "no-hash-tool")
                bindir.mkdir()
                for name in ("bash", "dirname", "awk", "go") + ((tool,) if tool else ()):
                    (bindir / name).symlink_to(shutil.which(name))
                result = subprocess.run(
                    [str(bindir / "bash"), str(self.repo / "scripts/tkl_env.sh"),
                     "bash", "-c", 'printf "%s" "$CGO_CFLAGS"'],
                    env=dict(self.env, PATH=str(bindir), NIM_TKL_LIB_DIR=str(lib),
                             NIM_TKL_INC_DIR=str(lib)), text=True, capture_output=True)
                if tool:
                    self.assertEqual(result.returncode, 0, result.stderr)
                    self.assertIn(f"-DTKL_BUILD_ID=0x{expected}", result.stdout)
                    self.assertFalse(self.source.exists())
                else:
                    self.assertNotEqual(result.returncode, 0)
                    self.assertIn("Install sha256sum or shasum", result.stderr)


if __name__ == "__main__":
    unittest.main()
