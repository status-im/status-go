## Overview

Functional tests for status-go

## Table of Contents

- [Overview](#overview)
- [How to Install](#how-to-install)
- [How to Run](#how-to-run)
  - [Running Tests](#running-tests)
- [Implementation details](#implementation-details)

## How to Install

* Install [Docker](https://docs.docker.com/engine/install/) and [Docker Compose](https://docs.docker.com/compose/install/)
* Install [Python 3.10.14](https://www.python.org/downloads/)
* In `./tests-functional`, run `pip install -r requirements.txt`
* **Optional (for test development)**: Use Python virtual environment for better dependency management. You can follow the guide [here](https://akrabat.com/creating-virtual-environments-with-pyenv/):

## How to Run

### Running dev RPC (anvil with contracts)
- In `./tests-functional` run `docker compose -f docker-compose.anvil.yml up --remove-orphans --build`, as result:
    * an [anvil](https://book.getfoundry.sh/reference/anvil/) container with ChainID 31337 exposed on `0.0.0.0:8545` will start running
    * Status-im contracts will be deployed to the network

### Run tests
- In `./tests-functional` run `docker compose -f docker-compose.anvil.yml -f docker-compose.test.status-go.yml -f docker-compose.status-go.local.yml up --build --scale status-backend=10 --remove-orphans`, as result:
    * a container with [status-go as daemon](https://github.com/status-im/status-go/issues/5175) will be created with APIModules exposed on `0.0.0.0:3333`
    * status-go will use [anvil](https://book.getfoundry.sh/reference/anvil/) as RPCURL with ChainID 31337 
    * all Status-im contracts will be deployed to the network

* In `./tests-functional/tests` directory run `pytest -m wallet` 

## Implementation details

- Functional tests are implemented in `./tests-functional/tests` based on [pytest](https://docs.pytest.org/en/8.2.x/)
- Every test has two types of verifications:
    - `verify_is_valid_json_rpc_response()` checks for status code 200, non-empty response, JSON-RPC structure, presence of the `result` field, and expected ID.
    - `jsonschema.validate()` is used to check that the response contains expected data, including required fields and types. Schemas are stored in `/schemas/wallet_MethodName`
    - New schemas can be generated using `./tests-functional/schema_builder.py` by passing a response to the `CustomSchemaBuilder(schema_name).create_schema(response.json())` method, should be used only on test creation phase, please search `how to create schema:` to see an example in a test