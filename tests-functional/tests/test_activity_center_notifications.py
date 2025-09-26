from typing import Union

import pytest

from clients.status_backend import StatusBackend
from steps.messenger import MessengerSteps
from clients.services.wakuext import ActivityCenterNotificationType


def _get_activity_center_notifications(
    backend_instance: StatusBackend,
    activity_types: list = [activity for activity in ActivityCenterNotificationType],
    read_type: Union[int, None] = None,
):
    params = {"cursor": "", "limit": 20, "activity_types": activity_types}
    if read_type:
        params["read_type"] = read_type
    return backend_instance.wakuext_service.get_activity_center_notifications(**params)


@pytest.mark.rpc
class TestActivityCenterNotifications(MessengerSteps):

    @pytest.fixture(autouse=True)
    def setup_backends(self, backend_new_profile):
        """Initialize two unprivileged backends (sender and receiver) for each test function"""
        self.sender = backend_new_profile("sender")
        self.receiver = backend_new_profile("receiver")

    def test_activity_center_notifications(self):
        message_id = self.send_contact_request_and_wait_for_signal_to_be_received(self.sender, self.receiver)
        response = _get_activity_center_notifications(
            backend_instance=self.receiver, activity_types=[ActivityCenterNotificationType.NOTIFICATION_TYPE_CONTACT_REQUEST], read_type=2
        )
        notification = response["notifications"][0]
        assert all(
            (
                notification["accepted"] is False,
                notification["author"] == self.sender.public_key,
                notification["chatId"] == self.sender.public_key,
                notification["id"] == message_id,
                notification["read"] is False,
                notification["lastMessage"]["contactRequestState"] == 1,
            )
        )

        self.accept_contact_request_and_wait_for_signal_to_be_received(message_id)
        response = _get_activity_center_notifications(backend_instance=self.sender)
        notification = response["notifications"][0]
        assert all(
            (
                notification["accepted"] is True,
                notification["author"] == self.sender.public_key,
                notification["chatId"] == self.receiver.public_key,
                notification["id"] == message_id,
                notification["read"] is True,
                notification["message"]["contactRequestState"] == 2,
                notification["lastMessage"]["text"] == f"@{self.receiver.public_key} accepted your contact request",
            )
        )

    def test_activity_center_notifications_count(self):
        self.send_contact_request_and_wait_for_signal_to_be_received(self.sender, self.receiver)
        response = self.receiver.wakuext_service.activity_center_notifications_count([1, 2, 3, 4, 5, 7, 8, 9, 10, 23, 24], 2)
        assert response["5"] == 1

    def test_seen_unseen_activity_center_notifications(self):
        self.send_contact_request_and_wait_for_signal_to_be_received(self.sender, self.receiver)
        response = self.receiver.wakuext_service.has_unseen_activity_center_notifications()
        assert response is True

        response = self.receiver.wakuext_service.mark_as_seen_activity_center_notifications()
        assert response.get("activityCenterState").get("hasSeen") is True
        assert response.get("discordOldestMessageTimestamp") == 0

        response = self.receiver.wakuext_service.has_unseen_activity_center_notifications()
        assert response is False

    def test_get_activity_center_state(self):
        self.send_contact_request_and_wait_for_signal_to_be_received(self.sender, self.receiver)
        response = self.receiver.wakuext_service.get_activity_center_state()
        assert response["hasSeen"] is False

        self.receiver.wakuext_service.mark_as_seen_activity_center_notifications()

        response = self.receiver.wakuext_service.get_activity_center_state()
        assert response["hasSeen"] is True

    def test_mark_all_activity_center_notifications_read(self):
        message_id = self.send_contact_request_and_wait_for_signal_to_be_received(self.sender, self.receiver)
        response = self.receiver.wakuext_service.mark_all_activity_center_notifications_read()
        assert all(
            (
                response["activityCenterState"]["hasSeen"] is True,
                response["activityCenterNotifications"][0]["id"] == message_id,
                response["activityCenterNotifications"][0]["read"] is True,
            )
        )

        response = self.receiver.wakuext_service.has_unseen_activity_center_notifications()
        assert response is False

    def test_mark_activity_center_notifications_read_unread(self):
        message_id = self.make_contacts(self.sender, self.receiver)
        response = self.sender.wakuext_service.mark_activity_center_notifications_read([message_id])
        assert all(
            (
                response["activityCenterNotifications"][0]["id"] == message_id,
                response["activityCenterNotifications"][0]["read"] is True,
            )
        )

        result = _get_activity_center_notifications(
            backend_instance=self.sender, activity_types=[ActivityCenterNotificationType.NOTIFICATION_TYPE_CONTACT_REQUEST]
        )
        assert result["notifications"][0]["read"] is True

        response = self.sender.wakuext_service.mark_activity_center_notifications_unread([message_id])
        assert all(
            (
                response["activityCenterState"]["hasSeen"] is False,
                response["activityCenterNotifications"][0]["id"] == message_id,
                response["activityCenterNotifications"][0]["read"] is False,
            )
        )

        result = _get_activity_center_notifications(
            backend_instance=self.sender, activity_types=[ActivityCenterNotificationType.NOTIFICATION_TYPE_CONTACT_REQUEST]
        )
        assert result["notifications"][0]["read"] is False

    def test_accept_activity_center_notifications(self):
        message_id = self.send_contact_request_and_wait_for_signal_to_be_received(self.sender, self.receiver)
        result = self.receiver.wakuext_service.accept_activity_center_notifications([message_id])
        assert result.get("activityCenterState").get("hasSeen") is True

        result = _get_activity_center_notifications(backend_instance=self.receiver)
        assert all((result["notifications"][0]["accepted"] is True, result["notifications"][0]["id"] == message_id))

    def test_dismiss_activity_center_notifications(self):
        message_id = self.send_contact_request_and_wait_for_signal_to_be_received(self.sender, self.receiver)
        result = self.receiver.wakuext_service.dismiss_activity_center_notifications([message_id])
        assert result is None

        result = _get_activity_center_notifications(backend_instance=self.receiver)
        assert all((result["notifications"][0]["dismissed"] is True, result["notifications"][0]["id"] == message_id))

    def test_delete_activity_center_notifications(self):
        message_id = self.make_contacts(self.sender, self.receiver)
        result = _get_activity_center_notifications(backend_instance=self.sender)
        assert len(result["notifications"]) == 1
        result = self.sender.wakuext_service.delete_activity_center_notifications([message_id])
        assert result is None

        result = _get_activity_center_notifications(backend_instance=self.sender)
        assert not result["notifications"]
