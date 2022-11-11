INSERT INTO cluster_nodes(node, type)
SELECT node, type 
FROM cluster_nodes
WHERE type = "relay"
