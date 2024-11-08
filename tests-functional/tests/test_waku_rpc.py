import random
import time
from dataclasses import dataclass
from typing import Optional

import pytest

from conftest import option
from test_cases import StatusBackendTestCase


class TestRpc(StatusBackendTestCase):

    @pytest.mark.parametrize(
        "method, params",
        [
            ("wakuext_peers", []),
            ("wakuext_activityCenterNotifications", [{"cursor": "", "limit": 20, "activityTypes": [5], "readType": 2}])
        ],
    )
    def test_(self, method, params):
        _id = str(random.randint(1, 8888))

        response = self.rpc_client.rpc_valid_request(method, params, _id)
        self.rpc_client.verify_json_schema(response.json(), method)


@pytest.mark.skip("to be reworked via status-backend")
class TestRpcMessaging(StatusBackendTestCase):
    @dataclass
    class User:
        rpc_url: str
        chat_public_key: Optional[str] = None
        chat_id: Optional[str] = None

    def test_add_contact(self):
        _id = str(random.randint(1, 8888))

        self.user_1 = self.User(rpc_url=option.rpc_url)
        self.user_2 = self.User(rpc_url=option.rpc_url_2)

        # get chat public key
        for user in self.user_1, self.user_2:
            response = self.rpc_client.rpc_request(
                "accounts_getAccounts", [], _id, url=user.rpc_url
            )
            self.rpc_client.verify_is_valid_json_rpc_response(response)

            user.chat_public_key = next(
                (
                    item["public-key"]
                    for item in response.json()["result"]
                    if item["chat"]
                ),
                None,
            )

        # send contact requests
        for sender in self.user_1, self.user_2:
            for receiver in self.user_1, self.user_2:
                if sender != receiver:
                    response = self.rpc_client.rpc_request(
                        method="wakuext_sendContactRequest",
                        params=[
                            {
                                "id": receiver.chat_public_key,
                                "message": f"contact request from {sender.chat_public_key}: sent at {time.time()}",
                            }
                        ],
                        request_id=99,
                        url=sender.rpc_url,
                    )

                    self.rpc_client.verify_is_valid_json_rpc_response(response)
                    sender.chat_id = response.json()["result"]["chats"][0]["lastMessage"]["id"]

        # accept contact requests
        for user in self.user_1, self.user_2:
            response = self.rpc_client.rpc_request(
                method="wakuext_acceptContactRequest",
                params=[
                    {
                        "id": user.chat_id,
                    }
                ],
                request_id=99,
                url=user.rpc_url,
            )
            self.rpc_client.verify_is_valid_json_rpc_response(response)

        # verify contacts
        for user in (self.user_1, self.user_2), (self.user_2, self.user_1):
            response = self.rpc_client.rpc_request(
                method="wakuext_contacts",
                params=[],
                request_id=99,
                url=user[0].rpc_url,
            )
            self.rpc_client.verify_is_valid_json_rpc_response(response)

            response = response.json()
            assert response["result"][0]["added"] is True
            assert response["result"][0]["id"] == user[1].chat_public_key
