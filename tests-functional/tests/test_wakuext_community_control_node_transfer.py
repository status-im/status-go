"""Community control-node transfer across an owner's paired devices — github.com/status-im/status-go/issues/7132."""

import logging
import os
import time
from contextlib import contextmanager

import pytest

from clients.services.wakuext import CommunityPermissionsAccess
from clients.signals import LocalPairingEventAction, LocalPairingEventType, SignalType
from steps import community_control_node, community_token_deploy, community_tokens, messenger
from steps.community_token_deploy import CommunityTokenDeployState
from utils import fake
from utils.config import Config

logger = logging.getLogger(__name__)


def _ci_marker(message):
    """Append+flush a marker to an archived log file (and stderr) so it survives xdist buffering and a
    CI kill — the last marker pinpoints a hung step, which pytest/xdist wouldn't forward until test end."""
    line = f"[CONTROL_NODE_TRANSFER] {time.strftime('%H:%M:%S')} {message}\n"
    try:
        os.write(2, line.encode())
    except OSError:
        pass
    if Config.logs_dir:
        try:
            with open(os.path.join(Config.logs_dir, "control_node_transfer_markers.log"), "a", encoding="utf-8") as marker_file:
                marker_file.write(line)
                marker_file.flush()
        except OSError:
            pass


@contextmanager
def _step(name):
    """Emit durable start/end markers (with elapsed) around a step."""
    start = time.monotonic()
    _ci_marker(f"STEP START: {name}")
    try:
        yield
    finally:
        _ci_marker(f"STEP END: {name} (elapsed={time.monotonic() - start:.1f}s)")


def _pairing_predicate(action_value, type_value):
    return lambda signal: (signal.get("event", {}).get("action") == action_value and signal.get("event", {}).get("type") == type_value)


def _pair_second_device(primary, secondary, timeout=120):
    """Bootstrap *secondary* as another device of *primary*, syncing the account and messages."""
    connection_string = primary.get_connection_string_for_bootstrapping_another_device(message_sync_enabled=True)
    with primary.expect_signal(
        SignalType.LOCAL_PAIRING,
        predicate=_pairing_predicate(LocalPairingEventAction.ACTION_PAIRING_INSTALLATION.value, LocalPairingEventType.EVENT_PROCESS_SUCCESS.value),
        timeout=timeout,
    ), secondary.expect_signal(
        SignalType.LOCAL_PAIRING,
        predicate=_pairing_predicate(LocalPairingEventAction.ACTION_PAIRING_INSTALLATION.value, LocalPairingEventType.EVENT_TRANSFER_SUCCESS.value),
        timeout=timeout,
    ):
        response = secondary.input_connection_string_for_bootstrapping(connection_string)
        assert response["error"] is None, f"Pairing failed: {response}"
        assert response["keyUID"] == primary.key_uid


def _login_device(backend, key_uid, password, online_timeout=60):
    backend.login(key_uid, password)
    backend.wait_for_login()
    backend.wait_for_wakuext_ready(timeout=30)
    # LoginAccount resets networks to defaults (#6010/#5597); re-add Anvil so token-gated events aren't dropped.
    backend.add_anvil_network()
    backend.wait_for_online(timeout=online_timeout)


@pytest.mark.rpc
@pytest.mark.usefixtures("community_token_deploy_context")
class TestCommunityControlNodeTransfer:
    snt_address: str
    snt_controller_address: str
    community_token_deployer: str
    deploy_state: CommunityTokenDeployState

    def _promote_device_2_to_control_node(self, owner_device_1, member, owner_device_2, anvil_client):
        """Token-gated community on device 1, with device 2 paired and promoted to control node."""
        community_resp = owner_device_1.wakuext_service.create_community(
            name=fake.community_name(),
            description=fake.community_description(),
            membership=CommunityPermissionsAccess.MANUAL_ACCEPT,
        )
        community_id = community_resp.get("communities", [{}])[0].get("id")
        assert community_id, "Community was not created"

        assert messenger.spectate_and_fetch_community(member, community_id), "Community not found for the member"
        messenger.join_community(member, owner_device_1, community_id)

        # Control-node transfer requires the community to own a token (HasTokenOwnership).
        community_tokens.fund_native_balance(owner_device_1, anvil_client)
        owner_device_1.wallet_service.restart_wallet_reload_timer()
        tokens = community_token_deploy.deploy_owner_and_master_tokens(
            owner_device_1,
            community_id,
            self.community_token_deployer,
            self.deploy_state,
            anvil_client=anvil_client,
            wait_for_deploy_status_signal=False,
        )
        assert tokens.owner_token_address, "Owner token was not deployed"
        community_control_node.publish_owner_token_to_community(owner_device_1, community_id, tokens.owner_token_address)

        owner_device_2.init_status_backend()
        _pair_second_device(owner_device_1, owner_device_2)
        _login_device(owner_device_2, owner_device_1.key_uid, owner_device_1.password)

        community_control_node.promote_to_control_node(owner_device_2, community_id, attempts=60)
        community_control_node.wait_until_local_control_node_state(owner_device_2, community_id, expected=True, attempts=60)
        # Device 1 relinquishing control is a slow, variable cross-node sync in CI; give it headroom.
        community_control_node.wait_until_local_control_node_state(owner_device_1, community_id, expected=False, attempts=120, delay=3)

        return community_id

    @pytest.mark.xfail(
        reason="Device B loses the Anvil network on LoginAccount and drops the token-gated community event; "
        "see https://github.com/status-im/status-go/issues/7615",
        run=False,
    )
    def test_control_node_transfer_across_devices(self, owner_backend, member_backend, backend_factory, anvil_client):
        owner_device_1 = owner_backend
        member = member_backend
        owner_device_2 = backend_factory("owner_device_2")
        community_id = self._promote_device_2_to_control_node(owner_device_1, member, owner_device_2, anvil_client)

        # Wait for node.stopped so device 1's edit can't race the shutdown.
        with owner_device_2.expect_signal(SignalType.NODE_STOPPED):
            owner_device_2.logout()

        name_from_device_1, description_from_device_1 = community_tokens.edit_community_and_wait_until_observer_sees_update(
            owner_device_1, member, community_id, attempts=60, wait_for_message_signal=False
        )

        _login_device(owner_device_2, owner_device_1.key_uid, owner_device_1.password, online_timeout=120)

        community_tokens.wait_until_member_sees_community_update(
            owner_device_2, community_id, name_from_device_1, description_from_device_1, attempts=60, spectate=True
        )

        name_from_device_2, description_from_device_2 = community_tokens.edit_community_and_wait_until_observer_sees_update(
            owner_device_2, member, community_id, attempts=60, wait_for_message_signal=False
        )
        community_tokens.wait_until_member_sees_community_update(
            owner_device_1, community_id, name_from_device_2, description_from_device_2, attempts=60, spectate=True
        )

    # run=False: offline store-catch-up is unreliable under shared-fleet contention (device 2 can't
    # retrieve device 1's edit among the cross-test flood) — needs an infra fix; kept ready to enable.
    @pytest.mark.xfail(
        reason="offline store-catch-up unreliable under shared-fleet store contention; pending infra fix",
        run=False,
    )
    def test_control_node_transfer_across_devices_with_container_pause(self, owner_backend, member_backend, backend_factory, anvil_client):
        # Pausing device B's container (vs logout + LoginAccount) keeps its Anvil network, avoiding #7615.
        _ci_marker("pause test: body entered (fixtures complete)")
        owner_device_1 = owner_backend
        member = member_backend
        owner_device_2 = backend_factory("owner_device_2")

        with _step("setup: token-gated community + promote device 2 to control node"):
            community_id = self._promote_device_2_to_control_node(owner_device_1, member, owner_device_2, anvil_client)

        with _step("pause device 2 container"):
            owner_device_2.container_pause()  # synchronous, so device 1's edit can't race it

        with _step("device 1 edits community and member sees update"):
            name_from_device_1, description_from_device_1 = community_tokens.edit_community_and_wait_until_observer_sees_update(
                owner_device_1, member, community_id, attempts=60, wait_for_message_signal=False
            )

        with _step("unpause device 2 container"):
            owner_device_2.container_unpause()

        with _step("device 2 becomes online after unpause"):
            owner_device_2.wait_for_online(timeout=120)

        with _step("device 2 sees device 1 community update after unpause"):
            # fetch_live=False: read device 2's local DB (non-blocking) instead of a live fetch that hangs under contention.
            community_tokens.wait_until_member_sees_community_update(
                owner_device_2, community_id, name_from_device_1, description_from_device_1, attempts=120, delay=3, spectate=True, fetch_live=False
            )

        with _step("device 2 is still local control node after unpause"):
            community_control_node.wait_until_local_control_node_state(owner_device_2, community_id, expected=True, attempts=30, delay=2)

        # Split device 2's edit: local-apply failure ⇒ not really control node; member not seeing it ⇒ send path stalled.
        with _step("device 2 edit RPC returns and local state updates"):
            name_from_device_2, description_from_device_2 = messenger.edit_community(owner_device_2, community_id)
            assert community_tokens.check_member_community_updated(
                owner_device_2, community_id, name_from_device_2, description_from_device_2, include_live_fetch=False
            ), "device 2 did not apply its own edit locally"

        with _step("member sees device 2 community update"):
            community_tokens.wait_until_member_sees_community_update(
                member, community_id, name_from_device_2, description_from_device_2, attempts=120, delay=3, spectate=True, fetch_live=False
            )

        with _step("device 1 sees device 2 community update"):
            community_tokens.wait_until_member_sees_community_update(
                owner_device_1, community_id, name_from_device_2, description_from_device_2, attempts=120, delay=3, spectate=True, fetch_live=False
            )

        _ci_marker("pause test: body complete")
