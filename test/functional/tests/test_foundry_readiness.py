"""Foundry setup regression tests; no Docker daemon is required."""

from types import SimpleNamespace
from unittest.mock import Mock

import pytest

from clients.foundry import Foundry
from resources.constants import ENS_ADDRESSES_CONTAINER_PATH

pytestmark = pytest.mark.rpc


@pytest.fixture
def foundry(monkeypatch):
    client = Foundry.__new__(Foundry)
    client.container = Mock(status="running", attrs={"State": {"ExitCode": 1}})
    client.container.logs.return_value = b"Deploying ENS contracts"
    client.is_connected = Mock(return_value=True)
    clock = [0.0]
    monkeypatch.setattr("clients.foundry.time.monotonic", lambda: clock[0])

    def sleep(seconds):
        clock[0] += seconds

    monkeypatch.setattr("clients.foundry.time.sleep", sleep)
    return client


def test_waits_for_ens_even_when_anvil_is_connected(foundry):
    # Anvil is already up, but ENS is deployed after the other contracts.
    readiness = iter([False, False, True])

    def exec_run(command):
        assert f"test -s {ENS_ADDRESSES_CONTAINER_PATH}" in command[2]
        return SimpleNamespace(exit_code=0 if next(readiness) else 1)

    foundry.container.exec_run.side_effect = exec_run
    foundry.wait_for_healthy(timeout=10)
    assert foundry.container.exec_run.call_count == 3


def test_waits_for_anvil_connectivity(foundry):
    foundry.is_connected.side_effect = [False, True]
    foundry.container.exec_run.return_value = SimpleNamespace(exit_code=0)
    foundry.wait_for_healthy(timeout=10)
    assert foundry.is_connected.call_count == 2


def test_deployment_exit_reports_original_clone_error(foundry):
    foundry.container.status = "exited"
    foundry.container.logs.return_value = b"fatal: could not read Username for 'https://github.com'"
    with pytest.raises(RuntimeError, match="(?s)exited.*code 1.*could not read Username"):
        foundry.wait_for_healthy()
    foundry.container.exec_run.assert_not_called()


def test_deployment_timeout_includes_logs(foundry):
    foundry.container.exec_run.return_value = SimpleNamespace(exit_code=1)
    with pytest.raises(TimeoutError, match="(?s)not ready after 3 seconds.*Deploying ENS contracts"):
        foundry.wait_for_healthy(timeout=3)


def test_finds_exited_container_to_report_deployment_failure(monkeypatch):
    container = Mock(name="foundry", status="exited", attrs={"State": {"ExitCode": 1}})
    container.logs.return_value = b"git clone failed"
    docker_client = Mock()
    docker_client.containers.list.return_value = [container]
    docker_client.containers.get.return_value = container
    monkeypatch.setattr("clients.foundry.docker.from_env", lambda: docker_client)
    with pytest.raises(RuntimeError, match="git clone failed"):
        Foundry()
    assert docker_client.containers.list.call_args.kwargs["all"] is True
