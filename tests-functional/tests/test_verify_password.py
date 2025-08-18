import pytest


@pytest.mark.rpc
class TestVerifyPassword:

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        self.account = backend_new_profile("sender")

    def test_verify_correct_password(self):
        response = self.account.accounts_service.verify_password(self.account.password)
        assert response.get("result") is True

    @pytest.mark.parametrize("password", ["testpassword", ""])
    def test_verify_wrong_password(self, password):
        response = self.account.accounts_service.verify_password(password)
        assert response.get("result") is False
