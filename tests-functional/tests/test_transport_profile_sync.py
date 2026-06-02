import logging
import time
from uuid import uuid4

import pytest

from clients.signals import SignalType
from resources.enums import StatusType
from steps import messenger

logger = logging.getLogger(__name__)


@pytest.mark.rpc
@pytest.mark.wakuext
class TestProfileSync:
    """Regression coverage for profile sync over the network (issue #7472).

    Display-name changes (advertised via the contact code) and user status
    updates must propagate from a sender to its contacts over the transport.
    """

    @pytest.fixture()
    def sender(self, backend_new_profile):
        return backend_new_profile("sender")

    @pytest.fixture()
    def receiver(self, backend_new_profile):
        return backend_new_profile("receiver")

    def test_display_name_propagates_to_contact(self, sender, receiver):
        messenger.make_contacts(sender, receiver)

        new_display_name = f"synced{uuid4().hex[:8]}"
        sender.wakuext_service.set_display_name(new_display_name)
        assert sender.settings_service.get_settings().get("display-name") == new_display_name

        # The new display name is advertised via the contact code; the contact
        # must pick it up over the network.
        self._wait_for_contact_display_name(receiver, sender.public_key, new_display_name)

    def test_status_update_propagates_to_contact(self, sender, receiver):
        messenger.make_contacts(sender, receiver)

        status_type = StatusType.DO_NOT_DISTURB  # a persisted (non-ephemeral) status
        status_text = f"status_{uuid4()}"

        with receiver.expect_signal(SignalType.MESSAGES_NEW, pattern=status_text, timeout=60, start="beginning"):
            sender.wakuext_service.set_user_status(status_type.value, status_text)

        status_updates = receiver.wakuext_service.status_updates().get("statusUpdates", [])
        matching = [s for s in status_updates if s.get("text") == status_text]
        assert matching, f"Receiver did not see status update with text '{status_text}'"
        assert matching[0].get("statusType") == status_type.value

    def _wait_for_contact_display_name(self, node, contact_id, expected_name, timeout=120):
        deadline = time.time() + timeout
        last_name = None
        while time.time() < deadline:
            contact = node.wakuext_service.get_contact_by_id(contact_id)
            last_name = (contact or {}).get("displayName")
            if last_name == expected_name:
                return
            time.sleep(2)
        raise AssertionError(f"Contact display name was '{last_name}', expected '{expected_name}'")
