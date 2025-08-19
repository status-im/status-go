## Overview

Functional tests for status-go

## Table of Contents

- [Overview](#overview)
- [How to Install](#how-to-install)
- [How to Run](#how-to-run)
  - [Running Tests](#running-tests)
- [Implementation details](#implementation-details)
- [Reliability tests](#reliability-tests)

## Prerequisites

1. Install [Docker](https://docs.docker.com/engine/install/) and [Docker Compose](https://docs.docker.com/compose/install/)
2. Install [Python 3](https://www.python.org/downloads/) (tested with 3.10 and 3.12 and it works with both)
   - **Note:** For Python 3.10 compatibility, the codebase uses `Iterator[str]` instead of `Generator[str]` for type annotations to avoid typing issues with older Python versions.
3. Set up a virtual environment (required for linting):
   - In `./tests-functional`, run:
     ```bash
     python3 -m venv .venv
     source .venv/bin/activate
     pip install -r requirements.txt
     pre-commit install
     ```
   - **Important**: The virtual environment must be created in the `./tests-functional` directory for pre-commit linting and type-checking tools like Pyright to work correctly.
   - **Optional (for test development)**: Use Python virtual environment for better dependency management. You can follow the guide [here](https://akrabat.com/creating-virtual-environments-with-pyenv/)

## How to Run

1. Functional tests will spawn `status-go` containers as needed, using the docker image name provided in `--docker-image`.
    Run this command to build a docker image for local runs in root dir `status-go`:
    ```sh
    # Note we tag the image with the commit hash. Functional tests will refer to the same when `--docker-image` is empty.
    docker build --tag "statusgo-$(git rev-parse --short HEAD)" .
    ```
2. Functional tests rely on local instances of Waku and Anvil.
   Run this command to start all environment dependencies:
    ```sh
    docker compose -f tests-functional/docker-compose.anvil.yml -f tests-functional/docker-compose.waku.yml up --build --remove-orphans
    ```
   This will:
    * Start a Waku fleet of 1 boot node + 1 store node
    * Start Anvil with ChainID 31337 exposed on `0.0.0.0:8545`
    * Deploy required Status contracts to Anvil

3. In `./tests-functional/tests` directory run (with the `venv` running):
   ```sh
    pytest -m rpc
   ```

### Running against a `status-backend` binary

By default, tests will spawn containers with `status-backend` when and as needed.
This might be inconvenient when:
- You make changes to the `status-go` code, so you have to rebuild it frequently
- You need to run `status-go` code with a debugger

For such cases, you can utilize `--status_backend_url` argument:
```shell
pytest --status_backend_url=http://<host>:<port> --status_backend_url=http://<host>:<port>
```

When provided, tests will connect to the given URL instead of spawning Docker containers.

NOTE:
1. You must provide as many URLs as you need in your test
2. URLs are not reused
3. URLs are not reusedURLs are selected in the provided order
4. Most tests do not call `Logout` in the end of the test, so make sure to restart `status-backend` in between test runs.
   You can also use `--logout` flag to automatically `Logout` before each `InitializeApplication` call, but it is
   recommended to restart `status-backend` for a full clean test.

### Options

#### Running with a custom Waku fleet

`status-backend` can use a hard-coded list of supported Waku fleets or fleets specified in a config file. Functional tests provide 2 options to specify which Waku fleet to use.

- Use `--waku-fleets-config` to override the hard-coded list of supported fleets. \
    The value will be passed to `InitializeApplication.wakuFleetsConfigFilePath` parameter. By default, it's set to JSON config for a local waku fleet. To run with a fleet from the hard-coded list use `--waku-fleets-config=""` and corresponding `--waku-fleet` value, e.g. `--waku-fleet=status.prod`
- Use `--waku-fleet` to select a Waku fleet to be used by `status-go`. \
    Default is `status-go.test` that will use local waku nodes for functional tests run.
    The value will be passed to all of these parameters:
    - `CreateAccount.wakuV2Fleet`
    - `RestoreAccount.wakuV2Fleet`

Please refer to the description of these parameters in `status-go` for details.

### Prerequisites for Mac OSx users

If you see errors at attempt to run tests, try to run in terminal:
```shell
sock.connect(self.unix_socket)
```
If you see
```
PermissionError: [Errno 13] Permission denied
```

Please follow this fix: https://github.com/docker/compose/issues/10299#issuecomment-1438247730

If you're on MacOS and `/var/run/docker.sock` doesn't exist, you need to create a symlink to the docker socket:
```shell
sudo ln -s $HOME/.docker/run/docker.sock /var/run/docker.sock
```

## Implementation details

- Functional tests are implemented in `./tests-functional/tests` based on [pytest](https://docs.pytest.org/en/8.2.x/)
- Backend fixtures and test infrastructure are defined in `./tests-functional/tests/conftest.py`
- Every test has the following verifications:
    - `verify_is_valid_json_rpc_response()` checks for status code 200, non-empty response, JSON-RPC structure, presence of the `result` / `error` field, and expected ID.

### Backend Setup and Cleanup with Pytest Fixtures

Functional tests in this suite rely on robust backend setup and teardown, managed via reusable pytest fixtures defined in `tests/conftest.py`. These fixtures ensure that backend containers are started, reused, and cleaned up efficiently, preventing resource leaks and test interference.

#### 1. Backend Factory Fixtures

The main backend factory fixture:

- **`backend_factory`** (function-scoped):  
  Allows you to create and clean up multiple backend instances per test function.  
  **Note:** Users/accounts are created from scratch for each backend instance.
  
  Example usage:
  ```python
  import pytest

  class TestMessenger:
      @pytest.fixture(autouse=True)
      def setup_backends(self, backend_factory):
          self.sender = backend_factory("sender")
          self.receiver = backend_factory("receiver")

      def test_send_message(self):
          # Use self.sender and self.receiver in your test
          self.sender.send_message(self.receiver, "Hello!")
          assert self.receiver.has_received_message("Hello!")
  ```
  All containers created by this fixture are automatically stopped and removed after the test.

#### 2. Specialized Backend Fixtures

For more specific use cases, you can use these specialized fixtures:

- **`backend_new_profile`** (function-scoped):  
  Creates a backend with a new user profile from scratch.
  
  Example usage:
  ```python
  def test_with_new_profile(self, backend_new_profile):
      # Create a backend with a new profile
      client = backend_new_profile("client1")
      
      # Or with light client mode
      light_client = backend_new_profile("light_client", waku_light_client=True)
  ```
  
- **`backend_recovered_profile`** (function-scoped):  
  Creates a backend by recovering an existing user profile from a seed phrase.
  
  Example usage:
  ```python
  from resources.constants import user_1
  
  def test_with_recovered_profile(self, backend_recovered_profile):
      # Recover a backend from an existing user
      client = backend_recovered_profile("client1", user=user_1)
      
      # Or with light client mode
      light_client = backend_recovered_profile("light_client", user=user_1, waku_light_client=True)
  ```

- **`waku_light_client`** (function-scoped):  
  A simple parametrization fixture for enabling/disabling Waku light client mode.

#### 3. Manual/Custom Backend Setup

For advanced scenarios, you may set up backends manually in your test or test class (for example, to use custom logic or direct instantiation).  
**In this case, you must ensure containers are cleaned up by using the `close_status_backend_containers` fixture.**

Example (from `test_local_pairing.py`):
```python
@pytest.fixture(autouse=True)
def setup_cleanup(self, close_status_backend_containers):
    """Automatically cleanup containers after each test"""
    yield
```
This fixture will stop, save logs, and remove all containers created during the test, preventing resource leaks.

#### 4. Summary Table

> **Note:** None of these fixtures are used automatically (`autouse=False` by default). You must explicitly use them in your test setup for visibility and control.

| Fixture                           | Scope     | Usage                        | Cleanup         | User Creation Type                |
|------------------------------------|-----------|------------------------------|-----------------|-----------------------------------|
| `backend_factory`                  | function  | Per-test backend creation    | Automatic       | Created from scratch              |
| `backend_new_profile`              | function  | New user profile creation    | Automatic       | Created from scratch              |
| `backend_recovered_profile`        | function  | Existing profile recovery    | Automatic       | Recovered from seed phrase        |
| `waku_light_client`                | function  | Light client parametrization | N/A             | N/A (configuration only)          |
| `close_status_backend_containers`  | function  | For manual/custom setup      | Must be used manually | N/A (depends on your setup)   |

#### 5. Best Practices

- **Prefer the factory fixtures** for most tests—they are robust, reusable, and handle cleanup for you.
- **Use specialized fixtures** (`backend_new_profile`, `backend_recovered_profile`) when you need specific profile creation behavior.
- **Use `close_status_backend_containers`** if you create backends manually or outside the provided factories.
- **Never leave containers running** after a test—always ensure cleanup is in place.

For more details, see the docstrings in `tests/conftest.py` and the example usages in the test files.

#### 6. Recent Architectural Improvements

The test fixture architecture has been recently improved with the following changes:
- **Separation of Concerns**: Backend fixtures moved from root `conftest.py` to `tests/conftest.py` for better organization
- **Simplified Architecture**: Removed complex class-scoped fixtures in favor of more predictable function-scoped ones
- **Better Fixture Wrappers**: Added specialized `backend_new_profile` and `backend_recovered_profile` fixtures for specific use cases
- **Network Configuration**: Network ID is now defined directly in `StatusBackend` for cleaner configuration management
- **Python Compatibility**: Fixed typing issues for better Python 3.10+ compatibility 

# Linting

We use `pre-commit` setup for linting. To run linters:
```shell
cd tests-functional
make lint
```
or run from the root of `status-go` repo:
```shell
make pytest-lint
```

Note that `pyright` uses `pyrightconfig.json` configuration from the repository root 
and expects a Python virtual environment to be in `./tests-functional/.venv`. 

# Known issues

## Import issues

If you see some import issues, make sure that you have all requirements installed from `requirements.txt`

### For PyCharm users

Make sure that you made `test-functional` source folder (Right click > Mark directory as > Source folder)


## Reliability tests

This framework also hosts a suite of reliability tests. See more info [here](https://github.com/status-im/status-go/blob/develop/tests-functional/tests/reliability/README.MD)

## RuntimeError: Docker image 'statusgo-xyz:latest' not found

First check with `docker images` that the image was created. If not, re-run the build command from [How to Run](#how-to-run)

Second, if the image is present in the list, check if you Python env is not connected to the wrong Docker daemon.

Run this in the shell:
```shell
echo $DOCKER_HOST
docker context ls
```

Run this in the Python env (`python`):
```py
import docker
print(docker.from_env().api.base_url)
```

You might see that Python is using something like:
- unix:///run/docker.sock
- vs your CLI using unix:///var/run/docker.sock

If that is so, force Python to use the same Docker socket (from `docker context ls`):
```shell
export DOCKER_HOST=unix:///var/run/docker.sock
```
Add this before launching your test runner