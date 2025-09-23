import os
import pytest
from resources.constants import user_1


@pytest.mark.accounts
@pytest.mark.rpc
class TestAccounts:

    @pytest.fixture(autouse=True)
    def setup_backend(self, backend_new_profile):
        """Initialize one backend for each test function"""
        self.rpc_client = backend_new_profile("rpc_client")

    def test_get_identity_images(self):
        result = self.rpc_client.multiaccounts_service.get_identity_images(user_1.private_key)
        assert result is None

    def test_store_identity_image(self):
        self.rpc_client.import_data(
            os.path.abspath(os.path.join(os.path.dirname(__file__), "../resources/images/test-image-200x200.jpg")), "/tmp/images/"
        )
        result = self.rpc_client.multiaccounts_service.store_identity_image(
            self.rpc_client.key_uid, "/tmp/images/test-image-200x200.jpg", 0, 0, 200, 200
        )

        assert result is not None, "Identity images were not stored (no result)"
        assert len(result) == 2, "Identity images were not stored (wrong count)"
        assert result[0]["localUrl"] is not None, "Local URL for identity image 1 is not set"
        assert result[1]["localUrl"] is not None, "Local URL for identity image 2 is not set"

        result = self.rpc_client.multiaccounts_service.get_identity_images(self.rpc_client.key_uid)

        assert result is not None, "Identity images were not retrieved (no result)"
        assert len(result) == 2, "Identity images were not retrieved (wrong count)"
        assert result[0]["localUrl"] is not None, "Local URL for identity image 1 is not set"
        assert result[1]["localUrl"] is not None, "Local URL for identity image 2 is not set"
