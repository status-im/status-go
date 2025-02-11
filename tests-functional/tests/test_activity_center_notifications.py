import pytest

from tests.test_cases import MessengerTestCase


@pytest.mark.usefixtures("setup_two_unprivileged_nodes")
@pytest.mark.rpc
class TestActivityCenterNotifications(MessengerTestCase):

    def test_activity_center_notifications(self):
        message_id = self.send_contact_request_and_wait_for_signal_to_be_received()
        response = self.receiver.wakuext_service.rpc_request(
            method="activityCenterNotifications", params=[{"cursor": "", "limit": 20, "activityTypes": [5], "readType": 2}]
        ).json()
        self.receiver.verify_json_schema(response, method="wakuext_activityCenterNotifications")
        result = response["result"]
        assert all(
            (
                result["notifications"][0]["accepted"] is False,
                result["notifications"][0]["author"] == self.sender.public_key,
                result["notifications"][0]["chatId"] == self.sender.public_key,
                result["notifications"][0]["id"] == message_id,
                result["notifications"][0]["read"] is False,
                result["notifications"][0]["lastMessage"]["contactRequestState"] == 1,
            )
        )

        self.accept_contact_request_and_wait_for_signal_to_be_received(message_id)
        response = self.sender.wakuext_service.rpc_request(
            method="activityCenterNotifications", params=[{"cursor": "", "limit": 20, "activityTypes": [1, 2, 3, 4, 5, 7, 8, 9, 10, 23, 24]}]
        ).json()
        self.sender.verify_json_schema(response, method="wakuext_activityCenterNotifications")
        result = response["result"]
        assert all(
            (
                result["notifications"][0]["accepted"] is True,
                result["notifications"][0]["author"] == self.sender.public_key,
                result["notifications"][0]["chatId"] == self.receiver.public_key,
                result["notifications"][0]["id"] == message_id,
                result["notifications"][0]["read"] is True,
                result["notifications"][0]["message"]["contactRequestState"] == 2,
                result["notifications"][0]["lastMessage"]["text"] == f"@{self.receiver.public_key} accepted your contact request",
            )
        )

    def test_activity_center_notifications_count(self):
        self.send_contact_request_and_wait_for_signal_to_be_received()
        response = self.receiver.wakuext_service.rpc_request(
            method="activityCenterNotificationsCount", params=[{"activityTypes": [1, 2, 3, 4, 5, 7, 8, 9, 10, 23, 24], "readType": 2}]
        ).json()
        self.receiver.verify_json_schema(response, method="wakuext_activityCenterNotificationsCount")
        assert response["result"]["5"] == 1

    def test_seen_unseen_activity_center_notifications(self):
        self.send_contact_request_and_wait_for_signal_to_be_received()
        response = self.receiver.wakuext_service.rpc_request(method="hasUnseenActivityCenterNotifications").json()
        self.receiver.verify_json_schema(response, method="wakuext_hasUnseenActivityCenterNotifications")
        assert response["result"] is True

        response = self.receiver.wakuext_service.rpc_request(method="markAsSeenActivityCenterNotifications").json()
        self.receiver.verify_json_schema(response, method="wakuext_markAsSeenActivityCenterNotifications")

        response = self.receiver.wakuext_service.rpc_request(method="hasUnseenActivityCenterNotifications").json()
        self.receiver.verify_json_schema(response, method="wakuext_hasUnseenActivityCenterNotifications")
        assert response["result"] is False

    def test_get_activity_center_state(self):
        self.send_contact_request_and_wait_for_signal_to_be_received()
        response = self.receiver.wakuext_service.rpc_request(method="getActivityCenterState").json()
        self.receiver.verify_json_schema(response, method="wakuext_getActivityCenterState")
        assert response["result"]["hasSeen"] is False

        self.receiver.wakuext_service.rpc_request(method="markAsSeenActivityCenterNotifications").json()

        response = self.receiver.wakuext_service.rpc_request(method="getActivityCenterState").json()
        assert response["result"]["hasSeen"] is True

    def test_mark_all_activity_center_notifications_read(self):
        message_id = self.send_contact_request_and_wait_for_signal_to_be_received()
        response = self.receiver.wakuext_service.rpc_request(method="markAllActivityCenterNotificationsRead").json()
        self.receiver.verify_json_schema(response, method="wakuext_markAllActivityCenterNotificationsRead")
        assert all(
            (
                response["result"]["activityCenterState"]["hasSeen"] is True,
                response["result"]["activityCenterNotifications"][0]["id"] == message_id,
                response["result"]["activityCenterNotifications"][0]["read"] is True,
            )
        )

        response = self.receiver.wakuext_service.rpc_request(method="hasUnseenActivityCenterNotifications").json()
        assert response["result"] is False

    def test_mark_activity_center_notifications_read_unread(self):
        message_id = self.make_contacts()
        response = self.sender.wakuext_service.rpc_request(
            method="markActivityCenterNotificationsRead",
            params=[
                [
                    message_id,
                ],
            ],
        ).json()
        self.sender.verify_json_schema(response, method="wakuext_markActivityCenterNotificationsRead")
        assert all(
            (
                response["result"]["activityCenterState"]["hasSeen"] is True,
                response["result"]["activityCenterNotifications"][0]["id"] == message_id,
                response["result"]["activityCenterNotifications"][0]["read"] is True,
            )
        )

        result = self.sender.wakuext_service.rpc_request(
            method="activityCenterNotifications", params=[{"cursor": "", "limit": 20, "activityTypes": [5]}]
        ).json()["result"]
        assert result["notifications"][0]["read"] is True

        response = self.sender.wakuext_service.rpc_request(
            method="markActivityCenterNotificationsUnread",
            params=[
                [
                    message_id,
                ],
            ],
        ).json()
        self.sender.verify_json_schema(response, method="wakuext_markActivityCenterNotificationsUnread")
        assert all(
            (
                response["result"]["activityCenterState"]["hasSeen"] is False,
                response["result"]["activityCenterNotifications"][0]["id"] == message_id,
                response["result"]["activityCenterNotifications"][0]["read"] is False,
            )
        )

        result = self.sender.wakuext_service.rpc_request(
            method="activityCenterNotifications", params=[{"cursor": "", "limit": 20, "activityTypes": [5]}]
        ).json()["result"]
        assert result["notifications"][0]["read"] is False

    def test_accept_activity_center_notifications(self):
        message_id = self.send_contact_request_and_wait_for_signal_to_be_received()
        response = self.receiver.wakuext_service.rpc_request(
            method="acceptActivityCenterNotifications",
            params=[
                [
                    message_id,
                ],
            ],
        ).json()
        self.receiver.verify_json_schema(response, method="wakuext_acceptActivityCenterNotifications")

        result = self.receiver.wakuext_service.rpc_request(
            method="activityCenterNotifications", params=[{"cursor": "", "limit": 20, "activityTypes": [1, 2, 3, 4, 5, 7, 8, 9, 10, 23, 24]}]
        ).json()["result"]
        assert all((result["notifications"][0]["accepted"] is True, result["notifications"][0]["id"] == message_id))

    def test_dismiss_activity_center_notifications(self):
        message_id = self.send_contact_request_and_wait_for_signal_to_be_received()
        response = self.receiver.wakuext_service.rpc_request(
            method="dismissActivityCenterNotifications",
            params=[
                [
                    message_id,
                ],
            ],
        ).json()
        self.receiver.verify_json_schema(response, method="wakuext_dismissActivityCenterNotifications")

        result = self.receiver.wakuext_service.rpc_request(
            method="activityCenterNotifications", params=[{"cursor": "", "limit": 20, "activityTypes": [1, 2, 3, 4, 5, 7, 8, 9, 10, 23, 24]}]
        ).json()["result"]
        assert all((result["notifications"][0]["dismissed"] is True, result["notifications"][0]["id"] == message_id))

    def test_delete_activity_center_notifications(self):
        message_id = self.make_contacts()
        result = self.sender.wakuext_service.rpc_request(
            method="activityCenterNotifications", params=[{"cursor": "", "limit": 20, "activityTypes": [1, 2, 3, 4, 5, 7, 8, 9, 10, 23, 24]}]
        ).json()["result"]
        assert len(result["notifications"]) == 1
        response = self.sender.wakuext_service.rpc_request(
            method="deleteActivityCenterNotifications",
            params=[
                [
                    message_id,
                ],
            ],
        ).json()
        self.sender.verify_json_schema(response, method="wakuext_deleteActivityCenterNotifications")
        result = self.sender.wakuext_service.rpc_request(
            method="activityCenterNotifications", params=[{"cursor": "", "limit": 20, "activityTypes": [1, 2, 3, 4, 5, 7, 8, 9, 10, 23, 24]}]
        ).json()["result"]
        assert not result["notifications"]
