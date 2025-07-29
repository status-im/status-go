from clients.status_backend import StatusBackend
import pytest
from resources.constants import Account


@pytest.mark.create_account
@pytest.mark.rpc
class TestRestoreAccount:

    @pytest.fixture(autouse=True)
    def setup_cleanup(self, close_status_backend_containers):
        """Automatically cleanup containers after each test"""
        yield

    def test_restore_account_with_seed_phrase(self):
        # Create a container with an account
        first_account = StatusBackend()
        first_account.init_status_backend()
        first_account.create_account_and_login()

        # Retrieve and save the seed phrase
        first_account_settings = first_account.settings_service.get_settings()
        seed_phrase = first_account_settings.get("result", {}).get("mnemonic", None)
        assert seed_phrase

        user_1 = Account(
            address="0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
            private_key="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
            password="Strong12345",
            passphrase=seed_phrase,
        )

        # Create a second container and restore the 1st account using the saved seed phrase
        second_account = StatusBackend()
        second_account.init_status_backend()
        second_account.restore_account_and_login(user=user_1)

        # Check that seed phrase is no longer present
        second_account_settings = second_account.settings_service.get_settings()
        seed_phrase = second_account_settings.get("result", {}).get("mnemonic", None)
        assert not seed_phrase
