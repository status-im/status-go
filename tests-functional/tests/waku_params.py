"""Shared pytest parametrization for waku light client tests."""

import pytest

# Fixture: ``waku_light_client``. Backend config key: ``wakuV2LightClient``.
# ``ids`` match the backend key and ``pytest -k`` (e.g. ``-k 'wakuV2LightClient_True'``).
parametrize_waku_light_client = pytest.mark.parametrize(
    "waku_light_client",
    [False, True],
    indirect=True,
    ids=["wakuV2LightClient_False", "wakuV2LightClient_True"],
)

parametrize_waku_light_client_non_lc = pytest.mark.parametrize(
    "waku_light_client",
    [False],
    indirect=True,
    ids=["wakuV2LightClient_False"],
)
