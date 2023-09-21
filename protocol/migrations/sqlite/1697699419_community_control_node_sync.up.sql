CREATE TABLE communities_control_node (
    community_id BLOB NOT NULL PRIMARY KEY ON CONFLICT REPLACE,
    clock INT NOT NULL,
    installation_id VARCHAR NOT NULL
);

INSERT INTO
    communities_control_node (community_id, clock, installation_id)
SELECT
    c.id,
    1 AS clock,
    s.installation_id
FROM
    communities_communities AS c
    JOIN shhext_config AS s
WHERE
    c.private_key IS NOT NULL
    AND c.private_key != '';
