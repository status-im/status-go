"""Temporary deliberate failures for the Jenkins issue-reporting experiment. Remove after validation."""

import pytest

pytestmark = pytest.mark.rpc


@pytest.mark.parametrize("peer", ["sender", "receiver", "relay"])
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


def test_nightly_repeated_store_timeout():
    raise RuntimeError("Deliberate unique failure: store request timed out")


@pytest.mark.parametrize("response", ["history", "messages"])
def test_nightly_shared_response_error(response):
    raise ValueError("Deliberate shared failure: malformed store response")


def test_nightly_unique_delivery_assertion():
    assert False, "Deliberate unique failure: delivery confirmation missing"
