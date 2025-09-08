import pytest


@pytest.mark.rpc
class TestResolveSuggestedPath:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("sender")

    @pytest.mark.parametrize(
        "key_uid", ["0x9d26312bace61a229c1db5a43d8f9b93d80d25398476405d66479df1504f0438", "0xC43f4Ab94eC965a3EE9815C5Df07383057d261A8", "test"]
    )
    def test_resolve_suggested_path_for_random_key(self, key_uid):
        response = self.account.accounts_service.resolve_suggested_path_for_keypair(key_uid)
        assert response == "m/44'/60'/0'/0/0"

    def test_resolve_suggested_path_for_used_key(self):
        response = self.account.accounts_service.resolve_suggested_path_for_keypair(self.account.key_uid)
        assert response == "m/44'/60'/0'/0/1"
