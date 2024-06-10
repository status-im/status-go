-- Query searches for additional details of a multi transaction
-- Pending transactions are saved as multi transactions. Some of data isn't available in multi transaction table, so we need to query pending transactions table to get it.

-- Tx property is only exctracted when values are not null to prevent errors during the scan.
SELECT
	transfers.tx_hash AS tx_hash,
	transfers.blk_number AS blk_number,
	transfers.network_id AS network_id,
	transfers.type AS type,
	transfers.account_nonce as nonce,
	transfers.contract_address as contract_address,
	CASE
		WHEN json_extract(transfers.tx, '$.gas') = '0x0' THEN NULL
		ELSE transfers.tx
	END as tx,
	transfers.base_gas_fee AS base_gas_fee
FROM
	transfers
WHERE
	multi_transaction_id = ?
UNION
ALL
SELECT
	pt.hash as tx_hash,
	NULL AS blk_number,
	pt.network_id as network_id,
	NULL as type,
	pt.nonce as nonce,
	NULL as contract_address,
	NULL as tx,
	NULL as base_gas_fee
FROM
	pending_transactions as pt
WHERE
	pt.multi_transaction_id = ?