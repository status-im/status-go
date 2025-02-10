import pytest
import logging

from test_cases import StatusBackendTestCase


@pytest.mark.rpc
@pytest.mark.wallet
class TestSavedAddresses(StatusBackendTestCase):

    @pytest.mark.parametrize(
        "method, params",
        [
            (
                "wakuext_upsertSavedAddress",
                [
                    {
                        "address": "0xcf2272205cc0cf96cfbb9dd740bd681d1e86901e",
                        "name": "some_random_address",
                        "colorId": "green",
                        "isTest": False,
                        "chainShortNames": "",
                    }
                ],
            ),
            (
                "wakuext_upsertSavedAddress",
                [
                    {
                        "address": "0x8e58eb36c7b77d6c43fc05c8fd3fe645d1d39588",
                        "mixedcaseAddress": "0x8e58eb36c7b77d6C43fC05C8Fd3FE645d1d39588",
                        "name": "yellow_ENS",
                        "colorId": "yellow",
                        "ens": "some_yellow_ENS.eth",
                        "isTest": False,
                        "chainShortNames": "",
                    }
                ],
            ),
            (
                "wakuext_upsertSavedAddress",
                [
                    {
                        "address": "0xc6a54e79fb8915efbe00a8adac5bd94b68022fb6",
                        "name": "test_address_pretty_long_name WITH Cap letters",
                        "colorId": "blue",
                        "ens": "test_some_yellow_ENS.eth",
                        "isTest": False,
                        "chainShortNames": "orb:opt",
                    }
                ],
            ),
        ],
    )
    def test_add_saved_address(self, method, params):
        """Test adding saved addresses and verifying their presence in the lists."""

        logging.info("Step: Adding item in mainnet mode")
        self.rpc_client.rpc_valid_request(method, params)
        response = self.rpc_client.rpc_valid_request("wakuext_getSavedAddresses", [])

        logging.info("Step: Verifying the item is in the saved addresses list")
        self.rpc_client.verify_json_schema(response.json(), "wakuext_getSavedAddresses")
        assert any(params[0].items() <= item.items() for item in response.json()["result"]), f"{params[0]['name']} not found in getSavedAddresses"

        logging.info("Step: Checking if the item is listed under mainnet saved addresses")
        response = self.rpc_client.rpc_valid_request("wakuext_getSavedAddressesPerMode", [False])
        self.rpc_client.verify_json_schema(response.json(), "wakuext_getSavedAddressesPerMode")
        assert any(
            params[0].items() <= item.items() for item in response.json()["result"]
        ), f"{params[0]['name']} not found in getSavedAddressesPerMode"

        logging.info("Step: Ensuring the item is NOT in the testnet saved addresses list")
        response = self.rpc_client.rpc_valid_request("wakuext_getSavedAddressesPerMode", [True])
        assert response.json()["result"] is None, "wakuext_getSavedAddressesPerMode for test mode is not empty"

    def test_delete_saved_address(self):
        """Test deleting a saved address and verifying its removal."""
        address, is_test = "0xc6a54e79fb8915efbe00a8adac5bd94b68022fb6", True
        params = [
            {
                "address": address,
                "name": "testnet_yellow_ENS",
                "colorId": "red",
                "ens": "some_red_ENS.stateofus.eth",
                "isTest": is_test,
            }
        ]

        logging.info("Step: Adding item in testnet mode")
        self.rpc_client.rpc_valid_request("wakuext_upsertSavedAddress", params)

        logging.info("Step: Verifying the item exists in testnet saved addresses")
        response = self.rpc_client.rpc_valid_request("wakuext_getSavedAddressesPerMode", [is_test])
        assert any(
            params[0].items() <= item.items() for item in response.json()["result"]
        ), f"{params[0]['name']} not found in getSavedAddressesPerMode"

        logging.info("Step: Deleting the item and verifying removal")
        self.rpc_client.rpc_valid_request("wakuext_deleteSavedAddress", [address, is_test])
        response = self.rpc_client.rpc_valid_request("wakuext_getSavedAddressesPerMode", [is_test])
        assert response.json()["result"] is None, "getSavedAddressesPerMode for test mode is not empty"

    def test_remaining_capacity_for_saved_addresses(self):
        """Test checking the remaining capacity for saved addresses."""
        is_test = False
        addresses = [
            "0x0a27AF951DAD6228Fd8A692992Be23527219FcaD",
            "0xAC65F396C9032e249F4c8Dc430531eEa57152fd4",
            "0x172cf0afc54C8A145bdD2781d04f3a1e2a074437",
            "0x4473a1AebEC875e8027544A032666C1A329021CE",
            "0x2E4FEC1aaE712dCAD560A69739d0DbB225Cd7c75",
            "0x90956c8d09D2651c7930996d61D21ba5D7D0bDf1",
            "0xc864d0Ec046ea0B8Edd00d49edfe3A19368C59F8",
            "0x89e03B5342c75a68FC61C85cC2a58bDd61Bf264e",
            "0x1049e21dE0fBDa877C2780A85875e135a73433F1",
            "0xc742524aEd5742aa75a5DcdCec1bFBe12Dc29BC0",
            "0xaE406B10C55924e96B5Ee267501B169ED37e6814",
            "0x9B6248818aab31018C54f9D63535D07bCeE061e5",
            "0x2C7b097427d0Da09A30d4FE9bD1aaBa956CE1538",
            "0xc9a087C44C7A098569cD9295593609050A0F292e",
            "0x901F90De45C31215b0F99b4F52F1D4b317302f72",
            "0xD73E3f566d3Eb55E9c292A798600F7dc0ece4351",
            "0xd06f43DEf63102A8137337eaafcF43c45D6a2708",
            "0x474fc93f36Aa0ed1A80c0D836cFB3acFE80C2D42",
            "0xE14e345d9bbadf6796DC7fEdf8f2625aDc11509b",
            "0x09B69c2F46E7F63131C54BAfae242EEc2C600762",
        ]

        logging.info("Step: Checking remaining capacity")
        response = self.rpc_client.rpc_valid_request("wakuext_remainingCapacityForSavedAddresses", [is_test])
        remaining_capacity = response.json()["result"]

        logging.info("Step: adding  addresses to fill capacity")
        for i in range(remaining_capacity):
            self.rpc_client.rpc_valid_request("wakuext_upsertSavedAddress", [{"address": addresses[i], "name": f"test{i}", "isTest": is_test}])

        logging.info("Step: Verifying that capacity is now 0")
        response = self.rpc_client.rpc_request("wakuext_remainingCapacityForSavedAddresses", [is_test])
        self.rpc_client.verify_is_json_rpc_error(response)
        assert response.json()["error"]["message"] == "no more save addresses can be added"
