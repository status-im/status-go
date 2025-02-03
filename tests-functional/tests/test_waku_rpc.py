import random
import time
from dataclasses import dataclass
from typing import Optional

import pytest

from conftest import option
from test_cases import StatusBackendTestCase, MessengerTestCase


class TestRpc(StatusBackendTestCase):

    @pytest.mark.parametrize(
        "method, params",
        [
            ("wakuext_peers", []),
            (
                "wakuext_activityCenterNotifications",
                [{"cursor": "", "limit": 20, "activityTypes": [5], "readType": 2}],
            ),
        ],
    )
    def test_(self, method, params):
        _id = str(random.randint(1, 8888))

        response = self.rpc_client.rpc_valid_request(method, params, _id)
        self.rpc_client.verify_json_schema(response.json(), method)


@pytest.mark.rpc
class TestRpcMessaging(MessengerTestCase):

    @pytest.mark.usefixtures("setup_two_nodes")
    def test_one_to_one_message(self):
        self.one_to_one_message(5)

    def test_add_contact(self):
        self.add_contact(execution_number=1, network_condition=None)
