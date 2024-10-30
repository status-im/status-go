## Overview

Functional tests for `status-go`

## Table of Contents

- [Overview](#overview)
- [How to Install](#how-to-install)
- [How to Run](#how-to-run)
  - [Running Tests](#running-tests)
- [Implementation Details](#implementation-details)
- [Build Status Backend](#build-status-backend)

## How to Install

1. Install [Docker](https://docs.docker.com/engine/install/) and [Docker Compose](https://docs.docker.com/compose/install/)
2. Install [Python 3.10.14](https://www.python.org/downloads/)
3. **Set up a virtual environment (recommended):**
   - In `./tests-functional`, run:
     ```bash
     python3 -m venv .venv
     source .venv/bin/activate
     pip install -r requirements.txt
     ```
   - **Optional (for test development)**: Use Python virtual environment for better dependency management. You can follow the guide [here](https://akrabat.com/creating-virtual-environments-with-pyenv/)

## How to Run

### Running dev RPC (Anvil with contracts)

In `./tests-functional`:
```bash
docker compose -f docker-compose.anvil.yml up --remove-orphans --build
```

This command will:
- Start an [Anvil](https://book.getfoundry.sh/reference/anvil/) container with ChainID `31337`, exposed on `0.0.0.0:8545`
- Deploy Status-im contracts to the Anvil network

### Running Tests

To run the tests:

1. In `./tests-functional`, start the testing containers:
   ```bash
   docker compose -f docker-compose.anvil.yml -f docker-compose.test.status-go.yml -f docker-compose.status-go.local.yml up --build --remove-orphans
   ```

   This command will:
   - Create a container with [status-go as daemon](https://github.com/status-im/status-go/issues/5175), exposing `APIModules` on `0.0.0.0:3333`
   - Configure `status-go` to use [Anvil](https://book.getfoundry.sh/reference/anvil/) as the `RPCURL` with ChainID `31337`
   - Deploy all Status-im contracts to the Anvil network

2. To execute tests:
   - Run all tests:
     ```bash
     pytest
     ```
   - Run tests marked as `wallet`:
     ```bash
     pytest -m wallet
     ```
   - Run a specific test:
     ```bash
     pytest -k "test_contact_request_baseline"
     ```

## Implementation Details

- Functional tests are implemented in `./tests-functional/tests` using [pytest](https://docs.pytest.org/en/8.2.x/).
- Each test performs two types of verifications:
  - **`verify_is_valid_json_rpc_response()`**: Checks for a status code `200`, a non-empty response, JSON-RPC structure, presence of the `result` field, and the expected ID.
  - **`jsonschema.validate()`**: Validates that the response contains expected data, including required fields and types. Schemas are stored in `/schemas/wallet_MethodName`.

- **Schema Generation**:
  - New schemas can be generated with `./tests-functional/schema_builder.py` by passing a response to the `CustomSchemaBuilder(schema_name).create_schema(response.json())` method. This should be used only during test creation.
  - Search `how to create schema:` in test files for examples.

## Build Status Backend

You can manually build the binary with the following command in the `status-go` root directory:

```bash
make status-backend
```

For further details on building and setting up `status-go` and `status-backend`, refer to the official documentation:
- [status-backend README](https://github.com/status-im/status-go/blob/develop/cmd/status-backend/README.md)
- [status-go cmd directory](https://github.com/status-im/status-go/tree/develop/cmd/status-backend)

Location of the binary: `cmd/status-backend/status-backend`

In test build is automatically being build and placed in right path. If build already exists then new build is not generated.