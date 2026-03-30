import logging
import time
import uuid
import json
import re
from typing import Optional, Tuple, Any


import pytest
from web3 import Web3

from clients.api import ApiResponseError
from clients.services.wakuext import (
    ActivityCenterNotificationType,
    CommunityPermissionsAccess,
    CommunityTokenPermissionType,
    CommunityTokenPrivilegesLevel,
    CommunityTokenType,
    CommunityRoles,
)
from clients.signals import SignalType
from clients.status_backend import StatusBackend
from steps import community_tokens, messenger
from utils import fake
from utils.keys import change_community_key_compression

logger = logging.getLogger(__name__)

COMMUNITY_DEPLOY_OWNER_TOKEN = 12
COMMUNITY_MINT_TOKENS = 13
NATIVE_TOKEN_ADDRESS = "0x0000000000000000000000000000000000000000"


@pytest.mark.rpc
class TestCommunityTokenPermissions:
    @pytest.fixture(autouse=True)
    def setup_tokens(self, snt_addresses):
        self.snt_address = snt_addresses["snt"]
        self.snt_controller_address = snt_addresses["controller"]

    @pytest.fixture(autouse=True)
    def get_communities_contracts(self, communities_addresses):
        self.community_token_deployer = next(
            info["value"] for info in communities_addresses.values() if info["internal_type"] == "contract CommunityTokenDeployer"
        )

    @staticmethod
    def _mint_tx_confirmed_predicate(signal):
        """Predicate for expect_signal: matches a successful community mint transaction."""
        event = signal.get("event", {})
        return event.get("sendType") == COMMUNITY_MINT_TOKENS and event.get("success") is True

    def check_member_community_updated(self, member_backend, community_id: str, expected_name: str, expected_description: str) -> bool:
        member_communities = member_backend.wakuext_service.communities()
        member_community = next((c for c in messenger.communities_list(member_communities) if c.get("id") == community_id), None)
        return bool(
            member_community and member_community.get("name") == expected_name and member_community.get("description") == expected_description
        )

    def temporary_master_contract_address(self, tx_hash: str) -> str:
        return f"{tx_hash}-master"

    def temporary_owner_contract_address(self, tx_hash: str) -> str:
        return f"{tx_hash}-owner"

    def _extract_master_token_address_from_event(self, event: dict) -> Optional[str]:
        candidates = [
            (event.get("masterToken") or {}).get("address"),
            event.get("masterTokenAddress"),
            event.get("masterTokenContractAddress"),
            ((event.get("tokenData") or {}).get("masterToken") or {}).get("address"),
            (event.get("tokenData") or {}).get("masterTokenAddress"),
            ((event.get("token") or {}).get("masterToken") or {}).get("address"),
            (event.get("token") or {}).get("masterTokenAddress"),
        ]
        for address in candidates:
            if isinstance(address, str) and re.fullmatch(r"0x[a-fA-F0-9]{40}", address):
                return address
        return None

    def resolve_master_token_address_from_receipt(self, anvil_client, tx_hash: Optional[str]) -> Optional[str]:
        """Resolve deployed master token address directly from deploy tx receipt logs."""
        _, master_token_address = self.resolve_owner_and_master_token_addresses_from_receipt(anvil_client, tx_hash)
        return master_token_address

    def resolve_owner_and_master_token_addresses_from_receipt(self, anvil_client, tx_hash: Optional[str]) -> Tuple[Optional[str], Optional[str]]:
        """Resolve deployed owner/master token addresses directly from deploy tx receipt logs."""
        if not anvil_client:
            return None, None
        if not isinstance(tx_hash, str) or re.fullmatch(r"0x[a-fA-F0-9]{64}", tx_hash) is None:
            return None, None

        try:
            receipt = anvil_client.transaction_receipt(tx_hash)
        except Exception as exc:
            logger.warning(f"Failed to fetch tx receipt for {tx_hash}: {exc}")
            return None, None

        if not receipt:
            return None, None

        status = receipt.get("status")
        if status not in (1, "0x1", True):
            logger.warning(f"Receipt for tx {tx_hash} is not successful yet (status={status})")
            return None, None

        expected_owner_topic0 = Web3.keccak(text="DeployOwnerToken(address)").hex().lower()
        expected_master_topic0 = Web3.keccak(text="DeployMasterToken(address)").hex().lower()
        expected_emitter = str(self.community_token_deployer).lower()

        owner_token_address = None
        master_token_address = None

        for v_log in receipt.get("logs", []) or []:
            topics = v_log.get("topics", []) or []
            if not topics:
                continue

            topic0 = topics[0].hex() if hasattr(topics[0], "hex") else str(topics[0])
            topic0_lower = str(topic0).lower()
            if topic0_lower not in (expected_owner_topic0, expected_master_topic0):
                continue

            emitter = str(v_log.get("address", "")).lower()
            if expected_emitter and emitter and emitter != expected_emitter:
                continue

            if len(topics) < 2:
                continue

            topic1 = topics[1].hex() if hasattr(topics[1], "hex") else str(topics[1])
            topic1_no_prefix = topic1[2:] if topic1.startswith("0x") else topic1
            if not re.fullmatch(r"[a-fA-F0-9]{64}", topic1_no_prefix):
                continue

            resolved_address = f"0x{topic1_no_prefix[-40:]}"
            if re.fullmatch(r"0x[a-fA-F0-9]{40}", resolved_address):
                checksum_address = Web3.to_checksum_address(resolved_address)
                if topic0_lower == expected_owner_topic0:
                    owner_token_address = checksum_address
                elif topic0_lower == expected_master_topic0:
                    master_token_address = checksum_address

        if owner_token_address is None:
            logger.warning(f"DeployOwnerToken event not found in receipt logs for tx {tx_hash}")
        if master_token_address is None:
            logger.warning(f"DeployMasterToken event not found in receipt logs for tx {tx_hash}")

        return owner_token_address, master_token_address

    def deploy_owner_token(self, owner_backend, community_id, anvil_client=None, wait_for_deploy_status_signal: bool = True):
        """Deploy and mint owner token for the community"""

        accounts = owner_backend.accounts_service.get_accounts()
        assert accounts, "No accounts found"
        wallet_account = next(a for a in accounts if not a.get("chat"))
        assert wallet_account, "No wallet account found"
        address_from = wallet_account["address"]

        # CommunityTokenDeployer contract address
        contract_address = self.community_token_deployer

        # Generate deployment signature
        chain_id = owner_backend.network_id
        signature = owner_backend.wakuext_service.create_community_token_deployment_signature(chain_id, address_from, community_id)
        logger.info(f"Community token deployment signature: {signature}")

        token_uri = change_community_key_compression(community_id) + "/"

        # Owner and master token parameters for deployment
        owner_token_parameters = {
            "name": "TestOwnerToken",
            "symbol": "TOT",
            "tokenUri": token_uri,
            "ownerTokenAddress": NATIVE_TOKEN_ADDRESS,
            "masterTokenAddress": NATIVE_TOKEN_ADDRESS,
            "description": "",
            "communityId": community_id,
            "supply": "1",
            "infiniteSupply": False,
            "decimals": 0,
            "transferable": True,
            "remoteSelfDestruct": False,
            "tokenType": CommunityTokenType.ERC721.value,
            "base64image": "data:image/gif;base64,R0lGODlhAQABAIAAAP///wAAACwAAAAAAQABAAACAkQBADs=",
        }
        master_token_parameters = {
            "name": "TestMasterToken",
            "symbol": "TMT",
            "tokenUri": token_uri,
            "ownerTokenAddress": NATIVE_TOKEN_ADDRESS,
            "masterTokenAddress": NATIVE_TOKEN_ADDRESS,
            "description": "",
            "communityId": community_id,
            "supply": "0",
            "infiniteSupply": True,
            "decimals": 0,
            "transferable": True,
            "remoteSelfDestruct": True,
            "tokenType": CommunityTokenType.ERC721.value,
            "base64image": "data:image/gif;base64,R0lGODlhAQABAIAAAP///wAAACwAAAAAAQABAAACAkQBADs=",
        }

        # Generate UUID for the transaction
        transaction_uuid = str(uuid.uuid4())

        # Fetch balances
        owner_backend.wallet_service.get_balances_at_by_chain([address_from], [f"{chain_id}-{NATIVE_TOKEN_ADDRESS}"])

        # Get suggested routes for deploying tokens
        signer_pub_key = owner_backend.public_key

        routes_result = owner_backend.wallet_service.suggested_community_routes(
            uuid=transaction_uuid,
            send_type=COMMUNITY_DEPLOY_OWNER_TOKEN,
            chain_id=chain_id,
            address_from=address_from,
            addr_to=contract_address,
            community_id=community_id,
            signer_pub_key=signer_pub_key,
            token_ids=[],
            wallet_addresses=[],
            transfer_details=[],
            signature=signature,
            owner_token_parameters=owner_token_parameters,
            master_token_parameters=master_token_parameters,
        )

        assert routes_result.get("Uuid") == transaction_uuid, f"Expected UUID {transaction_uuid}, got {routes_result.get('uuid')}"

        # Build transactions from route (async, sends SignRouterTransactions signal)
        with owner_backend.expect_signal(
            SignalType.WALLET_ROUTER_SIGN_TRANSACTIONS,
            predicate=lambda s: s.get("event", {}).get("sendDetails", {}).get("uuid") == transaction_uuid,
            timeout=60,
        ) as sign_exp:
            owner_backend.wallet_service.build_transactions_from_route(transaction_uuid)

        sign_signal = sign_exp.result

        signing_details = sign_signal["event"]["signingDetails"]
        signatures = {}
        for tx_hash in signing_details["hashes"]:
            sig_hex = owner_backend.wallet_service.sign_message(tx_hash, address_from, owner_backend.password)
            assert sig_hex and sig_hex.startswith("0x"), f"Invalid transaction signature for hash {tx_hash}: {sig_hex}"
            tx_signature = sig_hex[2:]
            signatures[tx_hash] = {
                "r": tx_signature[:64],
                "s": tx_signature[64:128],
                "v": tx_signature[128:],
            }

        # Send the signed transactions and collect tx hashes from route-sent signal first.
        # WALLET pending-tx status can include unrelated successes if not correlated strictly.
        with owner_backend.expect_signal(
            SignalType.WALLET_ROUTER_TRANSACTIONS_SENT,
            predicate=lambda s: s.get("event", {}).get("uuid") == transaction_uuid
            or s.get("event", {}).get("sendDetails", {}).get("uuid") == transaction_uuid,
            timeout=60,
        ) as sent_exp:
            owner_backend.wallet_service.send_router_transactions_with_signatures(transaction_uuid, signatures)

        sent_signal = sent_exp.result
        assert isinstance(sent_signal, dict), f"Unexpected sent signal payload: {sent_signal}"
        logger.debug(f"Sent router transactions for UUID {transaction_uuid}: {sent_signal}")

        def _normalize_tx_hash(value: Optional[str]) -> Optional[str]:
            if not isinstance(value, str) or not value:
                return None
            normalized = value if value.startswith("0x") else f"0x{value}"
            return normalized if re.fullmatch(r"0x[a-fA-F0-9]{64}", normalized) else None

        def _extract_tx_hashes(payload: dict) -> list[str]:
            hashes: list[str] = []

            def _push(candidate: Optional[str]):
                normalized = _normalize_tx_hash(candidate)
                if normalized and normalized not in hashes:
                    hashes.append(normalized)

            event = payload.get("event", {}) if isinstance(payload, dict) else {}
            root = payload if isinstance(payload, dict) else {}

            for container in (event, root):
                _push(container.get("hash"))
                _push(container.get("Hash"))
                _push(container.get("txHash"))
                _push(container.get("TxHash"))
                _push(container.get("transactionHash"))

                for h in container.get("hashes", []) or []:
                    _push(h)

                for key in ("sentTransactions", "transactions", "txs", "items"):
                    for item in container.get(key, []) or []:
                        if not isinstance(item, dict):
                            continue
                        _push(item.get("hash"))
                        _push(item.get("Hash"))
                        _push(item.get("txHash"))
                        _push(item.get("TxHash"))
                        _push(item.get("transactionHash"))

            return hashes

        sent_hashes = _extract_tx_hashes(sent_signal)

        # Mirror desktop flow: register owner/master deployment intent in backend immediately
        # after TX is submitted, before confirmation arrives. This must happen before the
        # wallet pending-tx success signal is received, because that same confirmation event
        # triggers handleDeployOwnerToken in the backend — which looks up the tokens in the DB.
        # Calling storeDeployedOwnerToken only after the success signal causes a race condition
        # where handleDeployOwnerToken runs first and fails to find the tokens.
        _store_deployed_called = False
        early_tx_hash = sent_hashes[0] if sent_hashes else None
        early_is_valid = isinstance(early_tx_hash, str) and re.fullmatch(r"0x[a-fA-F0-9]{64}", early_tx_hash) is not None
        if early_is_valid and isinstance(early_tx_hash, str):
            self._last_deployed_master_token_placeholder = self.temporary_master_contract_address(early_tx_hash)
            self._last_deployed_owner_token_placeholder = self.temporary_owner_contract_address(early_tx_hash)
            try:
                owner_backend.rpc_valid_request(
                    "communitytokens_storeDeployedOwnerToken",
                    [
                        address_from,
                        chain_id,
                        early_tx_hash,
                        owner_token_parameters,
                        master_token_parameters,
                    ],
                )
                _store_deployed_called = True
            except Exception as exc:
                logger.warning(f"communitytokens_storeDeployedOwnerToken (early) failed for tx {early_tx_hash}: {exc}")
        else:
            logger.warning(f"Skipping early communitytokens_storeDeployedOwnerToken: no valid tx hash in sent_hashes={sent_hashes}")

        def _wallet_tx_success(signal: dict) -> bool:
            event = signal.get("event", {})
            if event.get("type") != "pending-transaction-status-changed":
                return False

            try:
                tx_status = json.loads(event.get("message", "{}").replace("'", '"'))
            except Exception:
                return False

            if tx_status.get("status") != "Success":
                return False

            # If we already know route-sent hashes, only accept matching wallet updates.
            if sent_hashes:
                candidate_hashes = _extract_tx_hashes({"event": tx_status})
                return any(h in sent_hashes for h in candidate_hashes)

            return True

        tx_status = {}
        wallet_signal = None
        try:
            with owner_backend.expect_signal(
                SignalType.WALLET,
                accept_fn=_wallet_tx_success,
                timeout=60,
            ) as wallet_exp:
                pass
            wallet_signal = wallet_exp.result
            assert isinstance(wallet_signal, dict), f"Unexpected wallet signal payload: {wallet_signal}"
            tx_status = json.loads(wallet_signal["event"]["message"].replace("'", '"'))
        except Exception as exc:
            logger.warning(f"Did not capture correlated wallet pending-tx success signal for deploy UUID {transaction_uuid}: {exc}")

        # Prefer route-sent hash first, then wallet status fallback.
        tx_hash = sent_hashes[0] if sent_hashes else None
        if not tx_hash:
            wallet_hashes = _extract_tx_hashes({"event": tx_status})
            tx_hash = wallet_hashes[0] if wallet_hashes else None

        is_valid_tx_hash = isinstance(tx_hash, str) and re.fullmatch(r"0x[a-fA-F0-9]{64}", tx_hash) is not None

        if tx_status:
            logger.info(f"Initial owner token deployment tx status {tx_status}")
        logger.info(f"Resolved owner token deployment tx hash {tx_hash}")

        # Fallback: if storeDeployedOwnerToken was not called early (e.g. sent_hashes was empty),
        # call it now with the finalised tx_hash as a last resort.
        if not _store_deployed_called and is_valid_tx_hash and isinstance(tx_hash, str):
            self._last_deployed_master_token_placeholder = self.temporary_master_contract_address(tx_hash)
            self._last_deployed_owner_token_placeholder = self.temporary_owner_contract_address(tx_hash)

            try:
                owner_backend.rpc_valid_request(
                    "communitytokens_storeDeployedOwnerToken",
                    [
                        address_from,
                        chain_id,
                        tx_hash,
                        owner_token_parameters,
                        master_token_parameters,
                    ],
                )
            except Exception as exc:
                logger.warning(f"communitytokens_storeDeployedOwnerToken failed for tx {tx_hash}: {exc}")
            else:
                if wait_for_deploy_status_signal:
                    try:
                        with owner_backend.expect_signal(
                            SignalType.COMMUNITY_TOKEN_TRANSACTION_STATUS_CHANGED,
                            predicate=lambda s: s.get("event", {}).get("sendType") == COMMUNITY_DEPLOY_OWNER_TOKEN
                            and s.get("event", {}).get("success") is True
                            and str(s.get("event", {}).get("hash", "")).lower() == str(tx_hash).lower(),
                            timeout=60,
                            start="beginning",
                        ) as community_token_exp:
                            pass

                        community_token_signal = community_token_exp.result
                        assert isinstance(community_token_signal, dict), f"Unexpected community token signal payload: {community_token_signal}"
                        logger.debug(f"Community token transaction status signal: {community_token_signal}")

                        deploy_event = community_token_signal.get("event", {})
                        logger.info(
                            "[DIAGNOSTICS] deploy status event | "
                            f"tx_hash={tx_hash} success={deploy_event.get('success')} "
                            f"error_string={deploy_event.get('errorString')} "
                            f"event={json.dumps(deploy_event, default=str)}"
                        )

                        master_token_address = self._extract_master_token_address_from_event(community_token_signal.get("event", {}))
                        if master_token_address:
                            self._last_deployed_master_token_address = master_token_address
                        else:
                            logger.info(
                                f"Deploy status signal for tx {tx_hash} did not include master token address; " "continuing with later reconciliation"
                            )
                    except Exception as exc:
                        logger.warning(
                            f"Did not capture deploy status signal for tx {tx_hash}; " f"will reconcile later via signals/receipt/metadata: {exc}"
                        )

                    # Keep fallback path, but prefer authoritative deploy-status signal with address first.
                    if not isinstance(getattr(self, "_last_deployed_master_token_address", None), str):
                        for _ in range(10):
                            deployment_signals = owner_backend.received_signals.get(SignalType.COMMUNITY_TOKEN_TRANSACTION_STATUS_CHANGED, []) or []
                            for signal in reversed(deployment_signals):
                                event = signal.get("event", {})
                                signal_hash = (
                                    str(event.get("hash") or event.get("txHash") or event.get("transactionHash") or event.get("Hash") or "")
                                    .strip()
                                    .lower()
                                )
                                if signal_hash != str(tx_hash).lower():
                                    continue
                                if event.get("sendType") != COMMUNITY_DEPLOY_OWNER_TOKEN or event.get("success") is not True:
                                    continue

                                master_token_address = self._extract_master_token_address_from_event(event)
                                if master_token_address:
                                    self._last_deployed_master_token_address = master_token_address
                                    break

                            if isinstance(getattr(self, "_last_deployed_master_token_address", None), str):
                                break
                            time.sleep(1)
        else:
            logger.warning(f"Skipping communitytokens_storeDeployedOwnerToken due to invalid tx hash: {tx_hash}")

        # Optional concrete chain-level check: wait until tx is mined successfully.
        if anvil_client and tx_hash:
            for _ in range(30):
                try:
                    receipt = anvil_client.transaction_receipt(tx_hash)
                    if receipt and receipt.get("status") == 1:
                        logger.info("Owner token deployment tx minted on chain")
                        break
                except Exception:
                    pass
                time.sleep(1)

        return tx_hash if is_valid_tx_hash else None

    def mint_community_token(
        self,
        sender_backend: StatusBackend,
        community_id: str,
        token_contract_address: str,
        wallet_addresses: list[str],
        token_type: CommunityTokenType,
        privilege_level: int,
        amount: int = 1,
    ):
        address_from = messenger.wallet_address(sender_backend)
        chain_id = sender_backend.network_id
        sender_backend.wallet_service.get_balances_at_by_chain([address_from], [f"{chain_id}-{NATIVE_TOKEN_ADDRESS}"])

        transaction_uuid = str(uuid.uuid4())
        transfer_details = [
            {
                "tokenType": token_type.value,
                "privilegeLevel": privilege_level,
                "tokenContractAddress": token_contract_address,
                "amount": hex(amount),
            }
        ]

        routes_result = sender_backend.wallet_service.suggested_community_routes(
            uuid=transaction_uuid,
            send_type=COMMUNITY_MINT_TOKENS,
            chain_id=chain_id,
            address_from=address_from,
            addr_to=token_contract_address,
            community_id=community_id,
            signer_pub_key=sender_backend.public_key,
            token_ids=[],
            wallet_addresses=wallet_addresses,
            transfer_details=transfer_details,
        )
        assert routes_result.get("Uuid") == transaction_uuid, f"Expected UUID {transaction_uuid}, got {routes_result.get('uuid')}"

        with sender_backend.expect_signal(
            SignalType.WALLET_ROUTER_SIGN_TRANSACTIONS,
            predicate=lambda s: s.get("event", {}).get("sendDetails", {}).get("uuid") == transaction_uuid,
            timeout=60,
        ) as sign_exp:
            sender_backend.wallet_service.build_transactions_from_route(transaction_uuid)

        sign_signal = sign_exp.result
        assert isinstance(sign_signal, dict), f"Unexpected sign signal payload: {sign_signal}"
        signing_details = sign_signal["event"]["signingDetails"]
        signatures = {}
        for tx_hash in signing_details["hashes"]:
            sig_hex = sender_backend.wallet_service.sign_message(tx_hash, address_from, sender_backend.password)
            assert sig_hex and sig_hex.startswith("0x"), f"Invalid transaction signature for hash {tx_hash}: {sig_hex}"
            tx_signature = sig_hex[2:]
            signatures[tx_hash] = {
                "r": tx_signature[:64],
                "s": tx_signature[64:128],
                "v": tx_signature[128:],
            }

        with sender_backend.expect_signal(
            SignalType.WALLET_ROUTER_TRANSACTIONS_SENT,
            timeout=60,
        ) as sent_exp:
            sender_backend.wallet_service.send_router_transactions_with_signatures(transaction_uuid, signatures)

        return sent_exp.result

    def get_community_token_contract_address(
        self, backend: StatusBackend, community_id: str, privileges_level: int, attempts: int = 10, delay: int = 2
    ):
        for _ in range(attempts):
            # Keep subscription active and refresh both network and local-db views.
            try:
                backend.wakuext_service.spectate_community(community_id)
            except Exception:
                pass

            community = messenger.fetch_community(backend, community_id)
            communities_resp = backend.wakuext_service.communities()
            communities_list = messenger.communities_list(communities_resp)
            community_from_list = next((c for c in communities_list if c.get("id") == community_id), None)

            for source in [community, community_from_list]:
                token_metadata = (source or {}).get("communityTokensMetadata") or []

                for metadata in token_metadata:
                    level = metadata.get("privilegesLevel", metadata.get("privilegeLevel"))
                    if level is None or int(level) != int(privileges_level):
                        continue

                    contract_addresses = metadata.get("contractAddresses", {})
                    contract_address = contract_addresses.get(backend.network_id) or contract_addresses.get(str(backend.network_id))
                    if contract_address:
                        return contract_address

            time.sleep(delay)

        return None

    # Protobuf CommunityTokenPermission type values for privileged tokens.
    _PERM_TYPE_BECOME_TOKEN_MASTER = 5
    _PERM_TYPE_BECOME_TOKEN_OWNER = 6

    def _read_backend_token_addresses_by_privilege(self, backend: StatusBackend, community_id: str) -> tuple[list[str], list[str]]:
        owner_addresses: list[str] = []
        master_addresses: list[str] = []

        try:
            backend.wakuext_service.spectate_community(community_id)
        except Exception:
            pass

        community = messenger.fetch_community(backend, community_id)
        communities_resp = backend.wakuext_service.communities()
        communities_list = messenger.communities_list(communities_resp)
        community_from_list = next((c for c in communities_list if c.get("id") == community_id), None)

        def _add_address(contract_address: str, level: int) -> None:
            if int(level) == int(CommunityTokenPrivilegesLevel.OWNER_LEVEL.value):
                if contract_address not in owner_addresses:
                    owner_addresses.append(contract_address)
            elif int(level) == int(CommunityTokenPrivilegesLevel.MASTER_LEVEL.value):
                if contract_address not in master_addresses:
                    master_addresses.append(contract_address)

        for source in [community, community_from_list]:
            if not source:
                continue

            # Primary: read from tokenPermissions which carries the privilege type reliably.
            # tokenPermissions.type 6 = BECOME_TOKEN_OWNER, type 5 = BECOME_TOKEN_MASTER.
            token_permissions = source.get("tokenPermissions") or {}
            for _perm_id, permission in token_permissions.items():
                perm_type = permission.get("type")
                if perm_type not in (self._PERM_TYPE_BECOME_TOKEN_OWNER, self._PERM_TYPE_BECOME_TOKEN_MASTER):
                    continue
                level = (
                    CommunityTokenPrivilegesLevel.OWNER_LEVEL.value
                    if perm_type == self._PERM_TYPE_BECOME_TOKEN_OWNER
                    else CommunityTokenPrivilegesLevel.MASTER_LEVEL.value
                )
                for criteria in permission.get("token_criteria", []) or []:
                    addr_map = criteria.get("contract_addresses") or criteria.get("contractAddresses") or {}
                    addr = addr_map.get(backend.network_id) or addr_map.get(str(backend.network_id))
                    if isinstance(addr, str) and addr:
                        _add_address(addr, level)

            # Fallback: read from communityTokensMetadata for backends that embed a level field.
            token_metadata = source.get("communityTokensMetadata") or []
            for metadata in token_metadata:
                level = metadata.get("privilegesLevel", metadata.get("privilegeLevel"))
                if level is None:
                    continue
                addr_map = metadata.get("contractAddresses") or metadata.get("contract_addresses") or {}
                addr = addr_map.get(backend.network_id) or addr_map.get(str(backend.network_id))
                if isinstance(addr, str) and addr:
                    _add_address(addr, level)

        return owner_addresses, master_addresses

    def wait_for_backend_token_state_transition(
        self,
        backend: StatusBackend,
        community_id: str,
        deploy_tx_hash: str,
        attempts: int = 30,
        delay: int = 2,
    ) -> tuple[str, str, str, str, bool, bool]:
        temp_owner_token_address = self.temporary_owner_contract_address(deploy_tx_hash)
        temp_master_token_address = self.temporary_master_contract_address(deploy_tx_hash)

        saw_temp_owner = False
        saw_temp_master = False
        final_owner_token_address = None
        final_master_token_address = None
        retrack_method = "communitytokens_reTrackOwnerTokenDeploymentTransaction"
        safe_owner_method = "communitytokens_safeGetOwnerTokenAddress"
        chain_id = backend.network_id
        retrack_supported = True

        for attempt in range(1, attempts + 1):
            owner_addresses, master_addresses = self._read_backend_token_addresses_by_privilege(backend, community_id)

            saw_temp_owner = saw_temp_owner or temp_owner_token_address in owner_addresses
            saw_temp_master = saw_temp_master or temp_master_token_address in master_addresses

            for owner_address in owner_addresses:
                if re.fullmatch(r"0x[a-fA-F0-9]{40}", owner_address) and owner_address != temp_owner_token_address:
                    final_owner_token_address = Web3.to_checksum_address(owner_address)
                    break

            for master_address in master_addresses:
                if re.fullmatch(r"0x[a-fA-F0-9]{40}", master_address) and master_address != temp_master_token_address:
                    final_master_token_address = Web3.to_checksum_address(master_address)
                    break

            temp_replaced = temp_owner_token_address not in owner_addresses and temp_master_token_address not in master_addresses
            if final_owner_token_address and final_master_token_address and temp_replaced:
                return (
                    temp_owner_token_address,
                    temp_master_token_address,
                    final_owner_token_address,
                    final_master_token_address,
                    saw_temp_owner,
                    saw_temp_master,
                )

            # Best-effort acceleration: ask backend to reconcile deployment tracking.
            # This must not become the source of truth; refreshed token metadata remains authoritative.
            owner_for_retrack = final_owner_token_address
            if owner_for_retrack is None:
                for owner_address in owner_addresses:
                    if re.fullmatch(r"0x[a-fA-F0-9]{40}", owner_address):
                        owner_for_retrack = Web3.to_checksum_address(owner_address)
                        break

            if owner_for_retrack is None:
                try:
                    safe_owner = backend.rpc_valid_request(safe_owner_method, [chain_id, community_id])
                    if isinstance(safe_owner, str) and re.fullmatch(r"0x[a-fA-F0-9]{40}", safe_owner):
                        owner_for_retrack = Web3.to_checksum_address(safe_owner)
                except Exception as exc:
                    if attempt == 1 or attempt == attempts or attempt % 5 == 0:
                        logger.info(
                            f"[DIAGNOSTICS] {safe_owner_method} failed while waiting token transition " f"(attempt {attempt}/{attempts}): {exc}"
                        )

            if owner_for_retrack is not None and retrack_supported:
                try:
                    backend.rpc_valid_request(retrack_method, [chain_id, owner_for_retrack])
                except Exception as exc:
                    error_text = str(exc)
                    method_missing = ("-32601" in error_text) or ("does not exist/is not available" in error_text.lower())
                    if method_missing:
                        retrack_supported = False
                        logger.info(
                            f"[DIAGNOSTICS] {retrack_method} is unavailable in this backend; " "disabling retrack attempts for this wait cycle"
                        )
                    elif attempt == 1 or attempt == attempts or attempt % 5 == 0:
                        logger.info(
                            f"[DIAGNOSTICS] {retrack_method} failed while waiting token transition "
                            f"(attempt {attempt}/{attempts}, owner={owner_for_retrack}): {exc}"
                        )

            time.sleep(delay)

        raise AssertionError(
            "Backend token state transition did not complete in time. "
            f"community_id={community_id}, deploy_tx_hash={deploy_tx_hash}, "
            f"saw_temp_owner={saw_temp_owner}, saw_temp_master={saw_temp_master}, "
            f"final_owner_token_address={final_owner_token_address}, final_master_token_address={final_master_token_address}"
        )

    def wait_for_master_token_address_after_deploy(
        self,
        backend: StatusBackend,
        community_id: str,
        deploy_tx_hash: Optional[str],
        anvil_client=None,
        attempts: int = 30,
        delay: int = 2,
    ) -> Optional[str]:
        """Resolve real master token address after deploy (signal first, receipt fallback, metadata last)."""
        expected_hash = (deploy_tx_hash or "").lower()

        # A) Fast path: address already cached in this test process.
        cached_address = getattr(self, "_last_deployed_master_token_address", None)
        if isinstance(cached_address, str) and re.fullmatch(r"0x[a-fA-F0-9]{40}", cached_address):
            return cached_address

        def _find_master_from_received_signals() -> Optional[str]:
            deployment_signals = backend.received_signals.get(SignalType.COMMUNITY_TOKEN_TRANSACTION_STATUS_CHANGED, []) or []
            for signal in reversed(deployment_signals):
                event = signal.get("event", {})
                if event.get("sendType") != COMMUNITY_DEPLOY_OWNER_TOKEN or event.get("success") is not True:
                    continue

                signal_hash = str(event.get("hash") or event.get("txHash") or event.get("transactionHash") or event.get("Hash") or "").strip().lower()
                # Keep tx-hash correlation when present, but don't reject events that omit hash fields.
                if expected_hash and signal_hash and signal_hash != expected_hash:
                    continue

                master_token_from_signal = self._extract_master_token_address_from_event(event)
                if master_token_from_signal:
                    return master_token_from_signal

            return None

        # B) Use already-received deployment status signals.
        master_token_from_signal = _find_master_from_received_signals()
        if master_token_from_signal:
            self._last_deployed_master_token_address = master_token_from_signal
            return master_token_from_signal

        # C) Wait briefly for additional deployment status signals.
        for _ in range(attempts):
            master_token_from_signal = _find_master_from_received_signals()
            if master_token_from_signal:
                self._last_deployed_master_token_address = master_token_from_signal
                return master_token_from_signal

            time.sleep(delay)

        # D) Parse deployment tx receipt directly from chain when signal reconciliation is missing.
        owner_token_from_receipt, master_token_from_receipt = self.resolve_owner_and_master_token_addresses_from_receipt(anvil_client, deploy_tx_hash)
        if owner_token_from_receipt:
            self._last_deployed_owner_token_address = owner_token_from_receipt
        if master_token_from_receipt:
            self._last_deployed_master_token_address = master_token_from_receipt
            return master_token_from_receipt

        # E) Last resort: metadata polling (persistence-dependent).
        for _ in range(attempts):
            master_token_address = self.get_community_token_contract_address(
                backend,
                community_id,
                CommunityTokenPrivilegesLevel.MASTER_LEVEL.value,
                attempts=1,
                delay=0,
            )
            if isinstance(master_token_address, str) and re.fullmatch(r"0x[a-fA-F0-9]{40}", master_token_address):
                self._last_deployed_master_token_address = master_token_address
                return master_token_address
            time.sleep(delay)

        return None

    def _log_master_token_router_diagnostics(
        self,
        backend: StatusBackend,
        community_id: str,
        chain_id: int,
        master_token_address: str,
        owner_token_address: Optional[str],
        deploy_tx_hash: Optional[str],
        temp_master_token_address: Optional[str] = None,
        temp_owner_token_address: Optional[str] = None,
        route_debug_context: Optional[dict[str, Any]] = None,
        last_router_error: Optional[str] = None,
    ):
        logger.info(
            "[DIAGNOSTICS] master token router usability snapshot | "
            f"community_id={community_id} chain_id={chain_id} master_token={master_token_address} "
            f"owner_token={owner_token_address} deploy_tx_hash={deploy_tx_hash} "
            f"temp_master_token={temp_master_token_address} temp_owner_token={temp_owner_token_address}"
        )
        if route_debug_context:
            logger.info(f"[DIAGNOSTICS] route debug context | {json.dumps(route_debug_context, default=str)}")
        if last_router_error:
            logger.info(f"[DIAGNOSTICS] last router error | {last_router_error}")

        try:
            communities_resp = backend.wakuext_service.communities()
            community = next((c for c in messenger.communities_list(communities_resp) if c.get("id") == community_id), None)
            logger.info(
                "[DIAGNOSTICS] owner community snapshot | "
                f"community_found={community is not None} "
                f"community_tokens_metadata={json.dumps((community or {}).get('communityTokensMetadata', []), default=str)}"
            )
        except Exception as exc:
            logger.warning(f"[DIAGNOSTICS] failed to get owner community snapshot: {exc}")

        try:
            deployment_signals = backend.received_signals.get(SignalType.COMMUNITY_TOKEN_TRANSACTION_STATUS_CHANGED, []) or []
            deploy_signals = [
                s
                for s in deployment_signals
                if (s.get("event", {}).get("sendType") == COMMUNITY_DEPLOY_OWNER_TOKEN)
                and ((not deploy_tx_hash) or str(s.get("event", {}).get("hash", "")).lower() == str(deploy_tx_hash).lower())
            ]
            logger.info(
                "[DIAGNOSTICS] deploy status signals snapshot | "
                f"total_signals={len(deployment_signals)} matching_deploy_signals={len(deploy_signals)}"
            )
            if deploy_signals:
                logger.info(
                    "[DIAGNOSTICS] latest deploy status signal | " f"event={json.dumps((deploy_signals[-1] or {}).get('event', {}), default=str)}"
                )
        except Exception as exc:
            logger.warning(f"[DIAGNOSTICS] failed to inspect deploy status signals: {exc}")

        rpc_calls = [
            (
                "communitytokens_safeGetOwnerTokenAddress",
                [chain_id, community_id],
            ),
            (
                "communitytokens_safeGetSignerPubKey",
                [chain_id, community_id],
            ),
        ]

        for method, params in rpc_calls:
            try:
                rpc_result = backend.rpc_valid_request(method, params)
                logger.info(f"[DIAGNOSTICS] {method}({params}) => {rpc_result}")
            except Exception as exc:
                logger.warning(f"[DIAGNOSTICS] {method}({params}) failed: {exc}")

        token_addresses_to_probe: list[tuple[str, str]] = [("final_master", master_token_address)]
        if owner_token_address:
            token_addresses_to_probe.append(("final_owner", owner_token_address))
        if temp_master_token_address:
            token_addresses_to_probe.append(("temp_master", temp_master_token_address))
        if temp_owner_token_address:
            token_addresses_to_probe.append(("temp_owner", temp_owner_token_address))

        for label, address in token_addresses_to_probe:
            try:
                remaining_supply = backend.rpc_valid_request("communitytokens_remainingSupply", [chain_id, address])
                logger.info(
                    "[DIAGNOSTICS] communitytokens_remainingSupply " f"label={label} chain_id={chain_id} address={address} => {remaining_supply}"
                )
            except Exception as exc:
                logger.warning("[DIAGNOSTICS] communitytokens_remainingSupply failed " f"label={label} chain_id={chain_id} address={address}: {exc}")

    def _extract_router_error_details(self, error_text: str) -> dict[str, Any]:
        details: dict[str, Any] = {"raw": error_text}

        json_candidate: Optional[str] = None
        marker = "response:"
        marker_idx = error_text.find(marker)
        if marker_idx >= 0:
            json_candidate = error_text[marker_idx + len(marker) :].strip()
        else:
            first_brace = error_text.find("{")
            if first_brace >= 0:
                json_candidate = error_text[first_brace:].strip()

        if json_candidate:
            try:
                parsed = json.loads(json_candidate)
                details["parsed"] = parsed
                if isinstance(parsed, dict):
                    if "error" in parsed:
                        details["error"] = parsed.get("error")
                    if "message" in parsed:
                        details["message"] = parsed.get("message")

                    nested_message = parsed.get("message")
                    if isinstance(nested_message, str):
                        nested_first_brace = nested_message.find("{")
                        if nested_first_brace >= 0:
                            try:
                                details["nested_message_parsed"] = json.loads(nested_message[nested_first_brace:])
                            except Exception:
                                pass
            except Exception:
                pass

        return details

    def log_master_token_contract_address_from_hash_diagnostics(
        self,
        backend: StatusBackend,
        deploy_tx_hash: str,
        expected_master_token_address: Optional[str] = None,
    ):
        if not (isinstance(deploy_tx_hash, str) and re.fullmatch(r"0x[a-fA-F0-9]{64}", deploy_tx_hash)):
            logger.info(f"[DIAGNOSTICS] skip getMasterTokenContractAddressFromHash due to invalid tx hash: {deploy_tx_hash}")
            return

        method = "communitytokens_getMasterTokenContractAddressFromHash"
        try:
            resolved = backend.rpc_valid_request(method, [backend.network_id, deploy_tx_hash])
            logger.info(
                "[DIAGNOSTICS] getMasterTokenContractAddressFromHash " f"chain_id={backend.network_id} tx_hash={deploy_tx_hash} resolved={resolved}"
            )
            if expected_master_token_address and isinstance(resolved, str) and re.fullmatch(r"0x[a-fA-F0-9]{40}", resolved):
                assert Web3.to_checksum_address(resolved) == Web3.to_checksum_address(expected_master_token_address), (
                    "getMasterTokenContractAddressFromHash disagrees with refreshed backend state: "
                    f"resolved={resolved}, refreshed={expected_master_token_address}"
                )
        except Exception as exc:
            logger.info(
                "[DIAGNOSTICS] getMasterTokenContractAddressFromHash unavailable or failed "
                f"chain_id={backend.network_id} tx_hash={deploy_tx_hash} error={exc}"
            )

    def derive_frontend_known_owner_and_master_token_addresses(
        self,
        deploy_tx_hash: str,
        receipt_owner_token_address: str,
        receipt_master_token_address: str,
    ) -> Tuple[str, str, str, str]:
        """
        Simulate status-app cache-backed transition for owner/master tokens:
        temp tx-hash addresses -> final receipt-derived deployed addresses.
        """
        assert isinstance(deploy_tx_hash, str) and deploy_tx_hash, f"Invalid deploy tx hash: {deploy_tx_hash}"
        assert re.fullmatch(r"0x[a-fA-F0-9]{40}", receipt_owner_token_address), f"Invalid receipt owner token address: {receipt_owner_token_address}"
        assert re.fullmatch(
            r"0x[a-fA-F0-9]{40}", receipt_master_token_address
        ), f"Invalid receipt master token address: {receipt_master_token_address}"

        temp_owner_token_address = self.temporary_owner_contract_address(deploy_tx_hash)
        temp_master_token_address = self.temporary_master_contract_address(deploy_tx_hash)

        final_owner_token_address = Web3.to_checksum_address(receipt_owner_token_address)
        final_master_token_address = Web3.to_checksum_address(receipt_master_token_address)

        logger.info(
            "Simulating status-app processDeployOwnerToken() cache-backed transition "
            f"temp_owner={temp_owner_token_address} -> owner={final_owner_token_address}, "
            f"temp_master={temp_master_token_address} -> master={final_master_token_address}"
        )

        return temp_owner_token_address, temp_master_token_address, final_owner_token_address, final_master_token_address

    def wait_until_master_token_is_router_usable(
        self,
        backend: StatusBackend,
        community_id: str,
        master_token_address: str,
        sender_wallet_address: str,
        owner_token_address: Optional[str] = None,
        recipient_wallet_addresses: Optional[list[str]] = None,
        attempts: int = 30,
        delay: int = 2,
    ):
        """Wait until router accepts master token for mint route suggestion."""
        assert re.fullmatch(r"0x[a-fA-F0-9]{40}", master_token_address), f"Invalid master token address: {master_token_address}"
        assert re.fullmatch(r"0x[a-fA-F0-9]{40}", sender_wallet_address), f"Invalid sender wallet address: {sender_wallet_address}"

        chain_id = backend.network_id
        recipient_wallet_addresses = recipient_wallet_addresses or [sender_wallet_address]
        backend.wallet_service.get_balances_at_by_chain([sender_wallet_address], [f"{chain_id}-{NATIVE_TOKEN_ADDRESS}"])

        owner_token_is_valid = isinstance(owner_token_address, str) and re.fullmatch(r"0x[a-fA-F0-9]{40}", owner_token_address) is not None
        router_master_token_address = master_token_address
        router_owner_token_address = owner_token_address

        last_backend_error = None
        retrack_method = "communitytokens_reTrackOwnerTokenDeploymentTransaction"
        deploy_tx_hash = getattr(self, "_last_deployed_tx_hash", None)
        temp_master_token_address = (
            self.temporary_master_contract_address(deploy_tx_hash) if isinstance(deploy_tx_hash, str) and deploy_tx_hash else None
        )
        temp_owner_token_address = (
            self.temporary_owner_contract_address(deploy_tx_hash) if isinstance(deploy_tx_hash, str) and deploy_tx_hash else None
        )

        # Single readiness gate: router usability for the final (cache-transitioned) master token address.
        for attempt in range(1, attempts + 1):
            transaction_uuid = str(uuid.uuid4())
            transfer_details = [
                {
                    "tokenType": CommunityTokenType.ERC721.value,
                    "privilegeLevel": CommunityTokenPrivilegesLevel.MASTER_LEVEL.value,
                    "tokenContractAddress": router_master_token_address,
                    "amount": hex(1),
                }
            ]
            route_debug_context = {
                "attempt": attempt,
                "attempts": attempts,
                "uuid": transaction_uuid,
                "community_id": community_id,
                "chain_id": chain_id,
                "sender_wallet_address": sender_wallet_address,
                "recipient_wallet_addresses": recipient_wallet_addresses,
                "deploy_tx_hash": deploy_tx_hash,
                "temp_owner_token_address": temp_owner_token_address,
                "temp_master_token_address": temp_master_token_address,
                "final_owner_token_address": router_owner_token_address,
                "final_master_token_address": router_master_token_address,
                "transfer_details": transfer_details,
            }
            logger.info(f"[DIAGNOSTICS] mint route suggestion payload | {json.dumps(route_debug_context, default=str)}")

            try:
                routes_result = backend.wallet_service.suggested_community_routes(
                    uuid=transaction_uuid,
                    send_type=COMMUNITY_MINT_TOKENS,
                    chain_id=chain_id,
                    address_from=sender_wallet_address,
                    addr_to=router_master_token_address,
                    community_id=community_id,
                    signer_pub_key=backend.public_key,
                    token_ids=[],
                    wallet_addresses=recipient_wallet_addresses,
                    transfer_details=transfer_details,
                )
                assert routes_result.get("Uuid") == transaction_uuid, f"Expected UUID {transaction_uuid}, got {routes_result.get('uuid')}"
                return routes_result
            except ApiResponseError as exc:
                error_text = str(exc)
                error_text_lower = error_text.lower()
                token_not_ready = "can't find token" in error_text_lower or "token does not exist in database" in error_text_lower

                if not token_not_ready:
                    raise

                last_backend_error = error_text
                parsed_error = self._extract_router_error_details(error_text)
                logger.info(f"[DIAGNOSTICS] parsed router error | {json.dumps(parsed_error, default=str)}")

                if owner_token_is_valid:
                    try:
                        backend.rpc_valid_request(retrack_method, [chain_id, owner_token_address])
                    except Exception as retrack_exc:
                        logger.warning(f"{retrack_method} failed for owner token {owner_token_address} on chain {chain_id}: {retrack_exc}")

                if attempt == 1 or attempt == attempts or attempt % 5 == 0:
                    self._log_master_token_router_diagnostics(
                        backend=backend,
                        community_id=community_id,
                        chain_id=chain_id,
                        master_token_address=router_master_token_address,
                        owner_token_address=router_owner_token_address,
                        deploy_tx_hash=deploy_tx_hash,
                        temp_master_token_address=temp_master_token_address,
                        temp_owner_token_address=temp_owner_token_address,
                        route_debug_context=route_debug_context,
                        last_router_error=error_text,
                    )

                if attempt < attempts:
                    logger.info(f"Master token {router_master_token_address} not router-usable yet " f"(attempt {attempt}/{attempts}); retrying")
                    time.sleep(delay)
            except Exception:
                raise

        raise AssertionError(
            "Master token was resolved but never became router-usable for mint routing. "
            f"community_id={community_id}, master_token_address={router_master_token_address}, "
            f"last_backend_error={last_backend_error}"
        )

    def get_received_token_notifications_count(self, backend: StatusBackend) -> int:
        response = backend.wakuext_service.get_activity_center_notifications(
            activity_types=[
                ActivityCenterNotificationType.NOTIFICATION_TYPE_COMMUNITY_TOKEN_RECEIVED,
                ActivityCenterNotificationType.NOTIFICATION_TYPE_FIRST_COMMUNITY_TOKEN_RECEIVED,
            ],
            limit=100,
        )

        notifications = response.get("notifications", []) or []
        return len(notifications)

    @pytest.mark.skip(reason="Pending on issue https://github.com/status-im/status-go/issues/7114")
    def test_membership_no_valid_tokens_fake_address(self, owner_backend, member_backend):
        """Test that join request with no tokens/fake address fails permission check (no request created)"""

        # Owner creates token-gated community
        community_id = community_tokens.create_token_gated_community(
            owner_backend, self.snt_address, membership=CommunityPermissionsAccess.MANUAL_ACCEPT
        )

        time.sleep(2)

        # Fetch community as member
        messenger.fetch_community(member_backend, community_id)

        # Member tries to join without tokens and with fake address - should fail permission check
        fake_address = "0x" + "0" * 40

        # Verify request got rejected right away - no request created
        join_req = community_tokens.request_to_join_with_signatures(member_backend, community_id, [fake_address])
        requests = join_req.get("requestsToJoinCommunity", [])
        assert len(requests) == 0, "No request should get accepted"

        # Verify no declined requests created
        declined_reqs = owner_backend.wakuext_service.declined_requests_to_join_for_community(community_id)
        assert len(declined_reqs) == 0

        # Verify member is not in community
        communities = member_backend.wakuext_service.communities()
        member_community = next((c for c in messenger.communities_list(communities) if c.get("id") == community_id), None)
        assert member_community is None or not member_community.get("joined", False)

    def test_membership_with_valid_tokens(self, owner_backend, member_with_snt_backend, foundry_client):
        """Test that users with required tokens can successfully join community as member"""

        # Fund the member_with_snt with 10 SNT tokens
        member_address = community_tokens.fund_backend_account_with_tokens(
            member_with_snt_backend, foundry_client, self.snt_controller_address, self.snt_address
        )
        community_tokens.verify_token_balance(foundry_client, CommunityTokenType.ERC20, self.snt_address, member_address)

        # Owner creates token-gated community with the deployed token
        community_id = community_tokens.create_token_gated_community(
            owner_backend,
            self.snt_address,
            permission_types=[CommunityTokenPermissionType.BECOME_MEMBER],
            membership=CommunityPermissionsAccess.MANUAL_ACCEPT,
        )

        # Wait shortly to ensure that we fetch community with token permissions
        # TODO: Remove this sleep somehow
        time.sleep(2)

        member_community = messenger.spectate_and_fetch_community(member_with_snt_backend, community_id)
        assert member_community, "Community not found on member"

        req_id = None
        # Member tries to join with their wallet address (should have tokens)
        with owner_backend.expect_signal(
            SignalType.MESSAGES_NEW,
            predicate=lambda s: ((s.get("event", {}).get("requestsToJoinCommunity") or [{}])[0].get("id") == req_id),
            timeout=60,
        ):
            join_req = community_tokens.request_to_join_with_signatures(member_with_snt_backend, community_id, [member_address])
            requests = join_req.get("requestsToJoinCommunity", [])
            assert requests, "No requests to join community"
            assert len(requests) == 1, "Unexpected multiple requests to join community"

            req_id = requests[0].get("id")
            logger.info(f"Sent request to join community {community_id} with id {req_id}")

        # Accept request to join
        owner_backend.wakuext_service.accept_request_to_join_community(req_id)

        # Verify member is now in community (check from owner's perspective since acceptance happened there)
        communities = owner_backend.wakuext_service.communities()
        owner_community = next((c for c in messenger.communities_list(communities) if c.get("id") == community_id), None)
        assert owner_community is not None

        # Check that member is in the community members list
        member_public_key = member_with_snt_backend.public_key
        assert member_public_key in owner_community.get("members", {}), f"Member {member_public_key} not found in community members"

    def test_admin_token_permissions_with_valid_tokens(self, owner_backend, member_with_snt_backend, foundry_client):
        """Test that users with required tokens get admin privileges"""

        # Fund the member_with_snt with 10 SNT tokens
        member_address = community_tokens.fund_backend_account_with_tokens(
            member_with_snt_backend, foundry_client, self.snt_controller_address, self.snt_address
        )
        community_tokens.verify_token_balance(foundry_client, CommunityTokenType.ERC20, self.snt_address, member_address)

        # Owner creates token-gated community with member and admin permissions
        community_id = community_tokens.create_token_gated_community(
            owner_backend,
            self.snt_address,
            permission_types=[
                CommunityTokenPermissionType.BECOME_MEMBER,
                CommunityTokenPermissionType.BECOME_ADMIN,
            ],
            membership=CommunityPermissionsAccess.MANUAL_ACCEPT,
        )

        # We should give some time for the token permissions to be distributed over network to store nodes.
        # If we don't wait enough, community can be found without token permissions (or with only 1).
        # Proper way would be this:
        # 1. wakuext_SpectateCommunity - this makes us subscribe to community updates
        # 2. if token permissions are not present,
        #    wait for `messages.new` with community update and token permissions. (might need skip a few signals)
        time.sleep(2)

        # Fetch community as member
        response = messenger.spectate_and_fetch_community(member_with_snt_backend, community_id)
        assert response, "Community not found"
        assert response["tokenPermissions"], "No token permissions found"
        assert len(response["tokenPermissions"]) == 2, "Unexpected number of token permissions"

        # Wait for balance to be fetched (request to join uses cached balances)
        # Trigger explicit refresh and wait until permission check sees the balance
        member_with_snt_backend.wallet_service.fetch_or_get_cached_wallet_balances([member_address], True)
        permissions_resp = None
        for attempt in range(3):
            time.sleep(2)
            permissions_resp = member_with_snt_backend.wakuext_service.check_permissions_to_join_community(community_id)
            if permissions_resp and permissions_resp.get("satisfied"):
                break

            # Retry by forcing refresh again (balance update is async)
            member_with_snt_backend.wallet_service.fetch_or_get_cached_wallet_balances([member_address], True)

        assert permissions_resp, "Failed to check permissions to join community"
        assert permissions_resp.get("satisfied"), "Permissions to join are not satisfied"

        req_id = None
        # Member with tokens requests to join community
        with owner_backend.expect_signal(
            SignalType.MESSAGES_NEW,
            predicate=lambda signal: any(r.get("id") == req_id for r in (signal.get("event", {}).get("requestsToJoinCommunity") or [])),
            timeout=60,
        ):
            join_req = community_tokens.request_to_join_with_signatures(member_with_snt_backend, community_id, [member_address])
            requests = join_req.get("requestsToJoinCommunity", [])
            assert requests, "No requests to join community"
            assert len(requests) == 1, "Unexpected multiple requests to join community"

            req_id = requests[0].get("id")
            logger.info(f"Sent request to join community {community_id} with id {req_id}")

        accept_resp = owner_backend.wakuext_service.accept_request_to_join_community(req_id)
        assert accept_resp is not None, f"Failed to accept request: {accept_resp}"

        # Verify member is now in community and has admin role
        communities = owner_backend.wakuext_service.communities()
        owner_community = next(
            (c for c in messenger.communities_list(communities) if c.get("id") == community_id),
            None,
        )
        assert owner_community is not None

        member_key = member_with_snt_backend.public_key
        owner_key = owner_backend.public_key

        # Member should have admin role, granted via BECOME_ADMIN token permission
        assert member_key in owner_community.get("members", {})
        assert CommunityRoles.ROLE_ADMIN.value in owner_community["members"][member_key].get("roles", [])

        # Owner should remain owner only, not changed by token reevaluation
        assert owner_key in owner_community.get("members", {})
        assert CommunityRoles.ROLE_OWNER.value in owner_community["members"][owner_key].get("roles", [])

    def test_owner_edits_visible_before_and_after_minting_owner_token(self, owner_backend, member_backend, foundry_client, anvil_client):
        """Test that owner edits are visible before and after minting the owner token"""

        # Owner creates a token-gated community
        community_id = community_tokens.create_token_gated_community(
            owner_backend,
            self.snt_address,
            permission_types=[CommunityTokenPermissionType.BECOME_MEMBER],
            membership=CommunityPermissionsAccess.MANUAL_ACCEPT,
        )

        # Fetch/spectate community as member and wait until it is available with propagated metadata
        time.sleep(2)
        response = messenger.spectate_and_fetch_community(member_backend, community_id)
        assert response, "Community not found"

        # Fund member and verify token balance so member satisfies token-gated permission
        member_address = community_tokens.fund_backend_account_with_tokens(
            member_backend, foundry_client, self.snt_controller_address, self.snt_address
        )
        community_tokens.verify_token_balance(foundry_client, CommunityTokenType.ERC20, self.snt_address, member_address)

        # Balance and permission checks are async/cached, force refresh and retry.
        # `community.memberReevaluationStatus` is not reliably emitted in this path.
        member_backend.wallet_service.fetch_or_get_cached_wallet_balances([member_address], True)
        permissions_resp = None
        for _ in range(10):
            time.sleep(2)
            # Refresh community view too, token-permission propagation can lag behind fetchCommunity
            messenger.fetch_community(member_backend, community_id)
            permissions_resp = member_backend.wakuext_service.check_permissions_to_join_community(community_id)
            if permissions_resp and permissions_resp.get("satisfied"):
                break

            member_backend.wallet_service.fetch_or_get_cached_wallet_balances([member_address], True)

        assert permissions_resp, "Failed to check permissions to join community"
        assert permissions_resp.get("satisfied"), "Permissions to join are not satisfied"

        req_id = None

        # Wait for member validation
        with owner_backend.expect_signal(
            SignalType.MESSAGES_NEW,
            predicate=lambda signal: any(r.get("id") == req_id for r in (signal.get("event", {}).get("requestsToJoinCommunity") or [])),
            timeout=60,
        ):
            join_resp = community_tokens.request_to_join_with_signatures(member_backend, community_id, [member_address])
            requests = join_resp.get("requestsToJoinCommunity", [])

            assert requests, "No requests to join community"
            assert len(requests) == 1, "Unexpected multiple requests to join community"

            req_id = requests[0].get("id")

        time.sleep(2)

        accept_resp = owner_backend.wakuext_service.accept_request_to_join_community(req_id)
        assert accept_resp is not None, f"Failed to accept request: {accept_resp}"

        # Verify member is in community
        communities = owner_backend.wakuext_service.communities()
        owner_community = next((c for c in messenger.communities_list(communities) if c.get("id") == community_id), None)
        assert owner_community is not None
        assert member_backend.public_key in owner_community.get("members", {})

        # When the Owner edits the community
        with member_backend.expect_signal(
            SignalType.MESSAGES_NEW,
            predicate=lambda signal: community_id in json.dumps(signal),
            timeout=60,
        ):
            new_name, new_description = messenger.edit_community(owner_backend, community_id)

        # Then the Member sees the updated community
        assert self.check_member_community_updated(member_backend, community_id, new_name, new_description)

        # Fund owner wallet with native token on Anvil to cover gas for owner-token deployment
        owner_accounts = owner_backend.accounts_service.get_accounts()
        assert owner_accounts, "No owner accounts found"
        owner_wallet_account = next(a for a in owner_accounts if not a.get("chat"))
        assert owner_wallet_account, "No owner wallet account found"
        owner_address = owner_wallet_account["address"]

        ten_native_tokens_in_wei = 10 * 10**18
        anvil_client.set_balance(owner_address, ten_native_tokens_in_wei)

        # Refresh wallet balances cache after forcing Anvil balance
        owner_backend.wallet_service.fetch_or_get_cached_wallet_balances([owner_address], True)
        owner_backend.wallet_service.get_balances_at_by_chain([owner_address], [f"{owner_backend.network_id}-{NATIVE_TOKEN_ADDRESS}"])

        # When the Owner mints the owner token
        owner_backend.wallet_service.restart_wallet_reload_timer()
        time.sleep(2)  # Sync metadata
        self.deploy_owner_token(owner_backend, community_id, anvil_client=anvil_client)

        # And the Owner edits the community again. After deploy_owner_token the community
        # has HasTokenOwnership=true, so MESSAGES_NEW is unreliable (messages may be queued
        # for async owner-token verification). Use store-node polling instead.
        new_name2, new_description2 = messenger.edit_community(owner_backend, community_id)

        # Then the Member sees the updated community
        for _ in range(30):
            messenger.fetch_community(member_backend, community_id, try_database=False)
            if self.check_member_community_updated(member_backend, community_id, new_name2, new_description2):
                break
            time.sleep(2)
        assert self.check_member_community_updated(member_backend, community_id, new_name2, new_description2)

        # When the Owner logouts and logs back in
        owner_backend.logout()
        owner_backend.login(owner_backend.key_uid, owner_backend.password)
        owner_backend.wait_for_login()
        owner_backend.wait_for_wakuext_ready(timeout=60, start_messenger=True)

        # And the Owner edits the community again
        new_name3, new_description3 = messenger.edit_community(owner_backend, community_id)

        # Then the Member sees the updated community
        for _ in range(30):
            messenger.fetch_community(member_backend, community_id, try_database=False)
            if self.check_member_community_updated(member_backend, community_id, new_name3, new_description3):
                break
            time.sleep(2)
        assert self.check_member_community_updated(member_backend, community_id, new_name3, new_description3)

    def test_master_token_holder_can_edit_and_mint_tokens(
        self, owner_backend, member_backend, backend_new_profile, multicall3_deployer, anvil_client
    ):
        """Test that a master token holder can edit community and mint/airdrop tokens"""

        member_b_backend = backend_new_profile(
            name="member_b",
            token_overrides=[
                {
                    "symbol": "SNT",
                    "name": "Status Network Token",
                    "address": self.snt_address,
                    "decimals": 18,
                }
            ],
            multicall_contract_address=multicall3_deployer.contract_address,
            community_token_deployer_contract_address=self.community_token_deployer,
        )

        # Given the Owner has created a community
        community_resp = owner_backend.wakuext_service.create_community(
            name=fake.community_name(),
            description=fake.community_description(),
            membership=CommunityPermissionsAccess.MANUAL_ACCEPT,
        )
        community_id = community_resp.get("communities", [{}])[0].get("id")

        # And Member A and Member B have joined the community
        time.sleep(2)
        assert messenger.spectate_and_fetch_community(member_backend, community_id), "Community not found for member A"
        assert messenger.spectate_and_fetch_community(member_b_backend, community_id), "Community not found for member B"

        member_a_wallet = messenger.wallet_address(member_backend)
        member_b_wallet = messenger.wallet_address(member_b_backend)
        community_tokens.join_community_with_signatures_and_accept(owner_backend, member_backend, community_id, member_a_wallet)
        community_tokens.join_community_with_signatures_and_accept(owner_backend, member_b_backend, community_id, member_b_wallet)

        owner_community = next(
            (c for c in messenger.communities_list(owner_backend.wakuext_service.communities()) if c.get("id") == community_id), None
        )
        assert owner_community is not None
        assert member_backend.public_key in owner_community.get("members", {})
        assert member_b_backend.public_key in owner_community.get("members", {})

        # Fund Owner and Member A with native token to pay gas
        community_tokens.fund_native_balance(owner_backend, anvil_client)
        community_tokens.fund_native_balance(member_backend, anvil_client)

        # When the Owner mints the owner token
        owner_backend.wallet_service.restart_wallet_reload_timer()
        time.sleep(2)
        deploy_tx_hash = self.deploy_owner_token(
            owner_backend,
            community_id,
            anvil_client=anvil_client,
            wait_for_deploy_status_signal=False,
        )
        self._last_deployed_tx_hash = deploy_tx_hash

        # Derive frontend-known temporary identities from tx hash and verify they appear first.
        assert deploy_tx_hash and re.fullmatch(r"0x[a-fA-F0-9]{64}", deploy_tx_hash), f"Invalid deploy tx hash: {deploy_tx_hash}"
        expected_temp_owner = self.temporary_owner_contract_address(deploy_tx_hash)
        expected_temp_master = self.temporary_master_contract_address(deploy_tx_hash)
        assert getattr(self, "_last_deployed_owner_token_placeholder", None) == expected_temp_owner, (
            "Temporary owner token identity was not registered locally after deployment: "
            f"expected={expected_temp_owner}, observed={getattr(self, '_last_deployed_owner_token_placeholder', None)}"
        )
        assert getattr(self, "_last_deployed_master_token_placeholder", None) == expected_temp_master, (
            "Temporary master token identity was not registered locally after deployment: "
            f"expected={expected_temp_master}, observed={getattr(self, '_last_deployed_master_token_placeholder', None)}"
        )

        # Wait for backend publish/refresh transition and then use refreshed finalized addresses.
        (
            temp_owner_token_address,
            temp_master_token_address,
            owner_token_address,
            master_token_address,
            saw_temp_owner_in_backend,
            saw_temp_master_in_backend,
        ) = self.wait_for_backend_token_state_transition(
            backend=owner_backend,
            community_id=community_id,
            deploy_tx_hash=deploy_tx_hash,
            attempts=30,
            delay=2,
        )

        refreshed_owner_addresses, refreshed_master_addresses = self._read_backend_token_addresses_by_privilege(owner_backend, community_id)
        assert owner_token_address in refreshed_owner_addresses, (
            "Finalized owner token address missing from refreshed backend state: "
            f"final={owner_token_address}, observed={refreshed_owner_addresses}"
        )
        assert master_token_address in refreshed_master_addresses, (
            "Finalized master token address missing from refreshed backend state: "
            f"final={master_token_address}, observed={refreshed_master_addresses}"
        )
        if saw_temp_owner_in_backend:
            assert temp_owner_token_address not in refreshed_owner_addresses, (
                "Temporary owner token identity should be replaced after backend finalization: "
                f"temporary={temp_owner_token_address}, observed={refreshed_owner_addresses}"
            )
        if saw_temp_master_in_backend:
            assert temp_master_token_address not in refreshed_master_addresses, (
                "Temporary master token identity should be replaced after backend finalization: "
                f"temporary={temp_master_token_address}, observed={refreshed_master_addresses}"
            )

        logger.info(
            "Frontend-like token transition observed "
            f"(local temp always, backend temp owner={saw_temp_owner_in_backend}, backend temp master={saw_temp_master_in_backend})"
        )
        self.log_master_token_contract_address_from_hash_diagnostics(
            backend=owner_backend,
            deploy_tx_hash=deploy_tx_hash,
            expected_master_token_address=master_token_address,
        )

        # Optional debug aid only: receipt resolution must not be the mint source of truth.
        receipt_owner_token_address, receipt_master_token_address = self.resolve_owner_and_master_token_addresses_from_receipt(
            anvil_client=anvil_client,
            tx_hash=deploy_tx_hash,
        )
        if receipt_owner_token_address:
            assert Web3.to_checksum_address(receipt_owner_token_address) == owner_token_address
        if receipt_master_token_address:
            assert Web3.to_checksum_address(receipt_master_token_address) == master_token_address

        # And the Owner airdrops the master token to Member A using refreshed backend state.
        owner_wallet = messenger.wallet_address(owner_backend)
        minted_master_token_address = master_token_address
        logger.info(
            "Using refreshed backend state token addresses for mint routing "
            f"(frontend cache model): temp_owner={temp_owner_token_address}, temp_master={temp_master_token_address}, "
            f"owner={owner_token_address}, master={minted_master_token_address}"
        )

        self.wait_until_master_token_is_router_usable(
            backend=owner_backend,
            community_id=community_id,
            master_token_address=minted_master_token_address,
            sender_wallet_address=owner_wallet,
            owner_token_address=owner_token_address,
            recipient_wallet_addresses=[member_a_wallet],
            attempts=30,
            delay=2,
        )

        # Mint master token to Member A and wait for the on-chain confirmation signal before
        # triggering reevaluation. The community members reevaluation loop is not started
        # automatically by the deploy flow, so we must schedule it explicitly after the mint
        # tx is confirmed — otherwise the 5-minute cooldown would block an earlier loop run.
        with owner_backend.expect_signal(
            SignalType.COMMUNITY_TOKEN_TRANSACTION_STATUS_CHANGED,
            predicate=self._mint_tx_confirmed_predicate,
            timeout=60,
        ):
            self.mint_community_token(
                sender_backend=owner_backend,
                community_id=community_id,
                token_contract_address=minted_master_token_address,
                wallet_addresses=[member_a_wallet],
                token_type=CommunityTokenType.ERC721,
                privilege_level=CommunityTokenPrivilegesLevel.MASTER_LEVEL.value,
                amount=1,
            )

        # Trigger community members reevaluation so the owner (control node) picks up
        # Member A's new master token, assigns the role, and publishes over Waku.
        owner_backend.wakuext_service.reevaluate_community_members_permissions(community_id)

        # Poll the owner backend until it sees Member A's ROLE_TOKEN_MASTER.
        community_tokens.wait_for_member_role(
            owner_backend,
            community_id,
            member_backend.public_key,
            CommunityRoles.ROLE_TOKEN_MASTER.value,
            attempts=10,
            delay=2,
        )

        # After the owner sees the role, wait for member_backend's own local state to
        # reflect it. Force a store-node fetch on each iteration so the member pulls the
        # latest community description (token permissions + member roles).
        community_tokens.wait_for_member_role(
            member_backend,
            community_id,
            member_backend.public_key,
            CommunityRoles.ROLE_TOKEN_MASTER.value,
            attempts=30,
            delay=2,
            fetch_from_store=True,
            required=False,
        )

        # When Member A edits the community
        new_name, new_description = messenger.edit_community(member_backend, community_id)

        # Then the Owner and Member B see the updated community. Poll with store-node
        # fetches (try_database=False) because community description updates for
        # token-ownership communities are processed asynchronously on receivers and do
        # not reliably emit MESSAGES_NEW signals.
        for backend in (owner_backend, member_b_backend):
            for _ in range(30):
                messenger.fetch_community(backend, community_id, try_database=False)
                if self.check_member_community_updated(backend, community_id, new_name, new_description):
                    break
                time.sleep(2)
            assert self.check_member_community_updated(
                backend, community_id, new_name, new_description
            ), f"{backend} did not see the updated community name/description"

        # When Member A mints a new token (airdrop master token to Owner).
        # The owner is a privileged member and receives a CommunityTokenAction(AIRDROP)
        # message from member_backend's communitytokens service once the tx is confirmed.
        with owner_backend.expect_signal(
            SignalType.COMMUNITY_TOKEN_ACTION,
            predicate=lambda s: s.get("event", {}).get("actionType") == 1,  # AIRDROP
            timeout=60,
        ):
            self.mint_community_token(
                sender_backend=member_backend,
                community_id=community_id,
                token_contract_address=minted_master_token_address,
                wallet_addresses=[messenger.wallet_address(owner_backend)],
                token_type=CommunityTokenType.ERC721,
                privilege_level=CommunityTokenPrivilegesLevel.MASTER_LEVEL.value,
                amount=1,
            )

        # Then the Owner has the master token
        owner_address = messenger.wallet_address(owner_backend)
        balance_of_abi = [
            {
                "inputs": [{"name": "owner", "type": "address"}],
                "name": "balanceOf",
                "outputs": [{"name": "", "type": "uint256"}],
                "stateMutability": "view",
                "type": "function",
            }
        ]
        master_token_contract = anvil_client.eth.contract(address=Web3.to_checksum_address(minted_master_token_address), abi=balance_of_abi)
        owner_master_balance = master_token_contract.functions.balanceOf(Web3.to_checksum_address(owner_address)).call()
        assert owner_master_balance >= 1, f"Owner should hold at least 1 master token, got {owner_master_balance}"

        # When Member A airdrops the minted token to Member B.
        # Member B is not a privileged member so they don't receive a CommunityTokenAction
        # message directly. Verify the mint tx was confirmed on-chain via the sender signal.
        with member_backend.expect_signal(
            SignalType.COMMUNITY_TOKEN_TRANSACTION_STATUS_CHANGED,
            predicate=self._mint_tx_confirmed_predicate,
            timeout=60,
        ):
            self.mint_community_token(
                sender_backend=member_backend,
                community_id=community_id,
                token_contract_address=minted_master_token_address,
                wallet_addresses=[member_b_wallet],
                token_type=CommunityTokenType.ERC721,
                privilege_level=CommunityTokenPrivilegesLevel.MASTER_LEVEL.value,
                amount=1,
            )

        # Then Member B has the master token
        member_b_master_balance = master_token_contract.functions.balanceOf(Web3.to_checksum_address(member_b_wallet)).call()
        assert member_b_master_balance >= 1, f"Member B should hold at least 1 master token, got {member_b_master_balance}"
