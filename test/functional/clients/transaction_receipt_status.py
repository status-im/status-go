"""Ethereum transaction receipt status (EIP-658, post-Byzantium)."""

from enum import IntEnum


class TransactionReceiptStatus(IntEnum):
    REVERTED = 0
    SUCCESS = 1


def receipt_status_is_success(status) -> bool:
    """Return True when *status* from a tx receipt indicates success."""
    if status is True:
        return True
    if isinstance(status, str):
        return status.lower() in ("0x1", "1")
    try:
        return TransactionReceiptStatus(int(status)) == TransactionReceiptStatus.SUCCESS
    except (TypeError, ValueError):
        return False
