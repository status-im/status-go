import pytest

from resources.constants import BURN_ADDRESS


@pytest.mark.rpc
@pytest.mark.ethclient
class TestEth:

    def test_estimate_gas(self, funded_new_profile):
        backend, wallet_address = funded_new_profile("sender")
        result = backend.eth_service.estimate_gas(31337, BURN_ADDRESS, 100)
        assert int(result, 16) > 0
