import asyncio
import tempfile

import pytest
import requests


def _media_server_health(port, certificate):
    with tempfile.NamedTemporaryFile(mode="w", suffix=".crt", delete=False) as cert_file:
        cert_file.write(certificate)
        cert_file_path = cert_file.name

    return requests.get(f"https://localhost:{port}/health", timeout=10, verify=cert_file_path)


@pytest.mark.rpc
@pytest.mark.asyncio
class TestMediaServer:

    async def test_media_server_health(self, async_backend_new_profile):
        backend = await async_backend_new_profile("sender")

        media_server_url = f"https://localhost:{backend.backend.media_server_port}"
        certificate = backend.backend.image_server_tls_cert()

        with tempfile.NamedTemporaryFile(mode="w", suffix=".crt", delete=False) as cert_file:
            # Save the certificate to a temporary file
            cert_file.write(certificate)
            cert_file_path = cert_file.name

        # Make a health request to the media server with the loaded certificate
        response = requests.get(media_server_url + "/health", timeout=10, verify=cert_file_path)
        assert response.status_code == 200

    async def test_multiple_media_servers(self, async_backend_new_profile):
        backend_1, backend_2 = await asyncio.gather(
            async_backend_new_profile("backend_1"),
            async_backend_new_profile("backend_2"),
        )

        # Media server ports are available directly - no need to wait for signals
        # (signals arrive before async signal client connects, so they're always missed)

        with tempfile.NamedTemporaryFile(mode="w", suffix=".crt", delete=False) as cert_file:
            cert_file.write(backend_1.backend.image_server_tls_cert())
            cert_file_path_1 = cert_file.name

        with tempfile.NamedTemporaryFile(mode="w", suffix=".crt", delete=False) as cert_file:
            cert_file.write(backend_2.backend.image_server_tls_cert())
            cert_file_path_2 = cert_file.name

        response = requests.get(f"https://localhost:{backend_1.backend.media_server_port}/health", timeout=10, verify=cert_file_path_1)
        assert response.status_code == 200

        response = requests.get(f"https://localhost:{backend_2.backend.media_server_port}/health", timeout=10, verify=cert_file_path_2)
        assert response.status_code == 200

    async def test_media_server_port_stable_after_reinitialize(self, async_backend_new_profile):
        backend = await async_backend_new_profile("sender")

        port = backend.backend.media_server_port
        assert _media_server_health(port, backend.backend.image_server_tls_cert()).status_code == 200

        backend.backend.logout()
        backend.backend.init_status_backend()  # re-Initialize with the same port

        assert _media_server_health(port, backend.backend.image_server_tls_cert()).status_code == 200
