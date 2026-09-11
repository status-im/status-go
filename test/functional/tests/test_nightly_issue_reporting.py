"""Temporary deliberate failures for the Jenkins issue-reporting experiment. Remove after validation."""

import pytest

pytestmark = pytest.mark.rpc


@pytest.mark.parametrize("peer", ["sender", "receiver"])
def test_nightly_shared_assertion(peer):
    assert False, "Deliberate shared failure: nightly peer is offline"


def test_nightly_unique_assertion():
    assert 1 == 2, "Deliberate unique failure: message count mismatch"


def test_nightly_unique_exception():
    raise RuntimeError("Deliberate unique failure: store request timed out")


@pytest.fixture
def broken_nightly_setup():
    raise ValueError("Deliberate unique setup error: invalid fleet configuration")


def test_nightly_setup_error(broken_nightly_setup):
    pass
