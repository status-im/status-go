import hashlib
import os
import tempfile
import pytest
import requests

from clients.signals import SignalType
from clients.status_backend import StatusBackend
from utils import fake
from utils.image_utils import ImageCropRect
from clients.services.wakuext import CommunityPermissionsAccess


ASSETS_DIR = os.path.join(os.path.dirname(os.path.dirname(os.path.dirname(__file__))), "_assets", "tests")
ELEPHANT = os.path.join(ASSETS_DIR, "elephant.jpg")
STATUS = os.path.join(ASSETS_DIR, "status.png")
IMAGE_SIZES = ["large", "thumbnail"]


def sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


# Helper to images from media server
def fetch_image(url, cert_file_path) -> bytes:
    r = requests.get(url, timeout=10, verify=cert_file_path)
    r.raise_for_status()
    return r.content


def validate_community_image(community, size, name, cert_file_path):
    img1 = fetch_image(community["images"][size]["uri"], cert_file_path)
    assert img1 is not None and len(img1) > 0, "Community image not available from media server"

    # Load expected image and validate
    expected_path = os.path.join(ASSETS_DIR, f"{name}_expected_{size}.jpg")
    with open(expected_path, "rb") as f:
        expected_img = f.read()

    assert sha256(img1) == sha256(expected_img), f"Community image '{size}' does not match expected reference"


def validate_community_images(community, name, cert_file_path):
    for size in IMAGE_SIZES:
        validate_community_image(community, size, name, cert_file_path)


@pytest.mark.rpc
@pytest.mark.create_account
class TestEditCommunity:
    await_signals = [
        SignalType.MEDIASERVER_STARTED.value,
        SignalType.NODE_STARTED.value,
        SignalType.NODE_READY.value,
        SignalType.NODE_LOGIN.value,
    ]

    @pytest.fixture()
    def backend(self, backend_new_profile) -> StatusBackend:
        return backend_new_profile("sender")

    def test_edit_community_image(self, backend):
        backend.wait_for_signal(SignalType.MEDIASERVER_STARTED.value)

        # Save certificate to temporary file
        certificate = backend.image_server_tls_cert()
        with tempfile.NamedTemporaryFile(mode="w", suffix=".crt", delete=False) as cert_file:
            cert_file.write(certificate)
            cert_file_path = cert_file.name

        # Prepare assets inside backend environment
        dest_assets = os.path.join("tmp", "images")
        backend.import_data(ELEPHANT, dest_assets)
        backend.import_data(STATUS, dest_assets)
        elephant_path = os.path.join(dest_assets, os.path.basename(ELEPHANT))
        status_path = os.path.join(dest_assets, os.path.basename(STATUS))

        # Create a community with initial image
        create_resp = backend.wakuext_service.create_community(
            name=fake.community_name(),
            description=fake.community_description(),
            membership=CommunityPermissionsAccess.AUTO_ACCEPT,
            image=elephant_path,
            image_rect=ImageCropRect(10, 10, 70, 70),
        )
        communities = create_resp.get("communities", [])
        community = communities[0]
        validate_community_images(community, "elephant", cert_file_path)

        # Edit community to change image
        edit_resp = backend.wakuext_service.edit_community(
            community_id=community["id"],
            name=fake.community_name(),
            description=fake.community_description(),
            membership=CommunityPermissionsAccess.AUTO_ACCEPT,
            image=status_path,
            image_rect=ImageCropRect(40, 40, 200, 200),
        )
        community = edit_resp.get("communities")[0]
        validate_community_images(community, "status", cert_file_path)
