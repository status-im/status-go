#!/usr/bin/env python3
"""Exercise checkout/cache/cleanup without downloading or compiling Nim."""
import os
import hashlib
from pathlib import Path
import shutil
import subprocess
import tempfile
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
        source = self.root / "upstream"
        (source / "abi").mkdir(parents=True)
        (source / "abi/tkl.h").write_text("fixture header\n")
        (source / "Makefile").write_text(
            "isolate:\n\tmkdir -p $(OUT)\n"
            "\tprintf archive > $(OUT)/libtkl_isolated.a\n"
            "\techo build >> $(CURDIR)/build-count\n")
        (source / "scripts").mkdir()
        (source / "scripts/build_mobile.sh").write_text(
            '#!/bin/bash\nset -eu\ncd "$(dirname "$0")/.."\n'
            'test "$TKL_BUILD_TESTS" = 0\nmkdir -p "$OUT/link"\n'
            'printf "%s" "$1" > "$OUT/link/libtkl.a"\n'
            'echo "$1" >> build-count\n')
        self.run_cmd("git", "init", "-q", str(source))
        self.run_cmd("git", "-C", str(source), "add", ".")
        self.run_cmd("git", "-C", str(source), "-c", "user.name=Test", "-c",
                     "user.email=test@example.com", "commit", "-qm", "fixture")
        self.pin = self.run_cmd("git", "-C", str(source), "rev-parse", "HEAD").stdout.strip()
        (self.repo / "scripts/tkl.version").write_text(self.pin + "\n")
        (self.repo / "go.mod").write_text(
            "module fixture\nrequire github.com/status-im/nim-token-lists/go/tkl "
            f"v0.0.0-20261005000000-{self.pin[:12]}\n")
        self.env = dict(os.environ)
        self.env.update(GIT_CONFIG_COUNT="1", GIT_CONFIG_KEY_0=
                        f"url.{source}.insteadOf", GIT_CONFIG_VALUE_0=
                        "https://github.com/status-im/nim-token-lists.git", NIM="true")
        for key in ("NIM_TKL_INC_DIR", "NIM_TKL_LIB_DIR", "GOOS", "GOARCH", "OUT"):
            self.env.pop(key, None)
        self.source = self.repo / "build/deps/nim-token-lists"
        target = self.run_cmd("go", "env", "GOHOSTOS", "GOHOSTARCH").stdout.split()
        self.out = self.source / "build" / "-".join(target)

    def run_cmd(self, *args, **kwargs):
        return subprocess.run(args, text=True, stdout=subprocess.PIPE,
                              stderr=subprocess.PIPE, check=True, **kwargs)

    def helper(self, action="prepare", check=True, **env):
        return subprocess.run(["bash", str(self.repo / "scripts/tkl_native.sh"), action],
                              env=dict(self.env, **env), text=True, capture_output=True,
                              check=check)

    def test_reuses_checkout_and_archive_then_clean_recreates(self):
        self.helper()
        self.assertEqual((self.source / "build-count").read_text().splitlines(), ["build"])
        # Make any attempt to contact the remote fail on the second invocation.
        self.run_cmd("git", "-C", str(self.source), "remote", "set-url", "origin", "/missing")
        self.helper()
        self.assertEqual((self.source / "build-count").read_text().splitlines(), ["build"])
        self.helper("clean")
        self.assertFalse(self.source.exists())
        self.helper()
        self.assertTrue((self.out / "link/libtkl.a").is_file())

    def test_changed_flags_rebuild_and_missing_archive_rebuild(self):
        self.helper()
        self.helper(EXTRA_NIMFLAGS="-d:fixture")
        (self.out / "link/libtkl.a").unlink()
        self.helper(EXTRA_NIMFLAGS="-d:fixture")
        self.assertEqual(len((self.source / "build-count").read_text().splitlines()), 3)

    def test_wrong_revision_does_not_reset_checkout(self):
        self.helper()
        self.run_cmd("git", "-C", str(self.source), "-c", "user.name=Test", "-c",
                     "user.email=test@example.com", "commit", "--allow-empty", "-qm", "other")
        head = self.run_cmd("git", "-C", str(self.source), "rev-parse", "HEAD").stdout
        result = self.helper(check=False)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("make clean", result.stderr)
        self.assertEqual(self.run_cmd("git", "-C", str(self.source), "rev-parse", "HEAD").stdout, head)

    def test_prebuilt_mode_does_not_create_checkout(self):
        lib = self.root / "prebuilt"
        lib.mkdir()
        (lib / "libtkl.a").write_text("archive")
        (lib / "tkl.h").write_text("header")
        self.helper(NIM_TKL_LIB_DIR=str(lib), NIM_TKL_INC_DIR=str(lib))
        self.assertFalse(self.source.exists())
        self.helper("clean", NIM_TKL_LIB_DIR=str(lib), NIM_TKL_INC_DIR=str(lib))
        self.assertTrue((lib / "libtkl.a").exists())

    def test_windows_prebuilt_audit_fails_closed(self):
        lib = self.root / "prebuilt"
        lib.mkdir()
        (lib / "libtkl.a").write_text("archive")
        (lib / "tkl.h").write_text("header")
        inspector = self.root / "objdump"
        cases = (
            ("exit 1", False),
            ("printf '.drectve\\n'; awk 'BEGIN { for (i=0; i<10000; i++) print \"other sections\" }'", False),
            ("printf '.text\\n'", True),
        )
        for body, accepted in cases:
            with self.subTest(body=body):
                inspector.write_text("#!/bin/sh\n" + body + "\n")
                inspector.chmod(0o755)
                result = self.helper(check=False, NIM_TKL_LIB_DIR=str(lib),
                                     NIM_TKL_INC_DIR=str(lib), TKL_HIDE_EXPORTS="1",
                                     GOOS="windows", OBJDUMP=str(inspector))
                self.assertEqual(result.returncode == 0, accepted, result.stderr)

    def test_clean_cannot_remove_an_active_build(self):
        self.helper()
        (self.repo / "build/deps/.nim-token-lists-lock").mkdir()
        result = self.helper("clean", check=False)
        self.assertNotEqual(result.returncode, 0)
        self.assertTrue(self.source.exists())

    def test_corrupt_archive_is_rebuilt(self):
        self.helper()
        archive = self.out / "link/libtkl.a"
        archive.write_bytes(b"")
        self.helper()
        self.assertEqual(archive.read_text(), "archive")
        self.assertEqual(len((self.source / "build-count").read_text().splitlines()), 2)

    def test_android_archives_are_cached_separately_and_api_invalidates(self):
        host = "darwin" if os.uname().sysname == "Darwin" else "linux"
        ndk = self.root / "ndk"
        tools = ndk / f"toolchains/llvm/prebuilt/{host}-x86_64/bin"
        tools.mkdir(parents=True)
        for name in ("aarch64-linux-android28-clang", "x86_64-linux-android28-clang",
                     "aarch64-linux-android29-clang"):
            compiler = tools / name
            compiler.write_text('#!/bin/sh\necho "fixture clang"\n')
            compiler.chmod(0o755)
        env = dict(GOOS="android", GOARCH="arm64", ANDROID_NDK_ROOT=str(ndk))
        self.helper(**env)
        self.helper(**dict(env, GOARCH="amd64"))
        self.helper(**env)
        self.assertEqual((self.source / "build-count").read_text().splitlines(),
                         ["android-arm64", "android-x86_64"])
        self.helper(**dict(env, ANDROID_API="29"))
        self.assertEqual(len((self.source / "build-count").read_text().splitlines()), 3)
        for target in ("android-arm64", "android-x86_64"):
            self.assertEqual((self.source / f"build/{target}/link/libtkl.a").read_text(), target)
        self.helper("clean")
        self.assertFalse(self.source.exists())

    def test_unsupported_architecture_fails_before_checkout(self):
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
        self.assertEqual((self.source / "build-count").read_text().splitlines(),
                         ["ios-arm64", "ios-simulator-arm64"])
        self.helper(**dict(env, IPHONE_SDK="iphoneos", IOS_TARGET="15.0"))
        self.assertEqual(len((self.source / "build-count").read_text().splitlines()), 3)

    def test_prebuilt_hash_tools(self):
        lib = self.root / "prebuilt"
        lib.mkdir()
        archive = lib / "libtkl.a"
        header = lib / "tkl.h"
        archive.write_text("archive")
        header.write_text("header")
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
