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
    docker build --file _assets/build/Dockerfile --tag "statusgo-$(git rev-parse --short HEAD)" .
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
- Every test has two types of verifications:
    - `verify_is_valid_json_rpc_response()` checks for status code 200, non-empty response, JSON-RPC structure, presence of the `result` field, and expected ID.
    - `jsonschema.validate()` is used to check that the response contains expected data, including required fields and types. Schemas are stored in `/schemas/wallet_MethodName`
    - New schemas can be generated using `./tests-functional/utils/schema_builder.py` by passing a response to the `CustomSchemaBuilder(schema_name).create_schema(response.json())` method, should be used only on test creation phase, please search `how to create schema:` to see an example in a test

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