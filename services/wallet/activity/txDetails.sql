-- Query searches for additional details of a transaction

SELECT
	tx_hash,
	blk_number,
	network_id,
	account_nonce,
	tx,
	contract_address,
	base_gas_fee
FROM
	transfers
WHERE
	hash = ?