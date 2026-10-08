#!/usr/bin/env python3
"""Check the real mobile recipes without generating code or rebuilding SDS."""
from pathlib import Path
import subprocess
import unittest


class MobileBuildTest(unittest.TestCase):
    def test_shared_build_keeps_sds_prerequisite(self):
        repo = Path(__file__).resolve().parent.parent
        result = subprocess.run(
            ["make", "-qp", "statusgo-shared-library", "LIBSDS=/fixture/libsds.a"],
            cwd=repo, text=True, capture_output=True)
        self.assertIn(result.returncode, (0, 1), result.stderr)
        rule = next(line for line in result.stdout.splitlines()
                    if line.startswith("statusgo-shared-library:"))
        self.assertIn("/fixture/libsds.a", rule)

    def test_archive_tests_prepare_token_library_and_keep_storage_flags(self):
        repo = Path(__file__).resolve().parent.parent
        for target in ("test-storage", "test-torrent"):
            with self.subTest(target=target):
                result = subprocess.run(
                    ["make", "-n", target, "-o", "generate", "-o", "build-storage",
                     "LIBSDS=/fixture/libsds.a", "-o", "/fixture/libsds.a"],
                    cwd=repo, text=True, capture_output=True)
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                commands = [line for line in result.stdout.splitlines() if "gotestsum" in line]
                self.assertTrue(commands)
                for command in commands:
                    self.assertIn("bash scripts/tkl_env.sh gotestsum", command)
                    if target == "test-storage":
                        self.assertIn("-lstorage", command)

    def test_ios_deployment_target_defaults_and_overrides(self):
        repo = Path(__file__).resolve().parent.parent
        for sdk, arch, minimum in (("iphoneos", "arm64", "13.0"),
                                   ("iphonesimulator", "arm64", "14.0"),
                                   ("iphonesimulator", "x86_64", "13.0")):
            for override in (None, "17.0"):
                with self.subTest(sdk=sdk, arch=arch, override=override):
                    command = [
                        "make", "-n", "statusgo-ios-library", "-o", "generate",
                        "-o", "statusgo-c-bindings", "-o", "build-libsds-ios",
                        "USE_NIM_TOKEN_LISTS=true", f"ARCH={arch}", f"IPHONE_SDK={sdk}"]
                    if override:
                        command.append(f"IOS_TARGET={override}")
                    result = subprocess.run(command, cwd=repo, text=True, capture_output=True)
                    self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                    target = override or minimum
                    self.assertIn(f'IPHONE_SDK="{sdk}" IOS_TARGET="{target}"', result.stdout)
                    self.assertIn(f'-miphoneos-version-min={target}', result.stdout)

    def test_static_archive_always_bundles_native_library(self):
        repo = Path(__file__).resolve().parent.parent
        for tags in ("gowaku_no_rln", "gowaku_no_rln tkl", "gowaku_no_rln,tkl"):
            with self.subTest(tags=tags):
                result = subprocess.run(
                    ["make", "-n", "statusgo-library", "-o", "generate",
                     "-o", "statusgo-c-bindings", "LIBSDS=/fixture/libsds.a",
                     "-o", "/fixture/libsds.a", f"BUILD_TAGS={tags}"],
                    cwd=repo, text=True, capture_output=True, check=True)
                self.assertIn("scripts/tkl_env.sh", result.stdout)
                self.assertIn("scripts/tkl_bundle_archive.sh", result.stdout)

    def test_mobile_always_prepares_native_library(self):
        repo = Path(__file__).resolve().parent.parent
        for platform in ("android", "ios"):
            for enabled in ("false", "true"):
                with self.subTest(platform=platform, enabled=enabled):
                    result = subprocess.run(
                        ["make", "-n", f"statusgo-{platform}-library",
                         "-o", "generate", "-o", "statusgo-c-bindings",
                         "-o", f"build-libsds-{platform}",
                         f"USE_NIM_TOKEN_LISTS={enabled}", "ARCH=arm64",
                         "ANDROID_NDK_ROOT=/fixture/ndk", "HOST_OS=linux",
                         "IPHONE_SDK=iphonesimulator", "IOS_TARGET=14.0"],
                        cwd=repo, text=True, capture_output=True, check=True)
                    command = result.stdout
                    self.assertIn("bash scripts/tkl_env.sh", command)
                    self.assertEqual("scripts/tkl_bundle_archive.sh" in command,
                                     platform == "ios")
                    if platform == "android" or platform == "ios":
                        if platform == "android":
                            self.assertIn("TKL_HIDE_EXPORTS=1", command)
                            self.assertIn("scripts/check_tkl_exports.sh", command)
                        else:
                            self.assertIn('IPHONE_SDK="iphonesimulator" IOS_TARGET="14.0"', command)


if __name__ == "__main__":
    unittest.main()
