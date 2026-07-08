"""Community control-node transfer across an owner's paired devices — github.com/status-im/status-go/issues/7132."""

import logging

import pytest

from clients.services.wakuext import CommunityPermissionsAccess
from clients.signals import LocalPairingEventAction, LocalPairingEventType, SignalType
from steps import community_control_node, community_token_deploy, community_tokens, messenger
from steps.community_token_deploy import CommunityTokenDeployState
from utils import fake

logger = logging.getLogger(__name__)


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
    backend.wait_for_online(timeout=online_timeout)


@pytest.mark.rpc
@pytest.mark.usefixtures("community_token_deploy_context")
class TestCommunityControlNodeTransfer:
    snt_address: str
    snt_controller_address: str
    community_token_deployer: str
    deploy_state: CommunityTokenDeployState

    @pytest.mark.flaky(reruns=0)  # heavy e2e test (~25 min/run): reruns would multiply the whole run past the job timeout
    def test_control_node_transfer_across_devices(
        self,
        owner_backend,
        member_backend,
        backend_factory,
        anvil_client,
    ):
        # Owner account on two paired devices; the member is separate.
        owner_device_1 = owner_backend
        member = member_backend
        owner_key_uid = owner_device_1.key_uid
        owner_password = owner_device_1.password

        # Given the Owner on device 1 has created a community
        community_resp = owner_device_1.wakuext_service.create_community(
            name=fake.community_name(),
            description=fake.community_description(),
            membership=CommunityPermissionsAccess.MANUAL_ACCEPT,
        )
        community_id = community_resp.get("communities", [{}])[0].get("id")
        assert community_id, "Community was not created"

        # And the member has joined the community
        assert messenger.spectate_and_fetch_community(member, community_id), "Community not found for the member"
        messenger.join_community(member, owner_device_1, community_id)

        # The community needs an owner token for control-node transfer to be allowed (HasTokenOwnership).
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

        # And the Owner has logged into device 2 (the same account, paired second device)
        owner_device_2 = backend_factory("owner_device_2")
        owner_device_2.init_status_backend()
        _pair_second_device(owner_device_1, owner_device_2)
        _login_device(owner_device_2, owner_key_uid, owner_password)

        # When the Owner transfers control from device 1 to device 2
        community_control_node.promote_to_control_node(owner_device_2, community_id, attempts=60)
        community_control_node.wait_until_local_control_node_state(owner_device_2, community_id, expected=True, attempts=60)
        community_control_node.wait_until_local_control_node_state(owner_device_1, community_id, expected=False, attempts=180)

        # And device 2 goes offline (wait for node.stopped so device 1's edit can't race it).
        with owner_device_2.expect_signal(SignalType.NODE_STOPPED):
            owner_device_2.logout()

        # When the Owner on device 1 edits the community
        # Then the member sees the updated community
        name_from_device_1, description_from_device_1 = community_tokens.edit_community_and_wait_until_observer_sees_update(
            owner_device_1, member, community_id, attempts=60, wait_for_message_signal=False
        )

        # When device 2 comes online
        _login_device(owner_device_2, owner_key_uid, owner_password, online_timeout=120)

        # Then the Owner on device 2 sees the updated community
        community_tokens.wait_until_member_sees_community_update(
            owner_device_2, community_id, name_from_device_1, description_from_device_1, attempts=120, spectate=True
        )

        # When the Owner on device 2 edits the community
        # Then the member and the Owner on device 1 see the updated community
        name_from_device_2, description_from_device_2 = community_tokens.edit_community_and_wait_until_observer_sees_update(
            owner_device_2, member, community_id, attempts=60, wait_for_message_signal=False
        )
        community_tokens.wait_until_member_sees_community_update(
            owner_device_1, community_id, name_from_device_2, description_from_device_2, attempts=60, spectate=True
        )
