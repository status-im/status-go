import logging
from uuid import uuid4

import pytest

from steps import messenger
from utils import fake


@pytest.mark.rpc
class TestSharedURLs:

    def test_create_and_share_community_channel_url(self, backend_new_profile):
        backend = backend_new_profile("backend")
        community_id = messenger.create_community(backend)

        chat_request = {
            "identity": fake.community_channel_identity(),
            "viewersCanPostReactions": False,
            "hideIfPermissionsNotMet": False,
            "permissions": {"access": 1},
        }

        response = backend.wakuext_service.create_community_chat(community_id, chat_request)
        chats = response.get("chats")
        communities = response.get("communities")
        assert len(chats) == 1
        assert len(communities) == 1

        logging.info(f"Created community chat: {chats}")

        channel_id = str(uuid4())
        share_resp = backend.sharedurls_service.share_community_channel_url_with_chat_key(community_id, channel_id)
        assert share_resp is not None
        assert f"https://status.app/cc/{channel_id}" in share_resp

        parsed = backend.sharedurls_service.parse_shared_url(share_resp)
        assert parsed["channel"].get("channelUuid") == channel_id
        assert parsed["community"].get("communityId") == community_id

        share_comm_resp = backend.sharedurls_service.share_community_url_with_chat_key(community_id)
        assert "https://status.app/c" in share_comm_resp
        parsed_comm = backend.sharedurls_service.parse_shared_url(share_comm_resp)
        assert parsed_comm["community"].get("communityId") == community_id

    def test_share_user_url(self, backend_new_profile):
        backend = backend_new_profile("backend")
        serialized_key = backend.serialize_legacy_key(backend.public_key)
        logging.info(f"User public key: {serialized_key}")

        # Share user URL with chat key
        share_resp = backend.sharedurls_service.share_user_url_with_chat_key(backend.public_key)
        assert share_resp is not None
        assert "https://status.app/u" in share_resp
        logging.info(f"Shared user URL: {share_resp}")

        # Parse the shared URL and verify the public key
        parsed = backend.sharedurls_service.parse_shared_url(share_resp)
        assert parsed is not None
        assert "contact" in parsed
        assert parsed["contact"].get("publicKey") == serialized_key

        # Test share user URL with data
        share_data_resp = backend.sharedurls_service.share_user_url_with_data(backend.public_key)
        assert share_data_resp is not None
        assert "https://status.app/u" in share_data_resp

        # Parse and verify
        parsed_data = backend.sharedurls_service.parse_shared_url(share_data_resp)
        assert parsed_data is not None
        assert parsed_data["contact"].get("publicKey") == serialized_key
