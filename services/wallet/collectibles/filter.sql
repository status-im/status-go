WITH filter_conditions AS (
	SELECT
		? AS filterCommunityTypeAll,
		? AS filterCommunityTypeOnlyNonCommunity,
		? AS filterCommunityTypeOnlyCommunity,
		? AS communityIDFilterDisabled,
		? AS communityPrivilegesLevelDisabled
)
SELECT
	ownership.chain_id,
	ownership.contract_address,
	ownership.token_id
FROM
	collectibles_ownership_cache ownership
	LEFT JOIN collectible_data_cache data
	ON (
		ownership.chain_id = data.chain_id
		AND ownership.contract_address = data.contract_address
		AND ownership.token_id = data.token_id
	)
	CROSS JOIN filter_conditions
WHERE
	ownership.chain_id IN (?)
	AND ownership.owner_address IN (?)
	AND (
		filterCommunityTypeAll
		OR (
			filterCommunityTypeOnlyNonCommunity
			AND data.community_id = ""
		)
		OR (
			filterCommunityTypeOnlyCommunity
			AND data.community_id <> ""
		)
	)
	AND (
		communityIDFilterDisabled
		OR (
			data.community_id IN (?)
		)
	)
	AND (
		communityPrivilegesLevelDisabled
		OR (
			data.community_privileges_level IN (?)
		)
	)
LIMIT
	? OFFSET ?