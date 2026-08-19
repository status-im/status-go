import json
import logging
import re
import time
import uuid
from dataclasses import dataclass
from typing import Optional, Tuple

from web3 import Web3

from clients.api import ApiResponseError
from clients.transaction_receipt_status import (
    TransactionReceiptStatus,
    receipt_status_is_success,
)
from clients.wallet_pending_tx_status import WalletPendingTxStatus
from clients.services.wakuext import (
    CommunityTokenPermissionType,
    CommunityTokenPrivilegesLevel,
    CommunityTokenType,
    CommunityRoles,
)
from clients.signals import SignalType
from clients.status_backend import StatusBackend
from clients.services.wallet_send_type import WalletSendType
from steps import community_tokens, messenger
from steps.community_tokens import NATIVE_TOKEN_ADDRESS
from utils.keys import change_community_key_compression

logger = logging.getLogger(__name__)

DEFAULT_TOKEN_BASE64_IMAGE = "data:image/gif;base64,R0lGODlhAQABAIAAAP///wAAACwAAAAAAQABAAACAkQBADs="


@dataclass
class CommunityTokenDeployState:
    deploy_tx_hash: Optional[str] = None
    owner_token_placeholder: Optional[str] = None
    master_token_placeholder: Optional[str] = None
    master_token_address: Optional[str] = None
    owner_token_address: Optional[str] = None


def is_community_token_tx_success(signal: dict, send_type: WalletSendType, tx_hash: Optional[str] = None) -> bool:
    """Return True when the signal reports a successful community token router transaction."""
    event = signal.get("event") or {}
    if event.get("sendType") != send_type:
        return False
    if event.get("success") is not True:
        return False
    if tx_hash is not None and str(event.get("hash", "")).lower() != str(tx_hash).lower():
        return False
    return True


def _is_valid_tx_hash(tx_hash: Optional[str]) -> bool:
    return isinstance(tx_hash, str) and re.fullmatch(r"0x[a-fA-F0-9]{64}", tx_hash) is not None


def _require_tx_hash(tx_hash: Optional[str], *, context: str) -> str:
    if not _is_valid_tx_hash(tx_hash):
        raise AssertionError(f"{context}: expected a valid tx hash, got {tx_hash!r}")
    assert isinstance(tx_hash, str)
    return tx_hash


def _store_deployed_owner_token(
    owner_backend: StatusBackend,
    *,
    address_from: str,
    chain_id: int,
    tx_hash: str,
    owner_token_parameters: dict,
    master_token_parameters: dict,
    state: CommunityTokenDeployState,
) -> None:
    state.master_token_placeholder = temporary_master_contract_address(tx_hash)
    state.owner_token_placeholder = temporary_owner_contract_address(tx_hash)
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


def _wait_wallet_deploy_tx_success(owner_backend: StatusBackend, sent_hashes: list[str], timeout: float = 60) -> dict:
    def _accept_wallet_signal(signal: dict) -> bool:
        event = signal.get("event", {})
        if event.get("type") != "pending-transaction-status-changed":
            return False
        try:
            tx_status = json.loads(event.get("message", "{}").replace("'", '"'))
        except json.JSONDecodeError:
            return False
        if tx_status.get("status") != WalletPendingTxStatus.SUCCESS:
            return False
        if sent_hashes:
            return any(h in sent_hashes for h in _extract_tx_hashes({"event": tx_status}))
        return True

    with owner_backend.expect_signal(SignalType.WALLET, accept_fn=_accept_wallet_signal, timeout=timeout) as wallet_exp:
        pass

    wallet_signal = wallet_exp.result
    assert isinstance(wallet_signal, dict), f"Unexpected wallet signal payload: {wallet_signal}"
    return json.loads(wallet_signal["event"]["message"].replace("'", '"'))


def _resolve_deploy_tx_hash(sent_hashes: list[str], wallet_tx_status: dict) -> Optional[str]:
    if sent_hashes and _is_valid_tx_hash(sent_hashes[0]):
        return sent_hashes[0]
    for candidate in _extract_tx_hashes({"event": wallet_tx_status}):
        if _is_valid_tx_hash(candidate):
            return candidate
    return None


def _wait_deploy_owner_token_success_signal(owner_backend: StatusBackend, tx_hash: str, timeout: float = 60) -> dict:
    with owner_backend.expect_signal(
        SignalType.COMMUNITY_TOKEN_TRANSACTION_STATUS_CHANGED,
        predicate=lambda s: is_community_token_tx_success(s, WalletSendType.COMMUNITY_DEPLOY_OWNER_TOKEN, tx_hash),
        timeout=timeout,
        start="beginning",
    ) as community_token_exp:
        pass

    community_token_signal = community_token_exp.result
    assert isinstance(community_token_signal, dict), f"Unexpected community token signal payload: {community_token_signal}"
    return community_token_signal


def _wait_anvil_tx_mined(anvil_client, tx_hash: str, timeout_seconds: int = 30) -> None:
    receipt = anvil_client.eth.wait_for_transaction_receipt(tx_hash, timeout=timeout_seconds)
    assert receipt_status_is_success(receipt.get("status")), (
        f"Owner token deployment tx {tx_hash} failed on chain (status={receipt.get('status')}, " f"expected {TransactionReceiptStatus.SUCCESS.name})"
    )
    logger.info("Owner token deployment tx minted on chain")


def deploy_owner_token(
    owner_backend,
    community_id,
    community_token_deployer,
    state,
    anvil_client=None,
    wait_for_deploy_status_signal: bool = True,
):
    """Deploy and mint owner token for the community"""

    accounts = owner_backend.accounts_service.get_accounts()
    assert accounts, "No accounts found"
    wallet_account = next(a for a in accounts if not a.get("chat"))
    assert wallet_account, "No wallet account found"
    address_from = wallet_account["address"]

    # CommunityTokenDeployer contract address
    contract_address = community_token_deployer

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
        "base64image": DEFAULT_TOKEN_BASE64_IMAGE,
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
        "base64image": DEFAULT_TOKEN_BASE64_IMAGE,
    }

    # Generate UUID for the transaction
    transaction_uuid = str(uuid.uuid4())

    # Fetch balances
    owner_backend.wallet_service.get_balances_at_by_chain([address_from], [f"{chain_id}-{NATIVE_TOKEN_ADDRESS}"])

    # Get suggested routes for deploying tokens
    signer_pub_key = owner_backend.public_key

    routes_result = owner_backend.wallet_service.suggested_community_routes(
        uuid=transaction_uuid,
        send_type=WalletSendType.COMMUNITY_DEPLOY_OWNER_TOKEN,
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

    sent_signal, sent_hashes = _sign_and_send_community_route(owner_backend, transaction_uuid, address_from)
    logger.debug(f"Sent router transactions for UUID {transaction_uuid}: {sent_signal}")

    tx_hash: Optional[str] = sent_hashes[0] if sent_hashes else None
    stored = False
    if _is_valid_tx_hash(tx_hash):
        try:
            _store_deployed_owner_token(
                owner_backend,
                address_from=address_from,
                chain_id=chain_id,
                tx_hash=_require_tx_hash(tx_hash, context="early storeDeployedOwnerToken"),
                owner_token_parameters=owner_token_parameters,
                master_token_parameters=master_token_parameters,
                state=state,
            )
            stored = True
        except ApiResponseError as exc:
            logger.info(f"Early storeDeployedOwnerToken failed, will retry after tx confirmation: {exc}")

    wallet_tx_status = {}
    if not _is_valid_tx_hash(tx_hash):
        wallet_tx_status = _wait_wallet_deploy_tx_success(owner_backend, sent_hashes)
        tx_hash = _resolve_deploy_tx_hash(sent_hashes, wallet_tx_status)

    tx_hash = _require_tx_hash(
        tx_hash,
        context=f"owner token deploy (sent_hashes={sent_hashes}, wallet_status={wallet_tx_status})",
    )

    if wallet_tx_status:
        logger.info(f"Owner token deployment wallet status {wallet_tx_status}")
    logger.info(f"Owner token deployment tx hash {tx_hash}")

    if not stored:
        _store_deployed_owner_token(
            owner_backend,
            address_from=address_from,
            chain_id=chain_id,
            tx_hash=tx_hash,
            owner_token_parameters=owner_token_parameters,
            master_token_parameters=master_token_parameters,
            state=state,
        )

    if wait_for_deploy_status_signal:
        deploy_signal = _wait_deploy_owner_token_success_signal(owner_backend, tx_hash)
        logger.debug(f"Community token deployment status signal: {deploy_signal}")
        master_token_address = _extract_master_token_address_from_event(deploy_signal.get("event", {}))
        if master_token_address:
            state.master_token_address = master_token_address

    if anvil_client:
        _wait_anvil_tx_mined(anvil_client, tx_hash)

    state.deploy_tx_hash = tx_hash
    return tx_hash


def mint_community_token(
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
        send_type=WalletSendType.COMMUNITY_MINT_TOKENS,
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

    sent_signal, _ = _sign_and_send_community_route(sender_backend, transaction_uuid, address_from)
    return sent_signal


def _deploy_community_token_via_router(
    owner_backend,
    community_id: str,
    owner_token_address: str,
    master_token_address: str,
    community_token_deployer: str,
    foundry_client,
    *,
    send_type: WalletSendType,
    store_rpc_method: str,
    deployment_parameters: dict,
    anvil_client=None,
) -> str:
    """Deploy a community permission token contract via wallet router."""
    address_from = messenger.wallet_address(owner_backend)
    chain_id = owner_backend.network_id

    owner_backend.wallet_service.get_balances_at_by_chain([address_from], [f"{chain_id}-{NATIVE_TOKEN_ADDRESS}"])
    deploy_nonce = anvil_client.eth.get_transaction_count(Web3.to_checksum_address(address_from)) if anvil_client else 0
    predicted_address = _predict_contract_address(foundry_client, address_from, deploy_nonce)

    transaction_uuid = str(uuid.uuid4())
    routes_result = owner_backend.wallet_service.suggested_community_routes(
        uuid=transaction_uuid,
        send_type=send_type,
        chain_id=chain_id,
        address_from=address_from,
        addr_to=community_token_deployer,
        community_id=community_id,
        signer_pub_key=owner_backend.public_key,
        token_ids=[],
        wallet_addresses=[],
        transfer_details=[],
        deployment_parameters=deployment_parameters,
    )
    assert routes_result.get("Uuid") == transaction_uuid, f"Expected UUID {transaction_uuid}, got {routes_result.get('uuid')}"

    _, sent_hashes = _sign_and_send_community_route(owner_backend, transaction_uuid, address_from)
    tx_hash = sent_hashes[0] if sent_hashes else None
    assert tx_hash, f"No tx hash from deploy route ({send_type.name}): sent_hashes={sent_hashes}"

    owner_backend.rpc_valid_request(
        store_rpc_method,
        [address_from, predicted_address, chain_id, tx_hash, deployment_parameters],
    )

    if anvil_client:
        receipt = anvil_client.eth.wait_for_transaction_receipt(tx_hash, timeout=60)
        assert receipt_status_is_success(receipt.get("status")), (
            f"Deploy tx failed ({send_type.name}): {tx_hash} " f"(status={receipt.get('status')}, expected {TransactionReceiptStatus.SUCCESS.name})"
        )
        deployed_address = Web3.to_checksum_address(receipt["contractAddress"])
        assert (
            deployed_address.lower() == predicted_address.lower()
        ), f"Predicted contract address mismatch: predicted={predicted_address}, receipt={deployed_address}"
    else:
        deployed_address = predicted_address

    with owner_backend.expect_signal(
        SignalType.COMMUNITY_TOKEN_TRANSACTION_STATUS_CHANGED,
        predicate=lambda s: is_community_token_tx_success(s, send_type, tx_hash),
        timeout=60,
    ):
        pass

    return deployed_address


def deploy_community_erc20_asset(
    owner_backend,
    community_id: str,
    owner_token_address: str,
    master_token_address: str,
    community_token_deployer: str,
    foundry_client,
    anvil_client=None,
) -> str:
    """Deploy a community ERC20 asset token via wallet router (CommunityDeployAssets)."""
    token_uri = change_community_key_compression(community_id) + "/"
    deployment_parameters = {
        "name": "AdminToken",
        "symbol": "ADM",
        "supply": str(10**24),
        "infiniteSupply": False,
        "transferable": True,
        "remoteSelfDestruct": False,
        "tokenUri": token_uri,
        "ownerTokenAddress": Web3.to_checksum_address(owner_token_address),
        "masterTokenAddress": Web3.to_checksum_address(master_token_address),
        "communityId": community_id,
        "description": "Admin permission token",
        "base64image": DEFAULT_TOKEN_BASE64_IMAGE,
        "decimals": 18,
    }
    return _deploy_community_token_via_router(
        owner_backend,
        community_id,
        owner_token_address,
        master_token_address,
        community_token_deployer,
        foundry_client,
        send_type=WalletSendType.COMMUNITY_DEPLOY_ASSETS,
        store_rpc_method="communitytokens_storeDeployedAssets",
        deployment_parameters=deployment_parameters,
        anvil_client=anvil_client,
    )


def deploy_community_erc721_collectible(
    owner_backend,
    community_id: str,
    owner_token_address: str,
    master_token_address: str,
    community_token_deployer: str,
    foundry_client,
    anvil_client=None,
) -> str:
    """Deploy a community ERC721 collectible via wallet router (CommunityDeployCollectibles)."""
    token_uri = change_community_key_compression(community_id) + "/"
    deployment_parameters = {
        "name": "AdminCollectible",
        "symbol": "ADC",
        "supply": "1000",
        "infiniteSupply": False,
        "transferable": True,
        "remoteSelfDestruct": False,
        "tokenUri": token_uri,
        "ownerTokenAddress": Web3.to_checksum_address(owner_token_address),
        "masterTokenAddress": Web3.to_checksum_address(master_token_address),
        "communityId": community_id,
        "description": "Admin permission collectible",
        "base64image": DEFAULT_TOKEN_BASE64_IMAGE,
        "decimals": 0,
    }
    return _deploy_community_token_via_router(
        owner_backend,
        community_id,
        owner_token_address,
        master_token_address,
        community_token_deployer,
        foundry_client,
        send_type=WalletSendType.COMMUNITY_DEPLOY_COLLECTIBLES,
        store_rpc_method="communitytokens_storeDeployedCollectibles",
        deployment_parameters=deployment_parameters,
        anvil_client=anvil_client,
    )


def deploy_community_admin_permission_token(
    owner_backend,
    community_id: str,
    owner_token_address: str,
    master_token_address: str,
    token_type: CommunityTokenType,
    community_token_deployer: str,
    foundry_client,
    anvil_client=None,
) -> str:
    if token_type == CommunityTokenType.ERC20:
        return deploy_community_erc20_asset(
            owner_backend,
            community_id,
            owner_token_address,
            master_token_address,
            community_token_deployer,
            foundry_client,
            anvil_client=anvil_client,
        )
    return deploy_community_erc721_collectible(
        owner_backend,
        community_id,
        owner_token_address,
        master_token_address,
        community_token_deployer,
        foundry_client,
        anvil_client=anvil_client,
    )


def airdrop_community_admin_token(
    sender_backend: StatusBackend,
    community_id: str,
    token_contract_address: str,
    wallet_addresses: list[str],
    token_type: CommunityTokenType,
):
    amount = 10**18 if token_type == CommunityTokenType.ERC20 else 1
    with sender_backend.expect_signal(
        SignalType.COMMUNITY_TOKEN_TRANSACTION_STATUS_CHANGED,
        predicate=lambda s: is_community_token_tx_success(s, WalletSendType.COMMUNITY_MINT_TOKENS),
        timeout=60,
    ):
        mint_community_token(
            sender_backend=sender_backend,
            community_id=community_id,
            token_contract_address=token_contract_address,
            wallet_addresses=wallet_addresses,
            token_type=token_type,
            privilege_level=CommunityTokenPrivilegesLevel.COMMUNITY_LEVEL.value,
            amount=amount,
        )


def wait_until_community_admin_token_is_mint_usable(
    backend: StatusBackend,
    community_id: str,
    token_contract_address: str,
    sender_wallet_address: str,
    recipient_wallet_addresses: list[str],
    token_type: CommunityTokenType,
    attempts: int = 30,
    delay: int = 2,
):
    chain_id = backend.network_id
    mint_amount = 10**18 if token_type == CommunityTokenType.ERC20 else 1
    backend.wallet_service.get_balances_at_by_chain([sender_wallet_address], [f"{chain_id}-{NATIVE_TOKEN_ADDRESS}"])
    transfer_details = [
        {
            "tokenType": token_type.value,
            "privilegeLevel": CommunityTokenPrivilegesLevel.COMMUNITY_LEVEL.value,
            "tokenContractAddress": token_contract_address,
            "amount": hex(mint_amount),
        }
    ]
    last_error = None
    for attempt in range(1, attempts + 1):
        transaction_uuid = str(uuid.uuid4())
        try:
            routes_result = backend.wallet_service.suggested_community_routes(
                uuid=transaction_uuid,
                send_type=WalletSendType.COMMUNITY_MINT_TOKENS,
                chain_id=chain_id,
                address_from=sender_wallet_address,
                addr_to=token_contract_address,
                community_id=community_id,
                signer_pub_key=backend.public_key,
                token_ids=[],
                wallet_addresses=recipient_wallet_addresses,
                transfer_details=transfer_details,
            )
            assert routes_result.get("Uuid") == transaction_uuid
            return routes_result
        except ApiResponseError as exc:
            last_error = exc
            if attempt < attempts:
                time.sleep(delay)
    raise AssertionError(f"{token_type.name} admin token {token_contract_address} not mint-usable after {attempts} attempts: {last_error}")


def admin_permission_token_criteria(token_type: CommunityTokenType, network_id: int, contract_address: str) -> list[dict]:
    if token_type == CommunityTokenType.ERC20:
        return [
            {
                "type": CommunityTokenType.ERC20.value,
                "contract_addresses": {network_id: contract_address},
                "symbol": "ADM",
                "amountInWei": str(10**18),
                "decimals": 18,
            }
        ]
    return [
        {
            "type": CommunityTokenType.ERC721.value,
            "contract_addresses": {network_id: contract_address},
            "symbol": "ADC",
            "amountInWei": "1",
            "decimals": 0,
        }
    ]


def verify_member_holds_admin_token(
    foundry_client,
    token_type: CommunityTokenType,
    contract_address: str,
    wallet_address: str,
):
    if token_type == CommunityTokenType.ERC20:
        community_tokens.verify_token_balance(
            foundry_client,
            CommunityTokenType.ERC20,
            contract_address,
            wallet_address,
            min_balance=10**18,
        )
        return

    owner_result = foundry_client.get_erc721_owner(contract_address, 0)
    assert owner_result.exit_code == 0, "ERC721 ownerOf check failed"
    raw_owner = owner_result.output.decode().strip()
    token_owner = Web3.to_checksum_address(f"0x{raw_owner[-40:]}")
    assert (
        token_owner.lower() == Web3.to_checksum_address(wallet_address).lower()
    ), f"Member should hold ERC721 token id 0, owner={token_owner}, expected={wallet_address}"


def get_community_token_contract_address(
    backend: StatusBackend,
    community_id: str,
    privileges_level: int,
    attempts: int = 10,
    delay: int = 2,
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


def read_backend_token_addresses_by_privilege(backend: StatusBackend, community_id: str) -> tuple[list[str], list[str]]:
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
            if perm_type not in (
                CommunityTokenPermissionType.BECOME_TOKEN_OWNER.value,
                CommunityTokenPermissionType.BECOME_TOKEN_MASTER.value,
            ):
                continue
            level = (
                CommunityTokenPrivilegesLevel.OWNER_LEVEL.value
                if perm_type == CommunityTokenPermissionType.BECOME_TOKEN_OWNER.value
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
    backend: StatusBackend,
    community_id: str,
    deploy_tx_hash: str,
    attempts: int = 30,
    delay: int = 2,
) -> tuple[str, str, str, str, bool, bool]:
    temp_owner_token_address = temporary_owner_contract_address(deploy_tx_hash)
    temp_master_token_address = temporary_master_contract_address(deploy_tx_hash)

    saw_temp_owner = False
    saw_temp_master = False
    final_owner_token_address = None
    final_master_token_address = None
    retrack_method = "communitytokens_reTrackOwnerTokenDeploymentTransaction"
    safe_owner_method = "communitytokens_safeGetOwnerTokenAddress"
    chain_id = backend.network_id
    retrack_supported = True

    for attempt in range(1, attempts + 1):
        owner_addresses, master_addresses = read_backend_token_addresses_by_privilege(backend, community_id)

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
                    logger.info(f"[DIAGNOSTICS] {safe_owner_method} failed while waiting token transition " f"(attempt {attempt}/{attempts}): {exc}")

        if owner_for_retrack is not None and retrack_supported:
            try:
                backend.rpc_valid_request(retrack_method, [chain_id, owner_for_retrack])
            except Exception as exc:
                error_text = str(exc)
                method_missing = ("-32601" in error_text) or ("does not exist/is not available" in error_text.lower())
                if method_missing:
                    retrack_supported = False
                    logger.info(f"[DIAGNOSTICS] {retrack_method} is unavailable in this backend; " "disabling retrack attempts for this wait cycle")
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


def wait_until_master_token_is_router_usable(
    backend: StatusBackend,
    community_id: str,
    master_token_address: str,
    sender_wallet_address: str,
    state,
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
    deploy_tx_hash = state.deploy_tx_hash
    temp_master_token_address = temporary_master_contract_address(deploy_tx_hash) if isinstance(deploy_tx_hash, str) and deploy_tx_hash else None
    temp_owner_token_address = temporary_owner_contract_address(deploy_tx_hash) if isinstance(deploy_tx_hash, str) and deploy_tx_hash else None

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
        logger.debug(f"Mint route suggestion context | {json.dumps(route_debug_context, default=str)}")

        try:
            routes_result = backend.wallet_service.suggested_community_routes(
                uuid=transaction_uuid,
                send_type=WalletSendType.COMMUNITY_MINT_TOKENS,
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

            if owner_token_is_valid:
                try:
                    backend.rpc_valid_request(retrack_method, [chain_id, owner_token_address])
                except Exception as retrack_exc:
                    logger.warning(f"{retrack_method} failed for owner token {owner_token_address} on chain {chain_id}: {retrack_exc}")

            if attempt == 1 or attempt == attempts or attempt % 5 == 0:
                logger.info(
                    f"Master token {router_master_token_address} not router-usable yet " f"(attempt {attempt}/{attempts}); last_error={error_text}"
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


@dataclass
class OwnerMasterTokens:
    deploy_tx_hash: str
    owner_token_address: str
    master_token_address: str


def deploy_owner_and_master_tokens(
    owner_backend,
    community_id: str,
    community_token_deployer: str,
    state: CommunityTokenDeployState,
    anvil_client=None,
    wait_for_deploy_status_signal: bool = False,
) -> OwnerMasterTokens:
    """Deploy owner/master tokens and wait until backend resolves final addresses."""
    deploy_tx_hash = deploy_owner_token(
        owner_backend,
        community_id,
        community_token_deployer,
        state,
        anvil_client=anvil_client,
        wait_for_deploy_status_signal=wait_for_deploy_status_signal,
    )
    assert deploy_tx_hash and re.fullmatch(r"0x[a-fA-F0-9]{64}", deploy_tx_hash), f"Invalid deploy tx hash: {deploy_tx_hash}"
    state.deploy_tx_hash = deploy_tx_hash

    (
        _temp_owner,
        _temp_master,
        owner_token_address,
        master_token_address,
        _saw_temp_owner,
        _saw_temp_master,
    ) = wait_for_backend_token_state_transition(
        backend=owner_backend,
        community_id=community_id,
        deploy_tx_hash=deploy_tx_hash,
        attempts=30,
        delay=2,
    )
    return OwnerMasterTokens(
        deploy_tx_hash=deploy_tx_hash,
        owner_token_address=owner_token_address,
        master_token_address=master_token_address,
    )


def mint_master_to_member(
    owner_backend,
    community_id: str,
    master_token_address: str,
    member_wallet: str,
    owner_token_address: str,
    state: CommunityTokenDeployState,
    anvil_client=None,
):
    """Wait for router readiness and mint one master token to the member wallet."""
    owner_wallet = messenger.wallet_address(owner_backend)
    wait_until_master_token_is_router_usable(
        backend=owner_backend,
        community_id=community_id,
        master_token_address=master_token_address,
        sender_wallet_address=owner_wallet,
        state=state,
        owner_token_address=owner_token_address,
        recipient_wallet_addresses=[member_wallet],
        attempts=30,
        delay=2,
    )
    with owner_backend.expect_signal(
        SignalType.COMMUNITY_TOKEN_TRANSACTION_STATUS_CHANGED,
        predicate=lambda s: is_community_token_tx_success(s, WalletSendType.COMMUNITY_MINT_TOKENS),
        timeout=60,
    ):
        mint_community_token(
            sender_backend=owner_backend,
            community_id=community_id,
            token_contract_address=master_token_address,
            wallet_addresses=[member_wallet],
            token_type=CommunityTokenType.ERC721,
            privilege_level=CommunityTokenPrivilegesLevel.MASTER_LEVEL.value,
            amount=1,
        )


def _refresh_admin_token_wallet_state(
    owner_backend,
    member_backend,
    member_wallet: str,
    admin_token_address: str,
):
    admin_token_key = f"{owner_backend.network_id}-{admin_token_address.lower()}"
    for backend in (member_backend, owner_backend):
        backend.wallet_service.restart_wallet_reload_timer()
        backend.wallet_service.fetch_or_get_cached_wallet_balances([member_wallet], True)
    # Owner wallet indexes community tokens; member may not know the contract yet.
    owner_backend.wallet_service.get_balances_at_by_chain([member_wallet], [admin_token_key])


def setup_admin_via_token(
    owner_backend,
    member_backend,
    community_id: str,
    member_wallet: str,
    token_type: CommunityTokenType,
    community_token_deployer: str,
    foundry_client,
    anvil_client,
    owner_token_address: str,
    master_token_address: str,
):
    """Deploy admin permission token, airdrop to member, and wait until member becomes admin."""
    admin_token_address = deploy_community_admin_permission_token(
        owner_backend,
        community_id,
        owner_token_address=owner_token_address,
        master_token_address=master_token_address,
        token_type=token_type,
        community_token_deployer=community_token_deployer,
        foundry_client=foundry_client,
        anvil_client=anvil_client,
    )

    owner_wallet = messenger.wallet_address(owner_backend)
    wait_until_community_admin_token_is_mint_usable(
        backend=owner_backend,
        community_id=community_id,
        token_contract_address=admin_token_address,
        sender_wallet_address=owner_wallet,
        recipient_wallet_addresses=[member_wallet],
        token_type=token_type,
    )
    airdrop_community_admin_token(
        owner_backend,
        community_id,
        admin_token_address,
        wallet_addresses=[member_wallet],
        token_type=token_type,
    )
    verify_member_holds_admin_token(foundry_client, token_type, admin_token_address, member_wallet)

    admin_token_criteria = admin_permission_token_criteria(token_type, owner_backend.network_id, admin_token_address)
    owner_backend.wakuext_service.create_community_token_permission(
        community_id=community_id,
        permission_type=CommunityTokenPermissionType.BECOME_ADMIN,
        token_criteria=admin_token_criteria,
    )

    expected_perm_types = {
        CommunityTokenPermissionType.BECOME_ADMIN.value,
        CommunityTokenPermissionType.BECOME_TOKEN_MASTER.value,
        CommunityTokenPermissionType.BECOME_TOKEN_OWNER.value,
    }
    community_tokens.wait_until_community_has_permission_types(owner_backend, community_id, expected_perm_types, attempts=15, delay=2)

    member_role_attempts = 45 if token_type == CommunityTokenType.ERC721 else 30
    last_error = None
    for round_idx in range(6):
        try:
            messenger.spectate_and_fetch_community(member_backend, community_id)
        except Exception as exc:
            logger.debug(f"spectate/fetch before admin role wait failed (round {round_idx + 1}): {exc}")
        _refresh_admin_token_wallet_state(owner_backend, member_backend, member_wallet, admin_token_address)

        community_tokens.sync_community_member_permissions(owner_backend, community_id)
        try:
            community_tokens.wait_for_member_role(
                owner_backend,
                community_id,
                member_backend.public_key,
                CommunityRoles.ROLE_ADMIN.value,
                attempts=20,
                delay=2,
                fetch_from_store=True,
            )
            community_tokens.wait_for_member_role(
                member_backend,
                community_id,
                member_backend.public_key,
                CommunityRoles.ROLE_ADMIN.value,
                attempts=member_role_attempts,
                delay=2,
                fetch_from_store=True,
                spectate=True,
            )
            return admin_token_address
        except AssertionError as exc:
            last_error = exc
            logger.info(f"Admin role not visible yet after round {round_idx + 1}/6: {exc}")
            time.sleep(3)

    raise AssertionError(f"Member {member_backend.public_key} did not become admin via {token_type.name}: {last_error}")


def temporary_master_contract_address(tx_hash: str) -> str:
    return f"{tx_hash}-master"


def temporary_owner_contract_address(tx_hash: str) -> str:
    return f"{tx_hash}-owner"


def _extract_master_token_address_from_event(event: dict) -> Optional[str]:
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


def resolve_master_token_address_from_receipt(anvil_client, community_token_deployer, tx_hash: Optional[str]) -> Optional[str]:
    """Resolve deployed master token address directly from deploy tx receipt logs."""
    _, master_token_address = resolve_owner_and_master_token_addresses_from_receipt(anvil_client, community_token_deployer, tx_hash)
    return master_token_address


def resolve_owner_and_master_token_addresses_from_receipt(
    anvil_client, community_token_deployer, tx_hash: Optional[str]
) -> Tuple[Optional[str], Optional[str]]:
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
    if not receipt_status_is_success(status):
        logger.warning(f"Receipt for tx {tx_hash} is not successful yet (status={status})")
        return None, None

    expected_owner_topic0 = Web3.keccak(text="DeployOwnerToken(address)").hex().lower()
    expected_master_topic0 = Web3.keccak(text="DeployMasterToken(address)").hex().lower()
    expected_emitter = str(community_token_deployer).lower()

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


def _predict_contract_address(foundry_client, sender: str, nonce: int) -> str:
    sender = Web3.to_checksum_address(sender)
    result = foundry_client.container.exec_run(f"cast compute-address {sender} --nonce {nonce}")
    assert result.exit_code == 0, f"cast compute-address failed: {result.output.decode().strip()}"
    match = re.search(r"0x[a-fA-F0-9]{40}", result.output.decode())
    assert match, f"Contract address not found in cast output: {result.output.decode().strip()}"
    return Web3.to_checksum_address(match.group(0))


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


def _sign_and_send_community_route(backend: StatusBackend, transaction_uuid: str, address_from: str) -> tuple[dict, list[str]]:
    with backend.expect_signal(
        SignalType.WALLET_ROUTER_SIGN_TRANSACTIONS,
        predicate=lambda s: s.get("event", {}).get("sendDetails", {}).get("uuid") == transaction_uuid,
        timeout=60,
    ) as sign_exp:
        backend.wallet_service.build_transactions_from_route(transaction_uuid)

    sign_signal = sign_exp.result
    assert isinstance(sign_signal, dict), f"Unexpected sign signal payload: {sign_signal}"
    signing_details = sign_signal["event"]["signingDetails"]
    signatures = {}
    for tx_hash in signing_details["hashes"]:
        sig_hex = backend.wallet_service.sign_message(tx_hash, address_from, backend.password)
        assert sig_hex and sig_hex.startswith("0x"), f"Invalid transaction signature for hash {tx_hash}: {sig_hex}"
        tx_signature = sig_hex[2:]
        signatures[tx_hash] = {
            "r": tx_signature[:64],
            "s": tx_signature[64:128],
            "v": tx_signature[128:],
        }

    with backend.expect_signal(
        SignalType.WALLET_ROUTER_TRANSACTIONS_SENT,
        predicate=lambda s: s.get("event", {}).get("uuid") == transaction_uuid
        or s.get("event", {}).get("sendDetails", {}).get("uuid") == transaction_uuid,
        timeout=60,
    ) as sent_exp:
        backend.wallet_service.send_router_transactions_with_signatures(transaction_uuid, signatures)

    sent_signal = sent_exp.result
    assert isinstance(sent_signal, dict), f"Unexpected sent signal payload: {sent_signal}"
    return sent_signal, _extract_tx_hashes(sent_signal)
