import pytest


@pytest.mark.rpc
class TestGetRandomMnemonic:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("rpc_client")

    def test_get_random_mnemonic(self):
        result = self.account.accounts_service.get_random_mnemonic()
        assert len(result.split()) == 12  # basic check for mnemonic length
