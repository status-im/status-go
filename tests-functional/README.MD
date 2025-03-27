## Overview

Functional tests for status-go

## Table of Contents

- [Overview](#overview)
- [How to Install](#how-to-install)
- [How to Run](#how-to-run)
  - [Running Tests](#running-tests)
- [Implementation details](#implementation-details)

## How to Install

1. Install [Docker](https://docs.docker.com/engine/install/) and [Docker Compose](https://docs.docker.com/compose/install/)
2. Install [Python 3](https://www.python.org/downloads/) (tested with 3.10 and 3.12 and it works with both)
3. **Set up a virtual environment (required for linting):**
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

### Running dev RPC (anvil with contracts)
- In `./tests-functional` run `docker compose -f docker-compose.anvil.yml up --remove-orphans --build`, as result:
    * an [anvil](https://book.getfoundry.sh/reference/anvil/) container with ChainID 31337 exposed on `0.0.0.0:8545` will start running
    * Status-im contracts will be deployed to the network

### Run tests
- In `./tests-functional` run `docker compose -f docker-compose.anvil.yml -f docker-compose.test.status-go.yml -f docker-compose.status-go.local.yml up --build --remove-orphans`, as result:
    * a container with [status-backend](https://github.com/status-im/status-go/pull/5847) will be created with endpoint exposed on `0.0.0.0:3333`
    * status-go will use [anvil](https://book.getfoundry.sh/reference/anvil/) as RPCURL with ChainID 31337 
    * Status-im contracts will be deployed to the network

* In `./tests-functional/tests` directory run `pytest -m rpc`
* To run tests against binary run `pytest -m <your mark> --status_backend_url=http://<binary_url>:<binary_port>`

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

# Known issues

## Import issues

If you see some import issues, make sure that you have all requirements installed from `requirements.txt` 

### For PyCharm users
Make sure that you made `test-functional` source folder (Right click > Mark directory as > Source folder)
