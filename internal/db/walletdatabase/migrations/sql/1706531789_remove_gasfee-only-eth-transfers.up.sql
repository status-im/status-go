--- Select all token transfers
WITH token_rows AS (
    SELECT ROWID, *
    FROM transfers
    WHERE type != 'eth'
)
--- Select all 0-value transfers
, eth_zero_value_rows AS (
    SELECT ROWID, *
    FROM transfers
    WHERE type = 'eth' AND amount_padded128hex = '00000000000000000000000000000000'
)
-- Select gas-fee-only ETH transfers
, eth_gas_fee_only_rows AS (
    SELECT ROWID
    FROM eth_zero_value_rows
    WHERE (tx_hash, address, network_id, account_nonce) IN (
        SELECT tx_hash, address, network_id, account_nonce
        FROM token_rows
    )
)

DELETE FROM transfers WHERE ROWID in eth_gas_fee_only_rows;
