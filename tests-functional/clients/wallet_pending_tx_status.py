"""Wallet pending transaction status — mirrors services/wallet/activity/common/types.go TxStatus."""

from enum import Enum


class WalletPendingTxStatus(str, Enum):
    PENDING = "Pending"
    SUCCESS = "Success"
    FAILED = "Failed"
