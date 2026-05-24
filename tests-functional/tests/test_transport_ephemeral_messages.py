import logging
from time import sleep
from uuid import uuid4

import pytest

from clients.signals import SignalType
from steps import messenger

logger = logging.getLogger(__name__)

# protobuf.StatusUpdate StatusType values (see protocol/protobuf/status_update.proto).
# Only AUTOMATIC status updates are published as ephemeral Waku messages, so they
# are never persisted by the store node.
STATUS_AUTOMATIC = 1  # ephemeral
STATUS_DO_NOT_DISTURB = 2  # persisted


@pytest.mark.rpc
@pytest.mark.wakuext
class TestEphemeralMessages:
    """Regression coverage for ephemeral messages (issue #7472).

    Ephemeral messages are published with the Waku ephemeral flag and are not
    stored by the store node. A receiver that was offline when an ephemeral
    message was sent must never receive it, even though a regular (persisted)
    message sent in the same window is backfilled on reconnect.
    """

    @pytest.fixture()
    def sender(self, backend_new_profile):
        return backend_new_profile("sender")

    @pytest.fixture()
    def receiver(self, backend_new_profile):
        return backend_new_profile("receiver")

    def test_ephemeral_status_not_backfilled_while_persisted_message_is(self, sender, receiver):
        messenger.make_contacts(sender, receiver)

        ephemeral_text = f"ephemeral_{uuid4()}"
        control_text = f"persisted_{uuid4()}"

        # Receiver is offline while both an ephemeral status update and a normal
        # 1:1 message are published.
        with messenger.node_pause(receiver):
            sender.wakuext_service.set_user_status(STATUS_AUTOMATIC, ephemeral_text)
            response = sender.wakuext_service.send_one_to_one_message(receiver.public_key, control_text)
            control_id = response.get("messages", [])[0].get("id", "")
            assert control_id, "Sender did not get a control message id back"
            sleep(10)

        receiver.wait_for_online(timeout=60)

        # The persisted control message must be delivered once the receiver is
        # back online, proving it has reconnected and caught up.
        with receiver.expect_signal(SignalType.MESSAGES_NEW, pattern=control_id, timeout=120, start="beginning"):
            pass

        # Grace period so any (incorrectly) persisted ephemeral message would
        # have had time to arrive as well.
        sleep(10)

        status_updates = receiver.wakuext_service.status_updates().get("statusUpdates", []) or []
        ephemeral_texts = [s.get("text") for s in status_updates]
        assert ephemeral_text not in ephemeral_texts, "Ephemeral status update was unexpectedly delivered to an offline receiver"

    def test_ephemeral_status_delivered_live_when_online(self, sender, receiver):
        # Sanity check: while the receiver is online, the ephemeral status is
        # relayed in real time. Ephemerality only affects store persistence.
        messenger.make_contacts(sender, receiver)

        ephemeral_text = f"ephemeral_live_{uuid4()}"
        with receiver.expect_signal(SignalType.MESSAGES_NEW, pattern=ephemeral_text, timeout=60, start="beginning"):
            sender.wakuext_service.set_user_status(STATUS_AUTOMATIC, ephemeral_text)

        status_updates = receiver.wakuext_service.status_updates().get("statusUpdates", []) or []
        matching = [s for s in status_updates if s.get("text") == ephemeral_text]
        assert matching, "Ephemeral status update was not relayed to an online receiver"
