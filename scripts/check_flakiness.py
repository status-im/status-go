import argparse
import os
import subprocess
import sys
import concurrent.futures
import time

def run_single_test(package_name, test_name, suite_test_name, log_dir, timeout, iteration):
    cmd = ["go", "test", package_name, "-tags=gowaku_skip_migrations,gowaku_no_rln", "-count=1", f"-timeout={timeout}s"]
    if test_name:
        cmd.extend(["-run", f"^{test_name}$"])
    if suite_test_name:
        cmd.extend(["-testify.m", f"^{suite_test_name}$"])

    result = subprocess.run(cmd, capture_output=True, text=True)
    output = result.stdout + result.stderr

    if log_dir:
        os.makedirs(log_dir, exist_ok=True)
        log_file_path = os.path.join(log_dir, f"test_{test_name if test_name else 'package'}_{suite_test_name if suite_test_name else 'all'}_{iteration + 1}.log")
        with open(log_file_path, "w") as log_file:
            log_file.write(output)

    return result.returncode == 0

def run_tests(package_name, test_name, suite_test_name, run_count, log_dir, timeout, max_jobs):
    success_count = 0

    with concurrent.futures.ThreadPoolExecutor(max_workers=max_jobs) as executor:
        futures = [executor.submit(run_single_test, package_name, test_name, suite_test_name, log_dir, timeout, i) for i in range(run_count)]
        for future in concurrent.futures.as_completed(futures):
            if future.result():
                success_count += 1

    return success_count

def main():
    parser = argparse.ArgumentParser(description="Test flakiness checker")
    parser.add_argument("package", help="Name of the package containing the test")
    parser.add_argument("-t", "--test", help="Name of the test or suite to run (optional)")
    parser.add_argument("-s", "--suitetest", help="Name of the test within the suite to run with testify (optional)")
    parser.add_argument("-c", "--count", type=int, default=20, help="Number of times to run the test (default: 20)")
    parser.add_argument("-l", "--logdir", help="Directory to save test logs (optional)")
    parser.add_argument("--timeout", type=int, default=60, help="Timeout for each test run in seconds (default: 60)")
    parser.add_argument("--threshold", type=float, default=0.0, help="Threshold for flakiness in percentage (default: 0.0)")
    parser.add_argument("-j", "--jobs", type=int, default=8, help="Maximum number of parallel jobs (default: 8)")


    start_time = time.time()

    args = parser.parse_args()
    success_count = run_tests(args.package, args.test, args.suitetest, args.count, args.logdir, args.timeout, args.jobs)
    success_ratio = (success_count / args.count) * 100

    end_time = time.time()
    time_taken = end_time - start_time

    print(f"Test {args.test if args.test else 'package'} {args.suitetest if args.suitetest else 'all'} succeeded {success_count}/{args.count} times ({success_ratio:.2f}%)")
    print(f"Total time taken: {time_taken:.2f} seconds")

    flakiness_ratio = 100 - success_ratio
    if flakiness_ratio <= args.threshold:
        sys.exit(0)
    else:
        sys.exit(1)

if __name__ == "__main__":
    main()

