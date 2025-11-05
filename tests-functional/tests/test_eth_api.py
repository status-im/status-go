import pytest

from resources.constants import user_1, user_2


@pytest.mark.rpc
@pytest.mark.ethclient
@pytest.mark.xdist_group(name="Eth")
class TestEth:

    def test_estimate_gas(self, backend_recovered_profile):
        backend = backend_recovered_profile("sender", user=user_1)
        result = backend.eth_service.estimate_gas(31337, user_2.address, 100)
        assert int(result, 16) > 0
