import pytest


@pytest.mark.rpc
class TestGetRandomMnemonic:

    @pytest.fixture()
    def backend(self, backend_new_profile):
        return backend_new_profile("rpc-client-backend")

    def test_get_random_mnemonic(self, backend):
        result = backend.accounts_service.get_random_mnemonic()
        assert len(result.split()) == 12  # basic check for mnemonic length
