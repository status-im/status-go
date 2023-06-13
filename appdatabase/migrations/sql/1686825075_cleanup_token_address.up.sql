-- Fix the token_address column in the transfers table that has been
-- incorrectly set to a string of the form "0x12ab..34cd" instead of a byte array
UPDATE transfers SET token_address = replace(replace(token_address, '0x', ''), '"', '') WHERE token_address LIKE '"0x%"';
